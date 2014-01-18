package control

import (
	"bytes"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
)

type StartStopper interface {
	Start(id uint16, service net.Addr, recovery func(error) error) error
	Stop() error
}

type Cmd struct {
	id        uint16
	cmd       *exec.Cmd
	log       *os.File
	err       chan error // TODO: add Err() method, possibly to
	isclosing uint32     // StartStopper interface + add regular
	// error member to store result of err chan.
	// I may want to recover from recovery failure
	// elsewhere.
}

func NewCmd(exe string, log string) (c *Cmd, err error) {
	c = &Cmd{
		cmd: exec.Command(exe),
		err: make(chan error, 1),
	}
	if c.log, err = os.OpenFile(log, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
		return
	}
	c.cmd.Stdout, c.cmd.Stderr = cmd.log, cmd.log
	return
}

func (c *Cmd) Start(id uint16, service net.Addr, recovery func(error) error) error {
	_, port, err := net.SplitHostPort(service.Addr().String())
	if err != nil {
		return err
	}
	c.id, c.cmd.Stdin = id, bytes.NewReader([]byte(port+" "+strconv.Itoa(int(id))))
	if err = c.cmd.Start(); err != nil {
		return err
	}
	// TODO: zero isclosing
	go c.monitor(recovery)
	return
}

func (c *Cmd) Stop() error {
	atomic.StoreUint32(&c.isclosing, 1)
	// TODO call plugin and ask for gracefull shutdown
	err1 := c.Process.Kill()
	err2 := <-c.err
	if err2 != nil {
		return err2
	}
	return err1
}

func (c *Cmd) monitor(recovery func(error) error) {
	wait := make(chan error, 1)
	for {
		select {
		case wait <- c.cmd.Wait():
		case perr := <-wait:
			if atomic.LoadUint32(&c.isclosing) == 1 {
				c.err <- perr
				return
			}
			err := recovery(perr)
			if err != nil {
				log.Printf("control: failed to recover plugin #%d: %q", c.id, err)
				// TODO save err to member descibed at line 23
			}
			// TODO refactor c.monitor() to be go-started at NewCmd
			// in a daemon mannger, then refactor c.monitor to not return
			// on succussfull recovery, but to jump to watching new process
			// instead; possibly recovery func will be kept by c, like the id is. (sync?)
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
	return // TODO
}

func (pc *PluginControl) Dial(service string) (conn net.Conn, err error) {
	return // TODO
}
