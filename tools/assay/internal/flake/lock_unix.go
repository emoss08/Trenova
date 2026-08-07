//go:build unix

package flake

import (
	"os"
	"syscall"
)

// lockFile takes an exclusive advisory lock, blocking until it is granted.
// flock is inherited-safe and self-releasing on process death, which is what
// makes a crashed watch session harmless to the next writer.
func lockFile(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()

		return nil, err
	}

	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
	}, nil
}
