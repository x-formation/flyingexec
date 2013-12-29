package plugin

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"reflect"
	"strconv"
	"sync"

	"bitbucket.org/kardianos/osext"
)

const NonVersioned = "non-versioned"

type Plugin struct{}

func (p Plugin) Port(_, res *string) (err error) {
	defaultPluginSrv.mu.RLock()
	*res = defaultPluginSrv.port
	defaultPluginSrv.mu.RUnlock()
	return
}

func (p Plugin) Version(_, res *string) (err error) {
	*res = NonVersioned
	return
}

func (p Plugin) Init(_, _ *string) (err error) {
	return
}

type pluginSrv struct {
	port       string
	routerPort string
	mu         sync.RWMutex
	stdin      io.ReadWriter
	srv        *rpc.Server
}

var errRouterPort = errors.New(`plugin: router port ill-formed`)

var defaultPluginSrv = &pluginSrv{
	stdin: os.Stdin,
	srv:   rpc.NewServer(),
}

func (p *pluginSrv) readRouterPort() (port string, err error) {
	v := make(map[string]string)
	p.mu.RLock()
	err = json.NewDecoder(p.stdin).Decode(&v)
	p.mu.RUnlock()
	if err != nil {
		return
	}
	port, ok := v["port"]
	if !ok {
		err = errRouterPort
		return
	}
	if _, err = strconv.ParseUint(port, 10, 16); err != nil {
		err = errRouterPort
		return
	}
	return
}

func (p *pluginSrv) listenAndServe(rcrv interface{}) (err error) {
	var routerPort string
	if routerPort, err = p.readRouterPort(); err != nil {
		return
	}
	var l net.Listener
	if l, err = net.Listen("tcp", "localhost:0"); err != nil {
		return
	}
	defer l.Close()
	go p.serve(l)
	var port string
	if _, port, err = net.SplitHostPort(l.Addr().String()); err != nil {
		return
	}
	var path string
	if path, err = osext.Executable(); err != nil {
		return
	}
	p.mu.Lock()
	p.srv.Register(rcrv)
	p.port = port
	p.mu.Unlock()
	var r *rpc.Client
	if r, err = rpc.Dial("tcp", "localhost:"+routerPort); err != nil {
		return
	}
	req := map[string]string{
		"service": reflect.TypeOf(rcrv).Elem().Name(),
		"port":    port,
		"path":    path,
	}
	if err = r.Call("Router.Register", req, nil); err != nil {
		return
	}
	req = make(map[string]string)
	select {}
}

func (p *pluginSrv) serve(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go p.srv.ServeConn(conn)
	}
}

func ListenAndServe(rcrv interface{}) error {
	return defaultPluginSrv.listenAndServe(rcrv)
}
