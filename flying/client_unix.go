// +build !windows

package flying

import (
	"os"
	"os/signal"
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
	ch, err := make(chan os.Signal, 1), make(chan error, 1)
	signal.Notify(ch, Signals...)
	if err := c.Start(cmd); err != nil {
		return err
	}
	go func() {
		for _ = range ch {
			err <- c.Interrupt()
		}
	}()
	return nonil(c.Wait(), <-err)
}
