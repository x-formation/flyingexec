package plugin

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
	Start(id uint32, service net.Addr, onstop func(bool, error)) error
	Stop() error
}

type RunnerFactory interface {
	New(exe string) (Runner, error)
}

// TODO: add Err() method, possibly to StartStopper interface + add regular
// error member to store result of err chan. I may want to recover from recovery
// failure elsewhere.
type CmdRunner struct {
	err       chan error
	cmd       *exec.Cmd
	isclosing uint32
	id        uint32
}

type CmdRunnerFactory struct{}

// TODO go c.monitor() here
func (CmdRunnerFactory) New(exe string) (r Runner, err error) {
	c := &CmdRunner{
		cmd: exec.Command(exe),
		err: make(chan error, 1),
	}
	if c.cmd.Stdout, err = os.OpenFile(exe+".log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
		return
	}
	c.cmd.Stderr, r = c.cmd.Stdout, c
	return
}

func (c *CmdRunner) Start(id uint32, service net.Addr, onstop func(bool, error)) error {
	_, port, err := net.SplitHostPort(service.String())
	if err != nil {
		return err
	}
	c.id, c.cmd.Stdin = id, bytes.NewReader([]byte(port+" "+strconv.Itoa(int(id))))
	if err = c.cmd.Start(); err != nil {
		return err
	}
	go c.monitor(onstop)
	return nil
}

// TODO call plugin and ask for gracefull shutdown, probably net.Addr should be
// stored by c
func (c *CmdRunner) Stop() error {
	atomic.StoreUint32(&c.isclosing, 1)
	err, perr := c.cmd.Process.Kill(), <-c.err
	atomic.StoreUint32(&c.isclosing, 0)
	if perr != nil {
		return perr
	}
	return err
}

// TODO refactor c.monitor() to be go-started at NewCmd
// in a daemon manner, then refactor c.monitor to not return
// on succussfull recovery, but to jump to watching new process
// instead
func (c *CmdRunner) monitor(onstop func(bool, error)) {
	wait := make(chan error, 1)
	for {
		select {
		case wait <- c.cmd.Wait():
		case perr := <-wait:
			restart := true
			if atomic.LoadUint32(&c.isclosing) == 1 {
				restart = false
				c.err <- perr
			}
			onstop(restart, perr)
			return
		}
	}
}

var errTimeout = errors.New(`control: awaiting registration to complete has timed out`)

// TODO refactor: plugin and res overlaps
type plugin struct {
	serv string
	ver  string
	addr string
	exe  string
}

type Control struct {
	Runner RunnerFactory
	// TODO multi-index map
	plugins struct {
		id      map[uint32]*plugin // id-lookup used by Service
		service map[string]*plugin // service-name-lookup used by Router
		exe     map[string]*plugin // file-path-lookup used bu Loader
	}
	srvc    *Service
	mu      sync.RWMutex
	counter util.Counter
}

func NewControl() (ctrl *Control, err error) {
	ctrl = &Control{
		Runner:  CmdRunnerFactory{},
		counter: 1,
	}
	ctrl.plugins.id = make(map[uint32]*plugin)
	ctrl.plugins.service = make(map[string]*plugin)
	ctrl.plugins.exe = make(map[string]*plugin)
	if ctrl.srvc, err = newService(); err != nil {
		return
	}
	go ctrl.srvc.serve()
	return
}

// TODO mock logging, too verbose
func (ctrl *Control) restartOrRemove(id uint32, restart bool, err error) {
	if err != nil && restart {
		log.Printf("control: plugin #%d stopped unexpectedly and is going to "+
			"be restarted: %v", id, err)
	} else if err != nil {
		log.Printf("control: plugin #%d stopped with error: %v", id, err)
	} else {
		log.Printf("control: plugin #%d stopped", id)
	}
	// TODO handle missing id?
	ctrl.mu.Lock()
	p := ctrl.plugins.id[id]
	delete(ctrl.plugins.service, p.serv)
	delete(ctrl.plugins.id, id)
	ctrl.mu.Unlock()
	if restart {
		if err = ctrl.Run(p.exe); err != nil {
			log.Printf("control: failed to restart the plugin #%d: %v", id, err)
		}
	}
}

func (ctrl *Control) Run(exe string) error {
	run, err := ctrl.Runner.New(exe)
	if err != nil {
		return err
	}
	addr, id, ch := ctrl.srvc.lis.Addr(), ctrl.counter.Next(), make(chan res)
	ctrl.srvc.addPen(id, ch)
	// TODO document this magic / refactor to separate context?
	defer func() {
		ctrl.srvc.remPen(id)
		close(ch)
	}()
	// TODO onstop would benefit from more state
	onstop := func(restart bool, err error) {
		ctrl.restartOrRemove(id, restart, err)
	}
	if err = run.Start(id, addr, onstop); err != nil {
		return err
	}
	// TODO dispatch event to (*Control).EventLoop?
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
		serv: r.serv,
		ver:  r.ver,
		addr: r.addr,
		exe:  exe,
	}
	// TODO handle dups on service name
	ctrl.mu.Lock()
	ctrl.plugins.id[r.id] = p
	ctrl.plugins.service[r.serv] = p
	ctrl.mu.Unlock()
	return nil
}

func (ctrl *Control) Dial(service string) (conn net.Conn, err error) {
	ctrl.mu.RLock()
	p, ok := ctrl.plugins.service[service]
	ctrl.mu.RUnlock()
	if !ok {
		return nil, errors.New("rpc: can't find service " + service)
	}
	conn, err = util.DefaultNet.Dial("tcp", p.addr)
	return
}
