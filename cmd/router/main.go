package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rjeczalik/flyingexec/router"
)

var addr string

func init() {
	flag.StringVar(&addr, "addr", ":0", "the TCP network address for the RPC")
	flag.Parse()
}

func fatal(err error) {
	fmt.Println("router error:", err)
	os.Exit(1)
}

func main() {
	rt, err := router.NewRouter()
	if err != nil {
		fatal(err)
	}
	if err = rt.ListenAndServe(addr); err != nil {
		fatal(err)
	}
}
