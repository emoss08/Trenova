package deskbench

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrNamespaceBusy = errors.New("another bench is running in this namespace")

type namespaceLock struct {
	file *os.File
}

func lockNamespace(dir, namespace string) (*namespaceLock, error) {
	if dir == "" {
		return &namespaceLock{}, nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create %s: %w", dir, err)
	}

	path := filepath.Join(dir, namespace+".lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err = tryLock(file); err != nil {
		holder, _ := os.ReadFile(path)
		_ = file.Close()

		return nil, fmt.Errorf(
			"%w: %q is held by pid %s; its workers would take this run's turns and run them "+
				"on its own build. Wait for it, or pass --namespace with another name",
			ErrNamespaceBusy, namespace, strings.TrimSpace(string(holder)),
		)
	}
	if err = file.Truncate(0); err == nil {
		_, err = file.WriteAt([]byte(strconv.Itoa(os.Getpid())), 0)
	}
	if err != nil {
		_ = unlock(file)
		_ = file.Close()

		return nil, fmt.Errorf("record the bench's pid in %s: %w", path, err)
	}

	return &namespaceLock{file: file}, nil
}

func (l *namespaceLock) release() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := unlock(l.file)
	closeErr := l.file.Close()
	l.file = nil

	return errors.Join(unlockErr, closeErr)
}
