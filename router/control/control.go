package control

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

type StartStopper interface {
	Start(id uint16, service net.Addr, recovery func(error) error) error
	Stop() error
}

type Cmd struct {
	cmd  *exec.Cmd
	log  *os.File
	stop chan struct{}
}

func NewCmd(exe string, log string) (cmd *Cmd, err error) {
	cmd = &Cmd{
		cmd:  exec.Command(exe),
		stop: make(chan struct{}),
	}
	if cmd.log, err = os.OpenFile(log, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
		return
	}
	cmd.cmd.Stdout, cmd.cmd.Stderr = cmd.log, cmd.log
	return
}

func (c *Cmd) Start(id uint16, service net.Addr, recovery func(error) error) error {
	_, port, err := net.SplitHostPort(service.Addr().String())
	if err != nil {
		return err
	}
	cmd.cmd.Stdin = bytes.NewReader([]byte(port + " " + strconv.Itoa(int(id))))
	if err = cmd.cmd.Start(); err != nil {
		return err
	}
	go cmd.monitor(recovery)
	return
}

func (c *Cmd) Stop() error {
	cmd.stop <- struct{}{}
	return cmd.Process.Kill()
}

func (c *Cmd) monitor(recovery func(error) error) {
	wait := make(chan error, 1)
	for {
		select {
		case wait <- cmd.cmd.Wait():
		case err := <-wait:
			recovery(err)
		case <-cmd.stop:
			return
		}
	}
}

type plugin struct {
	StartStopper
	addr    net.Addr
	version string
}

type PluginControl struct {
	mu      sync.RWMutex
	plugins map[string]*plugin
	// TODO
}

func New() (pc *PluginControl, err error) {
	pc = &PluginControl{}
	return // TODO
}

func (pc *PluginControl) Run(s StartStopper) (err error) {
	i
	return // TODO
}

func (pc *PluginControl) Dial(service string) (conn net.Conn, err error) {
	return // TODO
}
