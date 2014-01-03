package util

import (
	"io"
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

type Dialer func(string, string) (CallCloser, error)

type Counter uint32

func (n *Counter) Next() uint32 {
	return atomic.AddUint32((*uint32)(n), 1)
}
