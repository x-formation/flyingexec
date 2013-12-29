package main

import (
	"fmt"
	"os"

	"github.com/rjeczalik/gpff/router"
)

func main() {
	rt, err := router.NewRouter()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_ = rt
	select {}
}
