package util

import (
	"bytes"
	"io"
	"os"
	"time"
)

type StatReader interface {
	Stat() (os.FileInfo, error)
	io.Reader
}

type Buffer struct {
	*bytes.Buffer
}

func (b *Buffer) Name() (s string) {
	return
}

func (b *Buffer) Size() int64 {
	return int64(b.Len())
}

func (b *Buffer) Mode() (mode os.FileMode) {
	return
}

func (b *Buffer) ModTime() (t time.Time) {
	return
}

func (b *Buffer) IsDir() (dir bool) {
	return
}

func (b *Buffer) Sys() (v interface{}) {
	return
}

func (b *Buffer) Stat() (os.FileInfo, error) {
	return b, nil
}

func NewStatReader(s string) StatReader {
	return &Buffer{bytes.NewBufferString(s)}
}
