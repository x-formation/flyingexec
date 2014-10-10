// +build windows

package flying

import (
	"os"
	"os/signal"
	"syscall"
)

// Signals TODO
var Signals = []os.Signal{
	os.Interrupt,
	os.Kill,
}

// Stolen from $GOROOT/src/os/signal/signal_windows_test.go.
func interrupt(proc *os.Process) (err error) {
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	p, err := dll.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		return
	}
	r, _, err := p.Call(syscall.CTRL_BREAK_EVENT, uintptr(proc.Pid))
	// TODO(rjeczalik) err = nil when r == 0?
	if r != 0 {
		return nil
	}
	return
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
