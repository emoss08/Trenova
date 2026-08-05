//go:build unix

package index

import (
	"os/exec"
	"syscall"
	"time"
)

// isolateProcessGroup puts the child in its own process group and kills the whole
// group on cancellation. `go test -c` fans out compile and link children; without
// this, cancelling assay reparents them to init and they keep burning cores.
func isolateProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}

		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 5 * time.Second
}
