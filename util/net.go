package util

import "net"

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

var DefaultNet Net = new(stdNet)
