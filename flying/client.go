package flying

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"bitbucket.org/kardianos/osext"
	"github.com/rjeczalik/rw"
)

// Client TODO
type Client struct {
	// Log TODO
	Log io.WriteCloser

	cmd *exec.Cmd
	wc  io.WriteCloser
}

// Start TODO
func (c *Client) Start(cmd []string) error {
	if c.cmd != nil {
		return errors.New("flying: command already started")
	}
	if len(cmd) == 0 {
		return errors.New("flying: empty command")
	}
	cwd := "<nil>"
	if wd, err := os.Getwd(); err == nil {
		cwd = wd
	}
	c.logf("flying: started with command=%s, args=[%v], CWD=%s\n", cmd[0],
		strings.Join(cmd[1:], ", "), cwd)
	path, err := exec.LookPath(cmd[0])
	if err != nil {
		return c.exit(err)
	}
	c.cmd = exec.Command(path, cmd[1:]...)
	c.cmd.Stdout = rw.PrefixWriter(c.log(), prefix(path))
	c.cmd.Stderr = rw.PrefixWriter(c.log(), prefix(path+"] [error"))
	if err = c.cmd.Start(); err != nil {
		return c.exit(err)
	}
	return nil
}

// Wait TODO
func (c *Client) Wait() error {
	if c.cmd == nil {
		return errors.New("flying: command is not running")
	}
	return c.exit(c.cmd.Wait())
}

// Run TODO
func Run(cmd []string) error {
	var c Client
	if err := c.Start(cmd); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Client) log() io.WriteCloser {
	if c.Log != nil {
		return c.Log
	}
	if c.wc != nil {
		return c.wc
	}
	name := "flying.log"
	if dir, err := osext.ExecutableFolder(); err == nil {
		name = filepath.Join(dir, name)
	}
	// TODO(rjeczalik): Rotate log each x MiB?
	wc, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0664)
	if err != nil {
		return NopCloser(ioutil.Discard)
	}
	c.wc = wc
	return c.wc
}

func (c *Client) exit(err error) error {
	if err != nil {
		c.logf("flying: failed with: %v\n", err)
	} else {
		c.logf("flying: exited successfully\n")
	}
	c.log().Close()
	return err
}

func (c *Client) logf(format string, v ...interface{}) {
	c.log().Write([]byte("[" + now() + "] " + fmt.Sprintf(format, v...)))
}

func now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func prefix(app string) func() string {
	return func() string {
		return fmt.Sprintf("[%s] [%s] ", now(), app)
	}
}
