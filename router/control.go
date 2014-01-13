package router

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/rjeczalik/flyingexec/util"
)

var errRegisterReq = errors.New(`router: register request ill-formed`)
var errPluginVersion = errors.New(`control: plugin version empty`)
var errPluginTimeout = errors.New(`control: awaiting registration to complete has timed out`)

type RegisterRequest struct {
	ID      uint16
	Service string
	Port    uint16
}

func (req *RegisterRequest) valid() (err error) {
	if req.ID == 0 || req.Service == "" || req.Port == 0 {
		err = errRegisterReq
	}
	return
}

type control struct {
	plugins   *pluginContainer
	event     chan interface{}
	counter   util.Counter
	listener  net.Listener
	pluginDir string
	logDir    string
}

func newControl(execdir string) (ctrl *control, err error) {
	ctrl = &control{
		plugins:   newPluginContainer(),
		event:     make(chan interface{}, 10),
		counter:   1,
		pluginDir: filepath.Join(execdir, "plugins"),
		logDir:    filepath.Join(execdir, "logs"),
	}
	if err = os.MkdirAll(ctrl.pluginDir, 0775); err != nil {
		return
	}
	if err = os.MkdirAll(ctrl.logDir, 0775); err != nil {
		return
	}
	// TODO listener / loadPlugins
	return
}

func (ctrl *control) pluginByService(name string) (*plugin, error) {
	return ctrl.pluginByService(name)
}

func (ctrl *control) serve() {
	srv := rpc.NewServer()
	srv.RegisterName("Control", ctrl)
	log.Println("router plugin control service listening on", ctrl.listener.Addr().String(), ". . .")
	for {
		conn, err := ctrl.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go srv.ServeConn(conn)
	}
}

func (ctrl *control) newPlugin(exe string) (p *plugin, err error) {
	_, port, err := net.SplitHostPort(ctrl.listener.Addr().String())
	if err != nil {
		return
	}
	p = &plugin{
		id:  uint16(ctrl.counter.Next()),
		cmd: exec.Command(filepath.Join(ctrl.pluginDir, exe)),
	}
	logFile := filepath.Join(ctrl.logDir, exe+".log")
	if p.log, err = os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
		return
	}
	p.cmd.Stdout, p.cmd.Stderr = p.log, p.log
	p.cmd.Stdin = bytes.NewReader([]byte(port + " " + strconv.Itoa(int(p.id))))
	return
}

func (ctrl *control) execPluginDir() (err error) {
	exes, err := ioutil.ReadDir(ctrl.pluginDir)
	if err != nil {
		return
	}
	for _, exe := range exes {
		p, err := ctrl.newPlugin(exe.Name())
		if err != nil {
			return err
		}
		if err = ctrl.plugins.addPending(p); err != nil {
			return err
		}
		// TODO exec + monitor a plugin
	}
	return
}

func (ctrl *control) Register(req RegisterRequest, _ *struct{}) (err error) {
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()
	if err = req.valid(); err != nil {
		return
	}
	p, err := ctrl.plugins.popPending(req.ID)
	if err != nil {
		return
	}
	p.service, p.addr = req.Service, "localhost:"+strconv.Itoa(int(req.Port))
	if err = p.init(ctrl.listener.Addr().String()); err != nil {
		return
	}
	if err = ctrl.plugins.addReg(p); err != nil {
		return
	}
	log.Println("router control: plugin successfully registered:", p)
	return
}
