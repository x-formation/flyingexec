package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"testing"
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

type SyncBuffer struct {
	b bytes.Buffer
	m sync.RWMutex
}

func (b *SyncBuffer) Read(p []byte) (n int, err error) {
	b.m.RLock()
	n, err = b.b.Read(p)
	b.m.RUnlock()
	return
}

func (b *SyncBuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	n, err = b.b.Write(p)
	b.m.Unlock()
	return
}
