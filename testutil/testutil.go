package testutil

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
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
