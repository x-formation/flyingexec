// +build windows

package flying

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"bitbucket.org/kardianos/service"
)

// Signals TODO
var Signals = []os.Signal{
	os.Interrupt,
	os.Kill,
}

func interrupt(proc *os.Process) error {
	return callkernel32("GenerateConsoleCtrlEvent", syscall.CTRL_BREAK_EVENT,
		uintptr(proc.Pid))
}

func console() error {
	return callkernel32("AllocConsole")
}

func callkernel32(name string, args ...uintptr) error {
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	p, err := dll.FindProc(name)
	if err != nil {
		return err
	}
	r, _, err := p.Call(args...)
	if r == 0 {
		return err
	}
	return nil
}

func run(c *Client, cmd []string) error {
	if name, cmd, ok := isservice(cmd); ok {
		if name == "" {
			return errEmptyService
		}
		return runservice(c, name, cmd)
	}
	return runconsole(c, cmd)
}

var errEmptyService = errors.New("flying: service name cannot be empty")

func isservice(cmd []string) (string, []string, bool) {
	var name string
	var is bool
	for i := range cmd {
		if cmd[i] == "-service" {
			is = true
			if i+1 == len(cmd) {
				cmd = cmd[:i]
				break
			}
			name = cmd[i+1]
			copy(cmd[i:], cmd[i+2:])
			cmd = cmd[:len(cmd)-2]
			break
		} else if strings.HasPrefix(cmd[i], "-service=") {
			is = true
			name = cmd[i][len("-service="):]
			copy(cmd[i:], cmd[i+1:])
			cmd = cmd[:len(cmd)-1]
			break
		}
	}
	return name, cmd, is
}

func startsig(c *Client, cmd []string, errch chan<- error, ch chan os.Signal, svc service.Service) error {
	signal.Notify(ch, Signals...)
	interrch := make(chan error, 1)
	if err := c.Start(cmd); err != nil {
		return err
	}
	go func() {
		go func() {
			<-ch
			interrch <- c.Interrupt()
		}()
		err := c.Wait()
		select {
		case e := <-interrch:
			errch <- nonil(err, e)
		default:
			errch <- nonil(err, svc.Stop())
		}
	}()
	return nil
}

func runservice(c *Client, name string, cmd []string) error {
	srvc, err := service.NewService(name, "", "")
	if err != nil {
		return err
	}
	errch, ch := make(chan error, 1), make(chan os.Signal, 1)
	err = srvc.Run(
		func() error {
			if err := console(); err != nil {
				return err
			}
			return startsig(c, cmd, errch, ch, srvc)
		},
		func() error {
			ch <- os.Interrupt
			if u := <-errch; u != nil {
				srvc.Error(err.Error())
			}
			return nil
		})
	if err != nil {
		srvc.Error(err.Error())
	}
	return nil
}

func command(cmd string, args ...string) *exec.Cmd {
	c := exec.Command(cmd, args...)
	c.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return c
}
