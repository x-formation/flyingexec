package router

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"bitbucket.org/kardianos/osext"
)

type plugin struct {
	port    string
	version string
	cmd     *exec.Cmd
	buf     *bytes.Buffer
	err     error
}

type kv struct {
	k string
	v *plugin
}

var errRegisterReq = errors.New(`router: register request ill-formed`)
var errPluginVersion = errors.New(`router: plugin version empty`)
var errTimeout = errors.New(`router: awaiting registration to complete has timed out`)

type Router struct {
	mu           sync.RWMutex
	internalPort string
	internal     *rpc.Server
	plugins      map[string]*plugin
	pending      map[string]*plugin
	valid        chan kv
	invalid      chan *plugin
}

func (rt *Router) Register(req *map[string]string, _ *int) (err error) {
	for _, k := range []string{"service", "port", "path"} {
		if v, ok := (*req)[k]; !ok || len(v) == 0 {
			err = errRegisterReq
			return
		}
	}
	rt.mu.RLock()
	p, ok := rt.pending[(*req)["path"]]
	rt.mu.RUnlock()
	if !ok {
		err = fmt.Errorf("router: no plugin awaiting registration for path %q", (*req)["path"])
		return
	}
	defer func() {
		if err == nil {
			rt.valid <- kv{(*req)["service"], p}
		} else {
			rt.mu.Lock()
			p.err = err
			rt.mu.Unlock()
			rt.invalid <- p
		}
	}()
	var client *rpc.Client
	if client, err = rpc.Dial("tcp", "localhost:"+(*req)["port"]); err != nil {
		return
	}
	defer client.Close()
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
	rt.mu.Lock()
	p.port = port
	p.version = version
	rt.mu.Unlock()
	return
}

func (rt *Router) loadPlugins() (err error) {
	var dir string
	if dir, err = osext.ExecutableFolder(); err != nil {
		return
	}
	dir = filepath.Join(dir, "plugins")
	var plugins []os.FileInfo
	if plugins, err = ioutil.ReadDir(dir); err != nil {
		return
	}
	var portJSON = []byte(`{"port":"` + rt.internalPort + `"}` + "\r\n")
	for _, p := range plugins {
		buf := new(bytes.Buffer)
		cmd := exec.Command(filepath.Join(dir, p.Name()))
		cmd.Stdin = bytes.NewReader(portJSON)
		cmd.Stdout = buf
		cmd.Stderr = buf
		go rt.run(&plugin{"", "", cmd, buf, nil})
	}
	return
}

func (rt *Router) daemon() {
	for {
		select {
		case p := <-rt.invalid:
			rt.remove("", p)
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
	case kv := <-rt.valid:
		rt.add(kv.k, kv.v)
	case <-time.After(30 * time.Second):
		rt.mu.Lock()
		p.err = errTimeout
		rt.mu.Unlock()
		rt.invalid <- p
		return
	}
	p.err = p.cmd.Wait()
}

func (rt *Router) add(service string, p *plugin) {
	rt.mu.Lock()
	rt.plugins[service] = p
	port, version := p.port, p.version
	delete(rt.pending, p.cmd.Path)
	rt.mu.Unlock()
	log.Printf("service successfully added: %s (%s) on localhost:%s",
		service, version, port)
}

func (rt *Router) remove(service string, p *plugin) {
	rt.mu.Lock()
	err := p.err
	delete(rt.pending, p.cmd.Path)
	rt.mu.Unlock()
	if len(service) > 0 && err == nil {
		rt.mu.Lock()
		port, version := p.port, p.version
		delete(rt.plugins, service)
		rt.mu.Unlock()
		log.Printf("service successfully removed: %s (%s) on localhost:%s",
			service, version, port)
	} else {
		rt.mu.RLock()
		output, path, err := string(p.buf.Bytes()), p.cmd.Path, p.err.Error()
		rt.mu.RUnlock()
		log.Printf("error running %s: error=%q, combined output=%q", path, err, output)
	}
}

func (rt *Router) internalServe(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go rt.internal.ServeConn(conn)
	}
}

func NewRouter() (rt *Router, err error) {
	rt = &Router{
		internal: rpc.NewServer(),
		plugins:  make(map[string]*plugin),
		pending:  make(map[string]*plugin),
		valid:    make(chan kv),
		invalid:  make(chan *plugin),
	}
	var l net.Listener
	if l, err = net.Listen("tcp", "localhost:0"); err != nil {
		return
	}
	if _, rt.internalPort, err = net.SplitHostPort(l.Addr().String()); err != nil {
		return
	}
	rt.internal.Register(rt)
	go rt.daemon()
	go rt.internalServe(l)
	if err = rt.loadPlugins(); err != nil {
		return
	}
	log.Println("router internal service listening on localhost:"+rt.internalPort, ". . .")
	return
}
