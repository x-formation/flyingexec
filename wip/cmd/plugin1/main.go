package main

import (
	"log"

	"github.com/rjeczalik/flyingexec/plugin"
)

type Plugin1 struct{}

func (p Plugin1) Init(_ string, version *string) (err error) {
	*version = "1.0.0"
	return
}

func main() {
	if err := plugin.ConnectAndServe(new(Plugin1)); err != nil {
		log.Fatalf("plugin1: serving plugin failed with %q", err)
	}
}
