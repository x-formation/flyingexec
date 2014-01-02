package plugin

import (
	"bufio"
	"errors"
	"log"
	"net"
	"net/rpc"
	"os"
	"reflect"
	"strconv"
)

const NonVersioned = "non-versioned"

var errRead = errors.New("plugin: reading port and/or ID from stdin failed")

type Plugin interface {
	Init(routerAddr string, version *string) error
}

type CallCloser interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
	Close() error
}

type Dialer func(string, string) (CallCloser, error)

type Connector struct {
	ID         string
	RouterAddr string
	Listener   net.Listener
	Dial       Dialer
}

func (c *Connector) serve(p Plugin) {
	srv := rpc.NewServer()
	srv.Register(p)
	for {
		conn, err := c.Listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go srv.ServeConn(conn)
	}
}

func (c *Connector) register(p Plugin) (string, error) {
	_, port, err := net.SplitHostPort(c.Listener.Addr().String())
	if err != nil {
		return "", err
	}
	cli, err := c.Dial("tcp", c.RouterAddr)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	cfg := map[string]string{
		"id":      c.ID,
		"service": reflect.TypeOf(p).Elem().Name(),
		"addr":    "localhost" + port,
	}
	var version string
	err = cli.Call("Router.Register", cfg, &version)
	return version, err
}

func readUint16From(f *os.File) (string, error) {
	if fi, err := f.Stat(); err != nil || fi.Size() == 0 {
		return "", errRead
	}
	scan := bufio.NewScanner(f)
	scan.Split(bufio.ScanWords)
	if !scan.Scan() || scan.Err() != nil {
		return "", errRead
	}
	t := scan.Text()
	if _, err := strconv.ParseUint(t, 10, 16); err != nil {
		return "", err
	}
	return t, nil
}

func newConnector(f *os.File) (c *Connector, err error) {
	c = &Connector{
		Dial: func(network, address string) (CallCloser, error) {
			return rpc.Dial(network, address)
		},
	}
	if c.ID, err = readUint16From(os.Stdin); err != nil {
		return
	}
	if c.RouterAddr, err = readUint16From(os.Stdin); err != nil {
		return
	}
	c.RouterAddr = "localhost:" + c.RouterAddr
	c.Listener, err = net.Listen("tcp", "localhost:0")
	return
}

func ListenAndServe(p Plugin) error {
	c, err := newConnector(os.Stdin)
	if err != nil {
		return err
	}
	return Serve(c, p)
}

func Serve(c *Connector, p Plugin) error {
	go c.serve(p)
	defer c.Listener.Close()
	if _, err := c.register(p); err != nil {
		return err
	}
	select {}
}
