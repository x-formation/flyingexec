package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"reflect"
	"strconv"
	"sync"
)

type Plugin struct{}

func (p Plugin) Port(_, res *string) (err error) {
	defaultPluginSrv.mu.RLock()
	*res = defaultPluginSrv.port
	defaultPluginSrv.mu.RUnlock()
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

func (p *pluginSrv) readRouterPort() (err error) {
	v := make(map[string]string)
	p.mu.RLock()
	err = json.NewDecoder(p.stdin).Decode(&v)
	p.mu.RUnlock()
	if err != nil {
		return
	}
	routerPort, ok := v["port"]
	if !ok {
		return errRouterPort
	}
	if _, err = strconv.ParseUint(routerPort, 10, 16); err != nil {
		return errRouterPort
	}
	p.mu.Lock()
	p.routerPort = routerPort
	p.mu.Unlock()
	return
}

func (p *pluginSrv) listenAndServe(rcrv interface{}) (err error) {
	if err = p.readRouterPort(); err != nil {
		return
	}
	var l net.Listener
	if l, err = net.Listen("tcp", "localhost:0"); err != nil {
		return
	}
	defer l.Close()
	var port string
	if _, port, err = net.SplitHostPort(l.Addr().String()); err != nil {
		return
	}
	p.mu.Lock()
	p.srv.Register(rcrv)
	p.port = port
	p.mu.Unlock()
	var r *rpc.Client
	if r, err = rpc.Dial("tcp", "localhost:"+port); err != nil {
		return
	}
	go p.serve(l)
	req := map[string]string{
		"service": reflect.TypeOf(rcrv).Name(),
		"port":    port,
	}
	if err = r.Call("Router.Register", req, nil); err != nil {
		return
	}
	fmt.Println(req["service"], " serving on localhost:"+port, ". . .")
	select {}
}

func (p *pluginSrv) serve(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go p.srv.ServeConn(conn)
	}
}

func ListenAndServe(rcrv interface{}) error {
	return defaultPluginSrv.listenAndServe(rcrv)
}
