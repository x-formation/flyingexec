package wip

import (
	_ "bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	_ "io/ioutil"
	"net"
	"net/rpc"
	"reflect"
	"strings"
)

type Plugin struct{}

func (p Plugin) Init(configuration []byte, res *int) error {
	*res = 10
	return nil
}

func StartPlugin(rcrv interface{}) (string, error) {
	s := rpc.NewServer()
	s.Register(rcrv)

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", err
	}
	fmt.Println("plugin port:", port)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			go s.ServeConn(conn)
		}
	}()
	return port, nil
}

type router struct {
	port    string
	plugins map[string]string
}

func (rt *router) routeConn(conn io.ReadWriteCloser) {
	var buf bytes.Buffer
	var req rpc.Request
	dec := gob.NewDecoder(io.TeeReader(conn, &buf))
	for {
		var err error
		defer fmt.Println("routeConn:", err)
		defer buf.Reset()
		if err = dec.Decode(&req); err != nil {
			break
		}
		if err = dec.Decode(nil); err != nil {
			break
		}
		var port string
		dot := strings.LastIndex(req.ServiceMethod, ".")
		fmt.Println(req)
		if dot < 0 {
			err = errors.New("rpc: service/method request ill-formed: " + req.ServiceMethod)
		} else {
			var ok bool
			serviceName := req.ServiceMethod[:dot]
			port, ok = rt.plugins[serviceName]
			if !ok {
				err = errors.New("rps: can't find service " + req.ServiceMethod)
			}
		}
		if err != nil {
			break
		}
		fmt.Println("net.Dial")
		plugin, err := net.Dial("tcp", "localhost:"+port)
		if err != nil {
			break
		}
		defer plugin.Close()
		if _, err = plugin.Write(buf.Bytes()); err != nil {
			break
		}
		if _, err = io.Copy(conn, plugin); err != nil {
			break
		}
	}
	conn.Close()
}

func NewRouter() (*router, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return nil, err
	}
	fmt.Println("router port:", port)
	r := &router{
		port:    port,
		plugins: make(map[string]string),
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			go r.routeConn(conn)
		}
	}()
	return r, nil
}

func (pd *router) init(name, port string) error {
	pd.plugins[name] = port
	c, err := rpc.Dial("tcp", "localhost:"+port)
	if err != nil {
		return err
	}
	var cfg []byte
	var res int
	err = c.Call(name+".Init", cfg, &res)
	if err != nil {
		return err
	}
	return nil
}

func (pd *router) Start(plugins ...interface{}) error {
	for _, p := range plugins {
		port, err := StartPlugin(p)
		if err != nil {
			return err
		}
		err = pd.init(reflect.TypeOf(p).Name(), port)
		if err != nil {
			return err
		}
	}
	return nil
}
