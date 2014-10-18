package flying

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"testing"
	"time"

	"github.com/rjeczalik/rw"
)

var verbose string
var timeout = 5 * time.Second

func init() {
	for _, arg := range os.Args {
		if arg == "-test.v" || arg == "-test.v=true" {
			verbose = "-test.v"
			break
		}
	}
	if d, err := time.ParseDuration(os.Getenv("FLYING_TIMEOUT")); err == nil {
		timeout = d
	}
}

func helperCmd(cmd ...string) []string {
	return append([]string{os.Args[0], verbose, "-test.run=TestHelperProcess", "--"}, cmd...)
}

func newcmd(cmd ...string) (*exec.Cmd, Awaiter, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	var ww interface {
		io.Writer
		Awaiter
	}
	switch cmd[0] {
	case "flying":
		ww = rw.WaitWriter(buf, []byte(cmd[1]+": ready"))
	default:
		ww = struct {
			io.Writer
			Awaiter
		}{buf, Done}
	}
	cmd = helperCmd(cmd...)
	c := command(cmd[0], cmd[1:]...)
	w := io.MultiWriter(os.Stdout, ww)
	c.Stdout, c.Stderr, c.Env = w, w, []string{"TEST_HELPER_PROCESS=1"}
	return c, ww, buf
}

// Start expectes the started process to print "{{.command}} ready"
// upon successful startup.
func start(t *testing.T, cmd ...string) (*exec.Cmd, *bytes.Buffer) {
	c, w, buf := newcmd(cmd...)
	if err := c.Start(); err != nil {
		t.Fatalf("want c.Start()=nil, got %v (cmd=%v)", err, cmd)
	}
	if err := w.Wait(timeout); err != nil {
		t.Fatalf("want w.Wait(...)=nil; got %v (cmd=%v)", err, cmd)
	}
	return c, buf
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
		buf := &bytes.Buffer{}
		ww := rw.WaitWriter(buf, []byte(args[0]+": exited"))
		c := &Client{Log: NopCloser(io.MultiWriter(ww, os.Stdout))}
		if err := c.Run(helperCmd(args...)); err != nil {
			die(err)
		}
		if err := ww.Wait(timeout); err != nil {
			die(err)
		}
		return
	case "client":
		var sleep time.Duration
		if len(args) == 1 {
			if d, err := time.ParseDuration(args[0]); err == nil {
				sleep = d
			}
		}
		ch, done := make(chan os.Signal, 1), make(chan struct{})
		signal.Notify(ch, Signals...)
		go func() {
			<-ch
			close(done)
		}()
		fmt.Println("client: ready")
		select {
		case <-done:
			fmt.Println("client: caught signal")
			time.Sleep(sleep)
			fmt.Println("client: exited")
			return
		case <-time.After(timeout):
			die("client: timeout waiting for signal (cmd=client)")
		}
	default:
		die("Unknown command", cmd)
	}
}

func testClient(t *testing.T, cmds ...string) {
	if os.Getenv("APPVEYOR_BUILD_FOLDER") != "" {
		t.Skip("client TODO(rjeczalik): AppVeyor kills a build on CTRL+BREAK")
	}

	defer discardsig()() // Because Windows.

	cmd, out := start(t, cmds...)
	if err := interrupt(cmd.Process); err != nil {
		t.Fatalf("want interrupt(...)=nil; got %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("want cmd.Wait()=nil; got %v", err)
	}
	// Check whether events happened in proper order.
	s := out.String()
	i := strings.Index(s, "client: ready")
	j := strings.Index(s, "client: caught signal")
	k := strings.Index(s, "client: exited")
	if i >= j || j >= k || i == -1 {
		t.Errorf("want -1 < i=%d < j=%d < k=%d", i, j, k)
	}
}

func TestClientInterrupt(t *testing.T) {
	testClient(t, "flying", "client")
}

func TestClientInterruptWait3s(t *testing.T) {
	testClient(t, "flying", "client", "3s")
}
