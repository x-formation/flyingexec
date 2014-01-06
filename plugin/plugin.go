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

	"github.com/rjeczalik/flyingexec/router"
	"github.com/rjeczalik/flyingexec/util"
)

var errRead = errors.New("plugin: reading port and/or ID from stdin failed")

type Plugin interface {
	Init(routerAddr string, version *string) error
}

type Connector struct {
	ID         uint16
	RouterAddr string
	Listener   net.Listener
	Dial       util.Dialer
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
	_, por, err := net.SplitHostPort(c.Listener.Addr().String())
	if err != nil {
		return "", err
	}
	port, _ := strconv.Atoi(por)
	cli, err := c.Dial("tcp", c.RouterAddr)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	req := router.RegisterRequest{
		ID:      c.ID,
		Service: reflect.TypeOf(p).Elem().Name(),
		Port:    uint16(port),
	}
	var version string
	err = cli.Call("Router.Register", req, &version)
	return version, err
}

func readUintFrom(r util.StatReader, count int) (arr []string, err error) {
	if fi, err := r.Stat(); err != nil || fi.Size() == 0 {
		return nil, errRead
	}
	scan := bufio.NewScanner(r)
	scan.Split(bufio.ScanWords)
	arr = make([]string, 0, count)
	for scan.Scan() && count > 0 {
		arr = append(arr, scan.Text())
		if _, err = strconv.ParseUint(arr[len(arr)-1], 10, 16); err != nil {
			return
		}
		count -= 1
	}
	if scan.Err() != nil || count != 0 {
		err = errRead
	}
	return
}

func newConnector(r util.StatReader) (c *Connector, err error) {
	c = &Connector{
		Dial: func(network, address string) (util.CallCloser, error) {
			return rpc.Dial(network, address)
		},
	}
	var arr []string
	if arr, err = readUintFrom(r, 2); err != nil {
		err = errRead
		return
	}
	id, _ := strconv.Atoi(arr[0])
	c.ID, c.RouterAddr = uint16(id), "localhost:"+arr[1]
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
