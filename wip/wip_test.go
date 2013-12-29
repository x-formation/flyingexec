package wip

import (
	"fmt"
	"net/rpc"
	"os"
	"os/signal"
	"runtime"
	"testing"
)

func stack(full bool) string {
	size := 1024
	if full {
		size = 8192
	}
	stack := make([]byte, size)
	_ = runtime.Stack(stack, full)
	return string(stack)
}

func init() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go func() {
		if _, ok := <-ch; ok {
			fmt.Printf("error: interrupted; stacktrace:\n\n%s\n", stack(true))
			os.Exit(1)
		}
	}()
}

func pguard(t *testing.T) {
	if r := recover(); r != nil {
		t.Errorf("recovered from panic \"%v\"; stacktrace:\n\n%s", r, stack(false))
	}
}

type Mul struct {
	Plugin
}

type Add struct {
	Plugin
}

func (p Mul) Two(req, res *int) (err error) {
	*res = (*req) * 2
	return
}

func (p Mul) Four(req, res *int) (err error) {
	*res = (*req) * 4
	return
}

func (p Add) Three(req, res *int) (err error) {
	*res = (*req) + 3
	return
}

func (p Add) Four(req, res *int) (err error) {
	*res = (*req) + 4
	return
}

func (pd *router) call(t *testing.T, method string, req int, exp int) {
	c, err := rpc.Dial("tcp", "localhost:"+pd.port)
	if err != nil {
		t.Error("call error:", method, pd.port)
	}
	defer c.Close()
	var res int
	if err = c.Call(method, &req, &res); err != nil {
		t.Fatal("call error:", method, err)
	}
	if res != exp {
		t.Error("call error:", method, res, exp)
	}
}

func Test(t *testing.T) {
	defer pguard(t)
	p, err := NewRouter()
	if err != nil {
		t.Fatal("newrouter:", err)
	}
	err = p.Start(Add{}, Mul{})
	if err != nil {
		t.Error("start:", err)
	}
	p.call(t, "Add.Three", 6, 9)
	p.call(t, "Add.Four", 10, 14)
	p.call(t, "Mul.Two", 6, 12)
	p.call(t, "Add.Three", 13, 16)
	p.call(t, "Mul.Four", 6, 24)
	p.call(t, "Mul.Four", 2, 8)
	p.call(t, "Add.Three", 12, 15)
	p.call(t, "Add.Four", -1, 3)
}
