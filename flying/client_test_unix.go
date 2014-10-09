// +build !windows

package flying

import "os/exec"

func testcmd(cmd string, args ...string) *exec.Cmd {
	return exec.Command(cmd, args...)
}
