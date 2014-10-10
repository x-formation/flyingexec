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

var errAlreadyStarted = errors.New("flying: command already started")
var errEmptyCommnd = errors.New("flying: empty command")
var errNotRunning = errors.New("flying: command is not running")

// SourceDir TODO
var SourceDir string

func init() {
	if dir, err := osext.ExecutableFolder(); err == nil {
		SourceDir = dir
	}
}

func now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func prefix(app string) func() string {
	return func() string {
		return fmt.Sprintf("[%s] [%s] ", now(), app)
	}
}

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
		return errAlreadyStarted
	}
	if len(cmd) == 0 {
		return errEmptyCommnd
	}
	cmdname := filepath.Base(cmd[0])
	cwd := "<nil>"
	if wd, err := os.Getwd(); err == nil {
		cwd = wd
	}
	c.logf("flying: started with command=%s, args=[%v], CWD=%s\n", cmdname,
		strings.Join(cmd[1:], ", "), cwd)
	path, err := exec.LookPath(cmd[0])
	if err != nil {
		return c.exit(err)
	}
	c.logf("flying: %s is %s\n", cmdname, path)
	c.cmd = exec.Command(path, cmd[1:]...)
	c.cmd.Stdout = rw.PrefixWriter(c.log(), prefix(cmdname))
	c.cmd.Stderr = rw.PrefixWriter(c.log(), prefix(cmdname+"] [error"))
	if err = c.cmd.Start(); err != nil {
		return c.exit(err)
	}
	return nil
}

// Interrupt TODO
func (c *Client) Interrupt() error {
	if c.cmd == nil {
		return errNotRunning
	}
	// Interrupt is a wrapper for (*os.Process).Signal(os.Interrupt) - remove it
	// after golang.org/issue/6720.
	return interrupt(c.cmd.Process)
}

// Wait TODO
func (c *Client) Wait() error {
	if c.cmd == nil {
		return errNotRunning
	}
	return c.exit(c.cmd.Wait())
}

// Run TODO
func (c *Client) Run(cmd []string) error {
	return run(c, cmd)
}

func (c *Client) log() io.WriteCloser {
	if c.Log != nil {
		return c.Log
	}
	if c.wc != nil {
		return c.wc
	}
	name := filepath.Join(SourceDir, "flying.log")
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

var defaultClient Client

// Run TODO
func Run(cmd []string) error {
	return defaultClient.Run(cmd)
}
