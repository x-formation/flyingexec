package util

import (
	"io"
	"os"
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
