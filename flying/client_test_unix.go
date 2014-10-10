// +build !windows

package flying

import "os/exec"

func command(cmd ...string) *exec.Cmd {
	return exec.Command(cmd[0], cmd[1:]...)
}
