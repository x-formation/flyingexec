package wip

import (
	"net/rpc"
	"testing"

	"github.com/rjeczalik/gpff/testext"
)

func init() {
	testext.WatchInterrupt()
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
	defer testext.GuardPanic(t)
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
