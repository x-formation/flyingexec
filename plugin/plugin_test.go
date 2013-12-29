package plugin

import (
	"encoding/json"
	"io"
	"net/rpc"
	"os"
	"testing"

	"github.com/rjeczalik/gpff/testext"
)

func init() {
	testext.WatchInterrupt()
}

func setup(t *testing.T) func() {
	stdout, stderr = new(testext.SyncBuffer), new(testext.SyncBuffer)
	return func() {
		stdout = os.Stdout
		stderr = os.Stderr
		testext.GuardPanic(t)
	}
}

type Add struct{}

func (a Add) One(req, res *int) (err error) {
	*res = *req + 1
	return
}

func Test(t *testing.T) {
	cleanup := setup(t)
	defer cleanup()

	dec := json.NewDecoder(io.MultiReader(stdout, stderr))

	go Serve(new(Add))

	v := make(map[string]string)
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("expected err to be nil, was %q instead", err)
	}
	port, ok := v["port"]
	if !ok {
		t.Fatal(`expected v to hold a "port" key`)
	}
	c, err := rpc.Dial("tcp", "localhost:"+port)
	if err != nil {
		t.Fatalf("expected err to be nil, was %q instead", err)
	}
	var res int
	if err = c.Call("Add.One", 10, &res); err != nil {
		t.Fatalf("expected err to be nil, was %q instead", err)
	}
	if res != 11 {
		t.Errorf("expected res to be 11, was %q instead", res)
	}
}
