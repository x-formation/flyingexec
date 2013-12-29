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

type Router struct {
	mu           sync.RWMutex
	internalPort string
	internal     *rpc.Server
	plugins      map[string]*plugin
	pending      map[string]*plugin
	valid        chan *plugin
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
		err = fmt.Errorf("router: no plugin awaiting registration for path %s", (*req)["path"])
		return
	}
	defer func() {
		if err == nil {
			rt.valid <- p
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
	p.service = (*req)["service"]
	p.version = version
	p.port = port
	p.client = client
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
		go rt.run(&plugin{"", "", "", cmd, buf, nil, nil})
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
		log.Printf("error adding plugin: service %q is already registered, removing %s", p.service, p.cmd.Path)
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
		log.Printf("error initializing plugin: error=%q, combined output=%q",
			err.Error(), string(p.buf.Bytes()))
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
		valid:    make(chan *plugin),
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
