// +build !windows

package flying

import (
	"os"
	"syscall"
)

// Signals TODO
var Signals = []os.Signal{
	syscall.SIGTERM,
	syscall.SIGSTOP,
	syscall.SIGABRT,
	syscall.SIGHUP,
	syscall.SIGINT,
	syscall.SIGKILL,
}

func interrupt(p *os.Process) error {
	return p.Signal(os.Interrupt)
}
