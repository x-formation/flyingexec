package wip

import (
	"fmt"
	"net/rpc"
	"reflect"
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

type Cmpx struct {
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

func (p Cmpx) Complex(req, res *map[string]string) (err error) {
	*res = *req
	fmt.Println("XD")
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

func (pd *router) call2(t *testing.T, method string, req, exp map[string]string) {
	c, err := rpc.Dial("tcp", "localhost:"+pd.port)
	if err != nil {
		t.Error("call error:", method, pd.port)
	}
	defer c.Close()
	var res map[string]string
	if err = c.Call(method, &req, &res); err != nil {
		t.Fatal("call error:", method, err)
	}
	if !reflect.DeepEqual(res, exp) {
		t.Error("call error:", method, res, exp)
	}
}

func (pd *router) call2p(t *testing.T, service, method string, req, exp map[string]string) {
	c, err := rpc.Dial("tcp", "localhost:"+pd.plugins[service])
	if err != nil {
		t.Error("call error:", method, pd.plugins[service])
	}
	defer c.Close()
	res := make(map[string]string)
	if err = c.Call(method, &req, &res); err != nil {
		t.Fatal("call error:", method, err)
	}
	if !reflect.DeepEqual(res, exp) {
		t.Error("call error:", method, res, exp)
	}
}
func Test(t *testing.T) {
	defer testext.GuardPanic(t)
	p, err := NewRouter()
	if err != nil {
		t.Fatal("newrouter:", err)
	}
	err = p.Start(Add{}, Mul{}, Cmpx{})
	if err != nil {
		t.Error("start:", err)
	}
	m := map[string]string{
		"sdfsdf": "sdfsdf",
		"XD":     "sdfdsf",
		"324sdf": "xcvxcv45",
		"erewr":  "sdfsdf",
		"ew35":   "werwersdvsdf",
		"wrwer":  "wer234234",
	}
	p.call(t, "Add.Three", 6, 9)
	p.call(t, "Add.Four", 10, 14)
	p.call2(t, "Cmpx.Complex", m, m)
	p.call2(t, "Cmpx.Complex", m, m)
	p.call(t, "Add.Three", 13, 16)
	p.call(t, "Mul.Two", 6, 12)
	p.call2p(t, "Cmpx", "Cmpx.Complex", m, m)
	p.call(t, "Mul.Four", 6, 24)
	p.call(t, "Mul.Four", 2, 8)
	p.call2(t, "Cmpx.Complex", m, m)
	p.call(t, "Add.Three", 12, 15)
	p.call(t, "Add.Four", -1, 3)
}
