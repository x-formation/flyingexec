package util

import (
	"io"
	"net"
	"os"
	"sync/atomic"
)

type StatReader interface {
	Stat() (os.FileInfo, error)
	io.Reader
}

type Dialer interface {
	Dial(network, address string) (io.ReadWriteCloser, error)
}

type Listener interface {
	Listen(network, address string) (net.Listener, error)
}

type NetDialer struct{}
type NetListener struct{}

func (d NetDialer) Dial(network, address string) (io.ReadWriteCloser, error) {
	return net.Dial(network, address)
}

func (l NetListener) Listen(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

var DefaultDialer = new(NetDialer)
var DefaultListener = new(NetListener)

type Counter uint32

func (n *Counter) Next() uint32 {
	return atomic.AddUint32((*uint32)(n), 1)
}
