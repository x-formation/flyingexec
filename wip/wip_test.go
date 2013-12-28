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

type MyPlugin struct {
	Plugin
}

type MyOtherPlugin struct {
	Plugin
}

func (p MyPlugin) Hello(req, res *string) error {
	*res = "Hello " + *req
	return nil
}

func (p MyOtherPlugin) Hai(req, res *string) error {
	*res = "Hai " + *req
	return nil
}

func (pd *plugind) call(method, req string) error {
	c, err := rpc.Dial("tcp", "localhost:"+pd.port)
	if err != nil {
		return err
	}
	var res string
	return c.Call(method, &req, &res)
}

func Test(t *testing.T) {
	defer pguard(t)
	p, err := NewPlugind()
	if err != nil {
		t.Fatal("newplugind:", err)
	}
	err = p.Start(MyPlugin{}, MyOtherPlugin{})
	if err != nil {
		t.Error("start:", err)
	}
	if err = p.call("MyPlugin.Hello", "Joe"); err != nil {
		t.Error("call:", err)
	}
	if err = p.call("MyOtherPlugin.Hai", "Joe"); err != nil {
		t.Error("call:", err)
	}
}
