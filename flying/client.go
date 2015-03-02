package flying

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kardianos/osext"
	"github.com/rjeczalik/rw"
)

var nl = "\n"

var errAlreadyStarted = errors.New("flying: command already started")
var errEmptyCommnd = errors.New("flying: empty command")
var errNotRunning = errors.New("flying: command is not running")

// SourceDir TODO
var SourceDir string

func init() {
	if dir, err := osext.ExecutableFolder(); err == nil {
		SourceDir = dir
	}
	if runtime.GOOS == "windows" {
		nl = "\r\n"
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

	// Env TODO
	Env []string

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
	c.printf("flying: started with command=%s, args=[%v], CWD=%s", cmdname,
		strings.Join(cmd[1:], ", "), cwd)
	path, err := exec.LookPath(cmd[0])
	if err != nil {
		return c.exit(err)
	}
	if abspath, err := filepath.Abs(path); err == nil {
		path = abspath
	}
	c.printf("flying: %s is %s", cmdname, path)
	c.cmd = command(path, cmd[1:]...)
	c.cmd.Env = mergenv(os.Environ(), c.Env...)
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
		return nopCloser(ioutil.Discard)
	}
	c.wc = wc
	return c.wc
}

func (c *Client) exit(err error) error {
	if err != nil {
		c.printf("flying: failed with: %v", err)
	} else {
		c.printf("flying: exited successfully")
	}
	c.log().Close()
	return err
}

func (c *Client) printf(format string, v ...interface{}) {
	c.log().Write([]byte("[" + now() + "] " + fmt.Sprintf(format, v...) + nl))
}

func runconsole(c *Client, cmd []string) error {
	ch, errch := make(chan os.Signal, 1), make(chan error, 1)
	signal.Notify(ch, Signals...)
	if err := c.Start(cmd); err != nil {
		return err
	}
	go func() {
		for _ = range ch {
			errch <- c.Interrupt()
		}
	}()
	err := c.Wait()
	select {
	case e := <-errch:
		err = nonil(err, e)
	default:
	}
	return err
}

var defaultClient Client

// Run TODO
func Run(cmd []string) error {
	return defaultClient.Run(cmd)
}
