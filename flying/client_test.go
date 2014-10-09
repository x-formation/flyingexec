package flying

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"testing"
	"time"

	"github.com/rjeczalik/rw"
)

var timeout = 5 * time.Second

func command(cmd ...string) (msg string, realcmd []string) {
	msg = cmd[0] + " ready"
	realcmd = append([]string{os.Args[0], "-test.run=TestHelperProcess", "--"}, cmd...)
	return
}

// Start expectes the started process to print "{{.command}} ready"
// upon successful startup.
func start(t *testing.T, cmd ...string) (c *exec.Cmd, buf *bytes.Buffer) {
	msg, cmd := command(cmd...)
	c, buf = testcmd(cmd[0], cmd[1:]...), &bytes.Buffer{}
	w := rw.WaitWriter(buf, []byte(msg))
	mw := io.MultiWriter(os.Stdout, w)
	c.Stdout, c.Stderr, c.Env = mw, mw, []string{"TEST_HELPER_PROCESS=1"}
	if err := c.Start(); err != nil {
		t.Fatalf("want c.Start()=nil, got %v (cmd=%v)", err, cmd)
	}
	if err := w.Wait(timeout); err != nil {
		t.Fatalf("want w.Wait(...)=nil; got %v (cmd=%v)", err, cmd)
	}
	return
}

func die(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(2)
}

// Based on the code stolen from $GOROOT/src/os/exec/exec_test.go.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("TEST_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		die("No command")
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	// Each helper command started by a helper flying must print "{{.command}} ready"
	// to os.Stdout upon successful startup. Otherwise it's going to timeout.
	case "flying":
		msg, cmd := command(args...)
		w := rw.WaitWriter(ioutil.Discard, []byte(msg))
		ch, err := make(chan os.Signal, 1), make(chan error)
		c := &Client{Log: NopCloser(io.MultiWriter(os.Stdout, w))}
		signal.Notify(ch, Signals...)
		if err := c.Start(cmd); err != nil {
			die(err)
		}
		if err := w.Wait(timeout); err != nil {
			die(err)
		}
		go func() {
			<-ch
			err <- nonil(c.Interrupt(), c.Interrupt())
		}()
		fmt.Println("flying ready")
		if err := c.Wait(); err != nil {
			die(err)
		}
		if err := <-err; err != nil {
			die(err)
		}
		return
	case "TestClientInterrupt":
		ch, done := make(chan os.Signal, 1), make(chan struct{})
		signal.Notify(ch, Signals...)
		go func() {
			<-ch
			close(done)
		}()
		fmt.Println("TestClientInterrupt ready")
		select {
		case <-done:
			fmt.Println("child interrupted")
			return
		case <-time.After(timeout):
			die("TestHelperProcess: timeout waiting for signal (cmd=TestClientInterruptChild)")
		}
	default:
		die("Unknown command", cmd)
	}
}

func TestClientInterrupt(t *testing.T) {
	if os.Getenv("APPVEYOR_BUILD_FOLDER") != "" {
		t.Skip("TestClientInterrupt TODO(rjeczalik): AppVeyor kills a build on CTRL+BREAK")
	}

	cmd, out := start(t, "flying", "TestClientInterrupt")
	if err := interrupt(cmd.Process); err != nil {
		t.Fatalf("want interrupt(...)=nil; got %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("want cmd.Wait()=nil; got %v", err)
	}
	// Check whether events happened in proper order.
	s := out.String()
	i := strings.Index(s, "TestClientInterrupt ready")
	j := strings.Index(s, "flying ready")
	k := strings.Index(s, "child interrupted")
	if i >= j || j >= k || i == -1 {
		t.Errorf("want i=%d < j=%d < k=%d and i != -1", i, j, k)
	}
}
