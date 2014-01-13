package router

import (
	"testing"

	"github.com/rjeczalik/flyingexec/testutil"
)

func init() {
	testutil.WatchInterrupt()
}

func TestRouter(t *testing.T) {
	defer testutil.GuardPanic(t)
}
