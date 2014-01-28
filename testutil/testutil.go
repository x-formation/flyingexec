package testutil

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"testing"

	"github.com/rjeczalik/flyingexec/util"
)

func init() {
	util.DefaultNet = InMemNet
	runtime.GOMAXPROCS(runtime.NumCPU())
	WatchInterrupt()
}

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

func Must(t *testing.T, f func() error) {
	wait := make(chan error, 1)
	for {
		select {
		case wait <- f():
		case err := <-wait:
			if err != nil {
				t.Fatalf("testutil.Must: expected err to be nil, got %q instead", err)
			}
			break
		}
	}
}
