package runlock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const fileMode = 0o600

var ErrHeld = errors.New("another tenant agent holds the lock; only one may run a server")

type Lock struct {
	file *os.File
}

func Acquire(dir string) (*Lock, error) {
	path := filepath.Join(dir, "tenant-agent.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, fileMode)
	if err != nil {
		return nil, fmt.Errorf("open the lock file: %w", err)
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrHeld
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return &Lock{file: f}, nil
}

func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	return errors.Join(err, l.file.Close())
}
