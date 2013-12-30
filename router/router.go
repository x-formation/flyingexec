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
	"runtime"
	"strings"
	"sync"
	"time"

	"bitbucket.org/kardianos/osext"
)

type plugin struct {
	service string
	version string
	port    string
	cmd     *exec.Cmd
	buf     *bytes.Buffer
	client  *rpc.Client
	err     error
}

func (p *plugin) String() string {
	return fmt.Sprintf("service %q (version=%s, path=%s), listening on localhost:%s",
		p.service, p.version, p.cmd.Path, p.port)
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

func (ra *routerAdmin) Register(req *map[string]string, _ *int) (err error) {
	for _, k := range []string{"service", "port", "path"} {
		if v, ok := (*req)[k]; !ok || len(v) == 0 {
			err = errRegisterReq
			return
		}
	}
	ra.rt.mu.RLock()
	p, ok := ra.rt.pending[(*req)["path"]]
	ra.rt.mu.RUnlock()
	if !ok {
		err = fmt.Errorf("router: no plugin awaiting registration for path %s", (*req)["path"])
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
	var client *rpc.Client
	if client, err = rpc.Dial("tcp", "localhost:"+(*req)["port"]); err != nil {
		return
	}
	var port string
	if err = client.Call((*req)["service"]+".Port", "", &port); err != nil {
		return
	}
	if port != (*req)["port"] {
		err = fmt.Errorf("router: ports do not match: Router.Register=%s and %s.Port=%s",
			(*req)["port"], (*req)["service"], port)
		return
	}
	var version string
	if err = client.Call((*req)["service"]+".Version", "", &version); err != nil {
		return
	}
	if len(version) == 0 {
		err = errPluginVersion
		return
	}
	ra.rt.mu.Lock()
	p.service = (*req)["service"]
	p.version = version
	p.port = port
	p.client = client
	ra.rt.mu.Unlock()
	return
}

type Router struct {
	admin   *routerAdmin
	mu      sync.RWMutex
	plugins map[string]*plugin
	pending map[string]*plugin
	valid   chan *plugin
	invalid chan *plugin
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
	var portJSON = []byte(`{"port":"` + rt.admin.port + `"}` + "\r\n")
	for _, p := range plugins {
		var err error
		var f io.Writer
		logFile, buf := filepath.Join(logDir, p.Name()+".log"), new(bytes.Buffer)
		var output io.Writer = buf
		cmd := exec.Command(filepath.Join(pluginDir, p.Name()))
		if f, err = os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err == nil {
			output = io.MultiWriter(buf, f)
		}
		cmd.Stdout = output
		cmd.Stderr = output
		cmd.Stdin = bytes.NewReader(portJSON)
		go rt.run(&plugin{"", "", "", cmd, buf, nil, err})
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
	rt.pending[p.cmd.Path] = p
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
	delete(rt.pending, p.cmd.Path)
	rt.mu.Unlock()
	defer func() {
		p.client.Close()
		p.client = nil
	}()
	var res string
	if err := p.client.Call(p.service+".Init", "", &res); err != nil {
		log.Printf("error initializing plugin %q: error=%q, combined output=%q",
			p.service, err.Error(), string(p.buf.Bytes()))
		rt.remove(p)
		return
	}
	log.Printf("plugin successfully added: %s", p)
}

func (rt *Router) remove(p *plugin) {
	rt.mu.Lock()
	err := p.err
	delete(rt.pending, p.cmd.Path)
	rt.mu.Unlock()
	if len(p.service) > 0 && err == nil {
		rt.mu.Lock()
		delete(rt.plugins, p.service)
		rt.mu.Unlock()
		log.Printf("plugin successfully removed: %s", p)
	} else {
		rt.mu.RLock()
		output, path, err := string(p.buf.Bytes()), p.cmd.Path, p.err.Error()
		rt.mu.RUnlock()
		log.Printf("error running plugin %s: error=%q, combined output=%q", path, err, output)
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
		pending: make(map[string]*plugin),
		valid:   make(chan *plugin),
		invalid: make(chan *plugin),
	}
	rt.admin = &routerAdmin{rt, "", rpc.NewServer()}
	return
}
