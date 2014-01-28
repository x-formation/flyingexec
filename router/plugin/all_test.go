package plugin

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"testing"

	client "github.com/rjeczalik/flyingexec/plugin"
	"github.com/rjeczalik/flyingexec/testutil"
	"github.com/rjeczalik/flyingexec/util"
)

func init() {
	util.DefaultNet = testutil.InMemNet
	testutil.WatchInterrupt()
}

type Echo struct{}

func (Echo) Init(_ string, ver *string) (err error) {
	*ver = "1.0.0"
	return
}

func (Echo) Echo(req, res *string) (err error) {
	*res = *req
	return
}

type PluginRunner struct {
	conn *client.Connector
	cli  client.Plugin
}

func (p *PluginRunner) Start(id uint16, service net.Addr, onstop func(bool, error)) (err error) {
	_, port, err := net.SplitHostPort(service.String())
	if err != nil {
		return
	}
	p.conn, err = client.NewConnector(port, strconv.Itoa(int(id)))
	if err != nil {
		return
	}
	go client.Serve(p.conn, p.cli)
	return
}

// TODO after client.(*Connector).Stop is implemented
func (p *PluginRunner) Stop() error {
	return nil
}

type PluginRunnerFactory struct {
	Plugins  map[string]client.Plugin
	srvcPort string
	mu       sync.Mutex
}

func (p *PluginRunnerFactory) New(exe string) (r Runner, err error) {
	p.mu.Lock()
	cli, ok := p.Plugins[exe]
	p.mu.Unlock()
	if !ok {
		err = errors.New("plugin: no such file: " + exe)
		return
	}
	r = &PluginRunner{
		cli: cli,
	}
	return
}

func Test(t *testing.T) {
	defer testutil.GuardPanic(t)
	runner := &PluginRunnerFactory{
		Plugins: map[string]client.Plugin{"echo.exe": new(Echo)},
	}
	ctrl, err := NewControl()
	if err != nil {
		t.Fatalf("expected err to be nil, got %q instead", err)
	}
	ctrl.Runner = runner
	if err = ctrl.Run("echo.exe"); err != nil {
		t.Fatalf("expected err to be nil, got %q instead", err)
	}
}
