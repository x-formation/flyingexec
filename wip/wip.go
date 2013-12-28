package wip

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"reflect"
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

type plugind struct {
	port    string
	plugins map[string]string
}

func (p *plugind) ReadRequestHeader(req *rpc.Request) error {
	return nil
}

func (p *plugind) ReadRequestBody(body interface{}) error {
	return nil
}

func (p *plugind) WriteResponse(res *rpc.Response, body interface{}) error {
	return nil
}

func (p *plugind) Close() error {
	return nil
}

func NewPlugind() (*plugind, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return nil, err
	}
	pd := &plugind{
		port:    port,
		plugins: make(map[string]string),
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			go func() {
				for _, port := range pd.plugins {
					plugin, err := net.Dial("tcp", "localhost:"+port)
					if err != nil {
						fmt.Println("plugind error:", err)
						continue
					}
					go io.Copy(conn, plugin)
					go io.Copy(plugin, conn)
				}
			}()
		}
	}()
	return pd, nil
}

func (pd *plugind) init(name, port string) error {
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

func (pd *plugind) Start(plugins ...interface{}) error {
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
