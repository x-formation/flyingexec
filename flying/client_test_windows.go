// +build windows

package flying

import (
	"os/exec"
	"syscall"
)

func command(cmd ...string) *exec.Cmd {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return c
}
