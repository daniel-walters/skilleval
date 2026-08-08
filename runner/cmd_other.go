//go:build !unix

package runner

import "os/exec"

// configureAgentCmd is a no-op outside Unix; CommandContext default kill applies.
func configureAgentCmd(cmd *exec.Cmd) {}
