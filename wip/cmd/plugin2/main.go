package main

import (
	"fmt"
	"log"

	"github.com/rjeczalik/flyingexec/plugin"
)

type Plugin2 struct{}

func (p Plugin2) Init(routerAddr string, _ *string) error {
	return fmt.Errorf("plugin: invalid router addr:", routerAddr)
}

func main() {
	if err := plugin.ListenAndServe(new(Plugin2)); err != nil {
		log.Fatalf("plugin2: serving plugin failed with %q", err)
	}
}
