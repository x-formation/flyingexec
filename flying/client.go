package flying

import (
	"io"
	"os/exec"
)

type client struct {
	cmd []string
	wc  io.WriteCloser
}

// NewClient assumes cmd is not empty.
func newClient(cmd []string, wc io.WriteCloser) client {
	return client{cmd: cmd, wc: wc}
}

func (c client) run() error {
	cmd := exec.Command(c.cmd[0], c.cmd[1:]...)
	// TODO(rjeczalik): Format child output?
	cmd.Stdout, cmd.Stderr = c.wc, c.wc
	return nonil(cmd.Run(), c.wc.Close())
}

// Run TODO
func Run(cmd []string, wc io.WriteCloser) error {
	c := newClient(cmd, wc)
	return c.run()
}
