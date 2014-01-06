package router

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.org/kardianos/osext"
	"github.com/rjeczalik/flyingexec/util"
)

type RegisterRequest struct {
	ID      uint16
	Service string
	Port    uint16
}

func (req *RegisterRequest) valid() bool {
	return req.ID > 0 && len(req.Service) > 0 && req.Port > 0
}

type plugin struct {
	id      uint16
	service string
	version string
	port    string
	cmd     *exec.Cmd
	log     *os.File
	err     error
}

func (p *plugin) String() string {
	cmd, log := "<invalid>", "<invalid>"
	if p.cmd != nil {
		cmd = p.cmd.Path
	}
	if p.log != nil {
		log = p.log.Name()
	}
	return fmt.Sprintf("service %q (id=%d, path=%s, log=%s, version=%s), listening on localhost:%s",
		p.service, p.id, cmd, log, p.version, p.port)
}

var errRegisterReq = errors.New(`router: register request ill-formed`)
var errPluginVersion = errors.New(`router: plugin version empty`)
var errTimeout = errors.New(`router: awaiting registration to complete has timed out`)

type Admin struct {
	rt       *Router
	Listener net.Listener
	Dialer   util.Dialer
}

func (a *Admin) serve() {
	srv := rpc.NewServer()
	srv.RegisterName("Router", a)
	log.Println("router admin service listening on", a.Listener.Addr().String(), ". . .")
	for {
		conn, err := a.Listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go srv.ServeConn(conn)
	}
}

func (a *Admin) listenAndServe() (err error) {
	if a.Listener, err = net.Listen("tcp", "localhost:0"); err != nil {
		return
	}
	go a.serve()
	if err = a.rt.loadPlugins(); err != nil {
		return
	}
	return
}

func (a *Admin) Register(req RegisterRequest, _ *struct{}) (err error) {
	if !req.valid() {
		return errRegisterReq
	}
	a.rt.mu.RLock()
	p, ok := a.rt.pending[req.ID]
	a.rt.mu.RUnlock()
	if !ok {
		err = fmt.Errorf("router: no plugin awaiting registration with ID=%s", req.ID)
		return
	}
	defer func() {
		if err == nil {
			a.rt.valid <- p
		} else {
			a.rt.mu.Lock()
			p.err = err
			a.rt.mu.Unlock()
			a.rt.invalid <- p
		}
	}()
	var port = strconv.Itoa(int(req.Port))
	var conn io.ReadWriteCloser
	if conn, err = a.Dialer.Dial("tcp", "localhost:"+port); err != nil {
		return
	}
	var version string
	var cli = rpc.NewClient(conn)
	defer cli.Close()
	if err = cli.Call(req.Service+".Init", a.Listener.Addr().String(), &version); err != nil {
		return
	}
	a.rt.mu.Lock()
	p.service, p.version, p.port = req.Service, version, port
	a.rt.mu.Unlock()
	return
}

type Router struct {
	admin   *Admin
	mu      sync.RWMutex
	plugins map[string]*plugin
	pending map[uint16]*plugin
	valid   chan *plugin
	invalid chan *plugin
	counter util.Counter
}

func (rt *Router) loadPlugins() (err error) {
	var dir, port string
	if dir, err = osext.ExecutableFolder(); err != nil {
		return
	}
	if _, port, err = net.SplitHostPort(rt.admin.Listener.Addr().String()); err != nil {
		return
	}
	pluginDir := filepath.Join(dir, "plugins")
	logDir := filepath.Join(dir, "logs")
	_ = os.MkdirAll(pluginDir, 0775)
	_ = os.MkdirAll(logDir, 0775)
	var plugins []os.FileInfo
	if plugins, err = ioutil.ReadDir(pluginDir); err != nil {
		return
	}
	for _, p := range plugins {
		if !p.IsDir() {
			logFile := filepath.Join(logDir, p.Name()+".log")
			p := &plugin{
				id:  uint16(rt.counter.Next()),
				cmd: exec.Command(filepath.Join(pluginDir, p.Name())),
			}
			if p.log, p.err = os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); p.err == nil {
				p.cmd.Stdout, p.cmd.Stderr = p.log, p.log
			}
			p.cmd.Stdin = bytes.NewReader([]byte(port + " " + strconv.Itoa(int(p.id))))
			go rt.run(p)
		}
	}
	return
}

func (rt *Router) daemon() {
	for {
		select {
		case p := <-rt.invalid:
			rt.remove(p)
		}
	}
}

func (rt *Router) run(p *plugin) {
	rt.mu.Lock()
	rt.pending[p.id] = p
	rt.mu.Unlock()
	if err := p.cmd.Start(); err != nil {
		p.err = err
		rt.invalid <- p
		return
	}
	select {
	case p := <-rt.valid:
		rt.add(p)
	case <-time.After(30 * time.Second):
		rt.mu.Lock()
		p.err = errTimeout
		rt.mu.Unlock()
		rt.invalid <- p
		return
	}
	p.err = p.cmd.Wait()
}

func (rt *Router) add(p *plugin) {
	rt.mu.RLock()
	_, ok := rt.plugins[p.service]
	rt.mu.RUnlock()
	if ok {
		log.Printf("error adding plugin %q: service is already registered, removing %s", p.service, p.cmd.Path)
		rt.remove(p)
		return
	}
	rt.mu.Lock()
	rt.plugins[p.service] = p
	delete(rt.pending, p.id)
	rt.mu.Unlock()
	log.Printf("plugin successfully added: %s", p)
}

func (rt *Router) remove(p *plugin) {
	rt.mu.Lock()
	err := p.err
	delete(rt.pending, p.id)
	rt.mu.Unlock()
	if len(p.service) > 0 && err == nil {
		rt.mu.Lock()
		delete(rt.plugins, p.service)
		rt.mu.Unlock()
		log.Printf("plugin successfully removed: %s", p)
	} else {
		log.Printf("error running plugin: %s", p)
	}
}

func (rt *Router) routeConn(conn io.ReadWriteCloser) {
	var buf bytes.Buffer
	var req rpc.Request
	dec := gob.NewDecoder(io.TeeReader(conn, &buf))
	for {
		var err error
		defer buf.Reset()
		if err = dec.Decode(&req); err != nil {
			break
		}
		if err = dec.Decode(nil); err != nil {
			break
		}
		var port string
		dot := strings.LastIndex(req.ServiceMethod, ".")
		if dot < 0 {
			err = errors.New("rpc: service/method request ill-formed: " + req.ServiceMethod)
		} else {
			p, ok := rt.plugins[req.ServiceMethod[:dot]]
			if !ok {
				err = errors.New("rps: can't find service " + req.ServiceMethod)
			} else {
				port = p.port
			}
		}
		if err != nil {
			break // TODO: send a response back
		}
		route, err := net.Dial("tcp", "localhost:"+port)
		if err != nil {
			break // TODO: send a response back
		}
		defer route.Close()
		if _, err = route.Write(buf.Bytes()); err != nil {
			break
		}
		if _, err = io.Copy(conn, route); err != nil {
			break
		}
	}
	conn.Close()
}

func (rt *Router) ListenAndServe(addr string) (err error) {
	if err = rt.admin.listenAndServe(); err != nil {
		return
	}
	var l net.Listener
	if l, err = net.Listen("tcp", addr); err != nil {
		return
	}
	go rt.daemon()
	log.Println("router listening on", l.Addr().String(), ". . .")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Print("error serving connection:", err)
			continue
		}
		go rt.routeConn(conn)
	}
	return
}

func NewRouter() (rt *Router) {
	rt = &Router{
		plugins: make(map[string]*plugin),
		pending: make(map[uint16]*plugin),
		valid:   make(chan *plugin),
		invalid: make(chan *plugin),
	}
	rt.admin = &Admin{
		rt:       rt,
		Listener: nil,
		Dialer:   util.DefaultDialer,
	}
	return
}
