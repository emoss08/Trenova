//go:build !unix

package proc

import (
	"os/exec"
	"time"
)

func Isolate(cmd *exec.Cmd) {
	cmd.WaitDelay = 5 * time.Second
}
