package flying

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rjeczalik/tools/rw"
)

func now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func prefix(app string) func() string {
	return func() string {
		return fmt.Sprintf("[%s] [%s] ", now(), app)
	}
}

type client struct {
	cmd []string
	wc  io.WriteCloser
}

// NewClient assumes cmd is not empty.
func newClient(cmd []string, wc io.WriteCloser) client {
	return client{
		cmd: cmd,
		wc:  wc,
	}
}

func (c client) logf(format string, v ...interface{}) {
	c.wc.Write([]byte("[" + now() + "] " + fmt.Sprintf(format, v...)))
}

func (c client) run() (err error) {
	defer func() {
		if err != nil {
			c.logf("cmd/flying failed with: %v\n", err)
		} else {
			c.logf("cmd/flying exited successfully\n")
		}
		if e := c.wc.Close(); e != nil && err == nil {
			err = e
		}
	}()
	cwd := "<nil>"
	if wd, err := os.Getwd(); err == nil {
		cwd = wd
	}
	c.logf("cmd/flying started: command=%s, args=[%v], CWD=%s\n", c.cmd[0],
		strings.Join(c.cmd[1:], ", "), cwd)
	path, err := exec.LookPath(c.cmd[0])
	if err != nil {
		return
	}
	cmd := exec.Command(path, c.cmd[1:]...)
	cmd.Stdout = rw.PrefixWriter(c.wc, prefix(path))
	cmd.Stderr = rw.PrefixWriter(c.wc, prefix(path+"] [error"))
	err = cmd.Run()
	return
}

// Run TODO
func Run(cmd []string, wc io.WriteCloser) error {
	c := newClient(cmd, wc)
	return c.run()
}
