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
	"github.com/rjeczalik/gpf/util"
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
	return fmt.Sprintf("service %q (id=%u, path=%s, version=%s), listening on localhost:%s",
		p.id, p.service, p.cmd.Path, p.version, p.port)
}

var errRegisterReq = errors.New(`router: register request ill-formed`)
var errPluginVersion = errors.New(`router: plugin version empty`)
var errTimeout = errors.New(`router: awaiting registration to complete has timed out`)

type routerAdmin struct {
	rt   *Router
	port string
	srv  *rpc.Server
}

func (ra *routerAdmin) serve(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go ra.srv.ServeConn(conn)
	}
}

func (ra *routerAdmin) Register(req RegisterRequest, _ *struct{}) (err error) {
	if !req.valid() {
		return errRegisterReq
	}
	ra.rt.mu.RLock()
	p, ok := ra.rt.pending[req.ID]
	ra.rt.mu.RUnlock()
	if !ok {
		err = fmt.Errorf("router: no plugin awaiting registration with ID=%s", req.ID)
		return
	}
	defer func() {
		if err == nil {
			ra.rt.valid <- p
		} else {
			ra.rt.mu.Lock()
			p.err = err
			ra.rt.mu.Unlock()
			ra.rt.invalid <- p
		}
	}()
	cli, port := (*rpc.Client)(nil), strconv.Itoa(int(req.Port))
	if cli, err = rpc.Dial("tcp", "localhost:"+port); err != nil {
		return
	}
	var version string
	if err = cli.Call(req.Service+".Init", "localhost:"+ra.port, &version); err != nil {
		return
	}
	ra.rt.mu.Lock()
	p.service, p.version, p.port = req.Service, version, port
	ra.rt.mu.Unlock()
	return
}

type Router struct {
	admin   *routerAdmin
	mu      sync.RWMutex
	plugins map[string]*plugin
	pending map[uint16]*plugin
	valid   chan *plugin
	invalid chan *plugin
	counter util.Counter
}

func (rt *Router) loadPlugins() (err error) {
	var dir string
	if dir, err = osext.ExecutableFolder(); err != nil {
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
			p.cmd.Stdin = bytes.NewReader([]byte(rt.admin.port + " " + strconv.Itoa(int(p.id))))
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

func (ra *routerAdmin) listenAndServe() (err error) {
	var l net.Listener
	if l, err = net.Listen("tcp", "localhost:0"); err != nil {
		return
	}
	if _, ra.port, err = net.SplitHostPort(l.Addr().String()); err != nil {
		return
	}
	ra.srv.RegisterName("Router", ra)
	go ra.rt.daemon()
	go ra.serve(l)
	if err = ra.rt.loadPlugins(); err != nil {
		return
	}
	log.Println("router admin service listening on localhost:"+ra.port, ". . .")
	return
}

func (rt *Router) ListenAndServe(addr string) (err error) {
	if err = rt.admin.listenAndServe(); err != nil {
		return
	}
	var l net.Listener
	if l, err = net.Listen("tcp", addr); err != nil {
		return
	}
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
	rt.admin = &routerAdmin{rt, "", rpc.NewServer()}
	return
}
