package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/x-formation/flyingexec/flying"

	"bitbucket.org/kardianos/osext"
)

const usage = `usage: flying command [args]...`

func die(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}

func main() {
	if len(os.Args) == 1 {
		die(usage)
	}
	if err := run(os.Args[1:]); err != nil {
		die(err)
	}
}

func run(cmd []string) (err error) {
	dir, err := osext.ExecutableFolder()
	if err != nil {
		return
	}
	// TODO(rjeczalik): Rotate log each x MiB?
	path := filepath.Join(dir, "flying."+cmd[0]+".log")
	log, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0664)
	if err != nil {
		return
	}
	err = flying.Run(cmd, flying.MultiWriteCloser(log, os.Stdout))
	return
}
