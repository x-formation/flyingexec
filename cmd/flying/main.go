package main

import (
	"fmt"
	"os"

	"github.com/x-formation/flyingexec/flying"
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
	// TODO(rjeczalik): File lock?
	if err := flying.Run(os.Args[1:]); err != nil {
		die(err)
	}
}
