// +build !windows

package flying

import (
	"os"
	"os/exec"
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

func run(c *Client, cmd []string) error {
	return runconsole(c, cmd)
}

func command(cmd string, args ...string) *exec.Cmd {
	return exec.Command(cmd, args...)
}
