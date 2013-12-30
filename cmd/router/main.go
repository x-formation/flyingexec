package main

import (
	"flag"
	"log"
	"os"

	"github.com/rjeczalik/gpff/router"
)

var addr string

func init() {
	flag.StringVar(&addr, "addr", ":0", "the TCP network address for the RPC")
	flag.Parse()
}

func main() {
	rt := router.NewRouter()
	if err := rt.ListenAndServe(addr); err != nil {
		log.Println("router error:", err)
		os.Exit(1)
	}
}
