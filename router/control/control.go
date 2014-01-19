package control

import (
	"bytes"
	"errors"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rjeczalik/flyingexec/util"
)

type Runner interface {
	Start(id uint16, service net.Addr, recovery func(error) error) error
	Stop() error
}

type RunnerMaker interface {
	Make(exe string) (Runner, error)
}

// TODO: add Err() method, possibly to StartStopper interface + add regular
// error member to store result of err chan. I may want to recover from recovery
// failure elsewhere.
type CmdRunner struct {
	err       chan error
	cmd       *exec.Cmd
	isclosing uint32
	id        uint16
}

type CmdRunnerMaker struct{}

// TODO go c.monitor() here
func (CmdRunnerMaker) Make(exe string) (c *Cmd, err error) {
	c = &Cmd{
		cmd: exec.Command(exe),
		err: make(chan error, 1),
	}
	if c.cmd.Stdout, err = os.OpenFile(exe+".log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
		return
	}
	c.cmd.Stderr = c.cmd.Stdout
	return
}

func (c *CmdRunner) Start(id uint16, service net.Addr, recovery func(error) error) error {
	_, port, err := net.SplitHostPort(service.Addr().String())
	if err != nil {
		return err
	}
	c.id, c.cmd.Stdin = id, bytes.NewReader([]byte(port+" "+strconv.Itoa(int(id))))
	if err = c.cmd.Start(); err != nil {
		return err
	}
	go c.monitor(id, recovery)
	return
}

// TODO call plugin and ask for gracefull shutdown, probably net.Addr should be
// stored by c
func (c *CmdRunner) Stop() error {
	atomic.StoreUint32(&c.isclosing, 1)
	err, perr := c.Process.Kill(), <-c.err
	atomic.StoreUint32(&c.isclosing, 0)
	if perr != nil {
		return perr
	}
	return err
}

// TODO refactor c.monitor() to be go-started at NewCmd
// in a daemon mannger, then refactor c.monitor to not return
// on succussfull recovery, but to jump to watching new process
// instead; possibly recovery func will be kept by c, like the id is. (sync?)
func (c *Runner) monitor(id uint16, recovery func(error) error) {
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
				log.Printf("control: failed to recover plugin #%d: %q", id, err)
				// TODO save err to member descibed at line 23
			}
			return
		}
	}
}

var errTimeout = errors.New(`control: awaiting registration to complete has timed out`)

type plugin struct {
	ver  string
	addr string
	run  Runner
}

type Control struct {
	maker   RunnerMaker
	plugins map[string]*plugin
	srvc    *Service
	mu      sync.RWMutex
	counter util.Counter
}

func NewControl() (ctrl *Control, err error) {
	ctrl = &Control{
		maker:   CmdRunnerMaker{},
		plugins: make(map[string]*plugin),
		counter: 1,
	}
	if ctrl.srvc, err = newService(); err != nil {
		return
	}
	return // TODO
}

func (ctrl *Control) Run(exe string) error {
	run, err := ctrl.maker.Make(exe)
	if err != nil {
		return err
	}
	addr, id, ch := ctrl.srvc.Addr(), uint16(ctrl.counter.Next()), make(chan res)
	pc.srvc.addPen(id, ch)
	defer func() {
		pc.srvc.remPen(id)
		close(ch)
	}()
	// TODO recovery
	if err = run.Start(id, ctrl.srvc.lis.Addr(), nil); err != nil {
		return err
	}
	var r res
	select {
	case r = <-ch:
		if r.err != nil {
			return r.err
		}
	case <-time.After(30 * time.Second):
		return errTimeout
	}
	p := &plugin{
		ver:  r.version,
		addr: "localhost:" + strconv.Itoa(int(r.port)),
		run:  run,
	}
	// TODO handle dups
	ctrl.mu.Lock()
	ctrl.plugins[r.name] = p
	ctrl.mu.Unock()
	return
}

func (ctrl *Control) Dial(service string) (conn net.Conn, err error) {
	ctrl.mu.RLock()
	// TODO
	ctrl.mu.RUnlock()
	return
}
