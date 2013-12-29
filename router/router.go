package router

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"os/exec"
)

type plugin struct {
	port string
	cmd  *exec.Cmd
}

type Router struct {
	internalPort string
	internal     *rpc.Server
	plugins      map[string]plugin
}

func (rt *Router) Register(req *map[string]string, _ *int) (err error) {
	fmt.Println(*req)
	return errors.New("NOT IMPLEMENTED")
}

func (rt *Router) loadPlugins() (err error) {
	return
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
		plugins:  make(map[string]plugin),
	}
	var l net.Listener
	if l, err = net.Listen("tcp", "localhost:0"); err != nil {
		return
	}
	if _, rt.internalPort, err = net.SplitHostPort(l.Addr().String()); err != nil {
		return
	}
	rt.internal.Register(rt)
	go rt.internalServe(l)
	fmt.Println("router serving on localhost:"+rt.internalPort, ". . .")
	return
}
