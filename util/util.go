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

type CallCloser interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
	Close() error
}

type Dialer interface {
	Dial(network, address string) (io.ReadWriteCloser, error)
}

type NetDialer struct{}

func (d NetDialer) Dial(network, address string) (io.ReadWriteCloser, error) {
	return net.Dial(network, address)
}

var DefaultDialer = new(NetDialer)

type Counter uint32

func (n *Counter) Next() uint32 {
	return atomic.AddUint32((*uint32)(n), 1)
}
