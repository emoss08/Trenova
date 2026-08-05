//go:build !unix

package index

import (
	"os/exec"
	"time"
)

func isolateProcessGroup(cmd *exec.Cmd) {
	cmd.WaitDelay = 5 * time.Second
}
