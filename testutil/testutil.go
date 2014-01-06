package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"testing"
	"time"

	"github.com/rjeczalik/flyingexec/util"
)

func stack(full bool) string {
	size := 1024
	if full {
		size = 8192
	}
	stack := make([]byte, size)
	_ = runtime.Stack(stack, full)
	return string(stack)
}

func WatchInterrupt() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go func() {
		if _, ok := <-ch; ok {
			fmt.Printf("error: interrupted; stacktrace:\n\n%s\n", stack(true))
			os.Exit(1)
		}
	}()
}

func GuardPanic(t *testing.T) {
	if r := recover(); r != nil {
		t.Errorf("recovered from panic \"%v\"; stacktrace:\n\n%s", r, stack(false))
	}
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

func NewStatReader(s string) util.StatReader {
	return &Buffer{bytes.NewBufferString(s)}
}
