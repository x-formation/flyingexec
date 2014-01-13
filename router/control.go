package router

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
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
		pluginDir: filepath.Join(dir, "plugins"),
		logDir:    filepath.Join(dir, "logs"),
	}
	if err = os.MkdirAll(ctrl.pluginDir, 0775); err != nil {
		return
	}
	if err = os.MkdirAll(ctrl.logDir, 0775); err != nil {
		return
	}
	// TODO listener / loadPlugins
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

func (ctrl *control) newPlugin(exec string) (p *plugin, err error) {
	_, port, err := net.SplitHostPort(ctrl.listener.Addr().String())
	if err != nil {
		return
	}
	p = &plugin{
		id:  uint16(ctrl.counter.Next()),
		cmd: exec.Command(filepath.Join(ctrl.pluginDir, exec)),
	}
	logFile := filepath.Join(logDir, exec+".log")
	if p.log, err = os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
		return
	}
	p.cmd.Stdout, p.cmd.Stderr = p.log, p.log
	p.cmd.Stdin = bytes.NewReader([]byte(port + " " + strconv.Itoa(int(p.id))))
	return
}

func (ctrl *control) execPluginDir() (err error) {
	execs, err := ioutil.ReadDir(pluginDir)
	if err != nil {
		return
	}
	for _, exec := range execs {
		p, err := ctrl.newPlugin(exec)
		if err != nil {
			return
		}
		if err = ctrl.plugin.addPending(p); err != nil {
			return
		}
		// TODO exec + monitor a plugin
	}
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
	p, err := ctrl.popPending(req.ID)
	if err != nil {
		return
	}
	p.service, p.port = req.Service, "localhost:"+strconv.Itoa(int(req.Port))
	if err = p.init(ctrl.listener.Addr().String()); err != nil {
		return
	}
	if err = ctrl.plugins.addReg(p); err != nil {
		return
	}
	log.Println("router control: plugin successfully registered:", p)
	return
}
