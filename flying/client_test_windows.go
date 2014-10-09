// +build windows

package flying

import (
	"os/exec"
	"syscall"
)

func testcmd(cmd string, args ...string) *exec.Cmd {
	c := exec.Command(cmd, args...)
	c.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return c
}
