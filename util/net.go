package util

import (
	"net"
	"strconv"
)

type Net interface {
	Dial(network, address string) (net.Conn, error)
	Listen(network, address string) (net.Listener, error)
}

type stdNet struct{}

func (stdNet) Dial(network, address string) (net.Conn, error) {
	return net.Dial(network, address)
}

func (stdNet) Listen(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func SplitHostPort(hostport string) (host string, portNum uint16, err error) {
	var port string
	if host, port, err = net.SplitHostPort(hostport); err != nil {
		return
	}
	var n uint64
	if n, err = strconv.ParseUint(port, 10, 16); err != nil {
		return
	}
	portNum = uint16(n)
	return
}

var DefaultNet Net = new(stdNet)
