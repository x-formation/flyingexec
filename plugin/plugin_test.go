package plugin

import (
	"net/rpc"
	"strconv"
	"testing"

	"github.com/rjeczalik/flyingexec/router"
	"github.com/rjeczalik/flyingexec/testutil"
	"github.com/rjeczalik/flyingexec/util"
)

func init() {
	util.DefaultNet = testutil.InMemNet
	testutil.WatchInterrupt()
}

func TestNew(t *testing.T) {
	defer testutil.GuardPanic(t)
	table := []struct {
		adminPort string
		ID        string
		err       error
	}{
		{"8080", "1", nil},
		{"33305", "510", nil},
		{"6600", "0", nil},
		{"55695", "43002", nil},
		{"", "", errRead},
		{"asd", "", errRead},
		{"13123", "", errRead},
		{"65560", "123", errRead},
		{"123", "-1", errRead},
		{"2342", "qwe", errRead},
	}
	for _, row := range table {
		c, err := NewConnector(row.adminPort, row.ID)
		if err != row.err {
			t.Errorf("expected %v, got %v instead", row.err, err)
			continue
		}
		if err == nil {
			if id := strconv.Itoa(int(c.ID)); id != row.ID {
				t.Errorf("expected %q, got %q instead", row.ID, id)
			}
			if adminAddr := "localhost:" + row.adminPort; c.AdminAddr != adminAddr {
				t.Errorf("expected localhost:%d, got %v instead", adminAddr, c.AdminAddr)
			}
			c.Listener.Close()
		}
	}
}

type Admin struct {
	t    *testing.T
	req  *router.RegisterRequest
	done chan<- bool
}

func (a Admin) Register(req router.RegisterRequest, _ *struct{}) (err error) {
	if a.req.ID != req.ID {
		a.t.Errorf("expected ID to be %v, was %v instead", a.req.ID, req.ID)
	}
	if a.req.Service != req.Service {
		a.t.Errorf("expected service name to be %v, was %v instead", a.req.Service, req.Service)
	}
	if a.req.Port != req.Port {
		a.t.Errorf("expected port to be %v, was %v instead", a.req.Port, req.Port)
	}
	a.done <- true
	return
}

func newTestAdmin(t *testing.T) (cleanup func(), req *router.RegisterRequest, wait <-chan bool) {
	var err error
	done := make(chan bool)
	defer func() {
		if err != nil {
			t.Fatalf("expected err to be nil, got %v instead", err)
		}
	}()
	req = new(router.RegisterRequest)
	a := &Admin{t: t, req: req, done: done}
	l, err := util.DefaultNet.Listen("tcp", ":0")
	if err != nil {
		return
	}
	if _, req.Port, err = util.SplitHostPort(l.Addr().String()); err != nil {
		return
	}
	srv := rpc.NewServer()
	srv.Register(a)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				break
			}
			go srv.ServeConn(conn)
		}
	}()
	cleanup = func() { l.Close() }
	wait = done
	return
}

func newTestConnector(t *testing.T, req *router.RegisterRequest) *Connector {
	c, err := NewConnector(strconv.Itoa(int(req.Port)), strconv.Itoa(int(req.ID)))
	if err != nil {
	}
	if _, req.Port, err = util.SplitHostPort(c.Listener.Addr().String()); err != nil {
		t.Fatalf("expected err to be nil, got %v instead", err)
	}
	return c
}

type PluginTest struct{}

func (PluginTest) Init(_ string, _ *string) (err error) {
	return
}

func TestRegisterRequest(t *testing.T) {
	defer testutil.GuardPanic(t)
	cleanup, req, wait := newTestAdmin(t)
	defer cleanup()
	req.ID, req.Service = 123, "PluginTest"
	c := newTestConnector(t, req)
	go Serve(c, new(PluginTest))
	<-wait
}
