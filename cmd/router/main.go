package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"bitbucket.org/kardianos/osext"
	"github.com/rjeczalik/gpff/router"
)

var addr string
var logFile string
var logDir string

func init() {
	dir, err := osext.ExecutableFolder()
	if err != nil {
		dir = "."
	}
	flag.StringVar(&addr, "addr", ":0", "the TCP network address for the RPC")
	flag.StringVar(&logFile, "log", filepath.Join(dir, "router.log"), "router log file")
	flag.StringVar(&logDir, "logdir", dir, "directory for plugin log files")
	flag.Parse()
}

func main() {
	rt := router.NewRouter()
	if err := rt.ListenAndServe(addr); err != nil {
		log.Println("router error:", err)
		os.Exit(1)
	}
}
