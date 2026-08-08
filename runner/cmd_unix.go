//go:build unix

package runner

import (
	"os/exec"
	"syscall"
)

// configureAgentCmd puts the helper in its own process group and kills the
// whole group when CommandContext cancels (so Node grandchildren do not linger).
func configureAgentCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
