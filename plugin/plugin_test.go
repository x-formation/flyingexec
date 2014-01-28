package plugin

import (
	"net"
	"net/rpc"
	"reflect"
	"testing"
	"time"

	"github.com/rjeczalik/flyingexec/testutil"
	"github.com/rjeczalik/flyingexec/util"
)

type controlServiceSpy struct {
	t    *testing.T
	l    net.Listener
	req  map[string]string
	ver  *string
	done chan<- bool
}

func (srvc *controlServiceSpy) Register(req map[string]string, _ *struct{}) (err error) {
	defer func() { srvc.done <- (err == nil) }()
	if !reflect.DeepEqual(srvc.req, req) {
		srvc.t.Errorf("expected req to be %+v, got %+v instead", srvc.req, req)
		return
	}
	var ver string
	conn, err := util.DefaultNet.Dial("tcp", ":"+req["Port"])
	if err != nil {
		srvc.t.Errorf("expected err to be nil, got %q instead", err)
	}
	cli := rpc.NewClient(conn)
	defer cli.Close()
	if err = cli.Call(req["Service"]+".Init", srvc.l.Addr().String(), &ver); err != nil {
		srvc.t.Errorf("expected err to be nil, got %q instead", err)
	}
	if ver != *srvc.ver {
		srvc.t.Errorf("expected ver to be %q, got %q instead", *srvc.ver, ver)
	}
	return
}

func setupControlService(t *testing.T) (cleanup func(), req map[string]string, ver *string, wait <-chan bool) {
	var err error
	done := make(chan bool, 1)
	defer func() {
		if err != nil {
			t.Fatalf("expected err to be nil, got %v instead", err)
		}
	}()
	req, ver = make(map[string]string), new(string)
	srvc := &controlServiceSpy{t: t, req: req, ver: ver, done: done}
	srvc.l, err = util.DefaultNet.Listen("tcp", ":0")
	if err != nil {
		return
	}
	if _, req["Port"], err = net.SplitHostPort(srvc.l.Addr().String()); err != nil {
		return
	}
	srv := rpc.NewServer()
	srv.RegisterName("__ControlService", srvc)
	go func() {
		for {
			conn, err := srvc.l.Accept()
			if err != nil {
				break
			}
			go srv.ServeConn(conn)
		}
	}()
	cleanup = func() { srvc.l.Close() }
	wait = done
	return
}

func newTestConnector(t *testing.T, req map[string]string) *Connector {
	c, err := NewConnector(req["ID"], "localhost:"+req["Port"])
	if err != nil {
		t.Fatalf("expected error to be nil, got %v instead", err)
	}
	if _, req["Port"], err = net.SplitHostPort(c.Listener.Addr().String()); err != nil {
		t.Fatalf("expected err to be nil, got %v instead", err)
	}
	return c
}

type PluginTest struct {
	ver string
}

func (p *PluginTest) Init(_ string, ver *string) (err error) {
	*ver = p.ver
	return
}

// TODO any simpler? at least refactor mocks to testutil package
func TestRegisterRequest(t *testing.T) {
	defer testutil.GuardPanic(t)
	cleanup, req, ver, wait := setupControlService(t)
	defer cleanup()
	req["ID"], req["Service"], *ver = "123", "PluginTest", "1.0.0"
	c := newTestConnector(t, req)
	go testutil.Must(t, func() error { return Serve(c, &PluginTest{ver: *ver}) })
	select {
	case ok := <-wait:
		if !ok {
			t.Errorf("expected ok to be true")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out after 5 seconds")
	}
}
