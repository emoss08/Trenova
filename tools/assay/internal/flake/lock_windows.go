//go:build !unix

package flake

import (
	"errors"
	"io/fs"
	"os"
	"time"
)

// lockFile emulates an exclusive lock with an exclusively-created sentinel
// file, retrying until it wins. A sentinel older than the stale age belongs to
// a process that died without unlocking and is taken over — without that, one
// crash would deadlock every future append.
func lockFile(path string) (func(), error) {
	const (
		retryEvery = 25 * time.Millisecond
		staleAfter = 30 * time.Second
		maxWait    = 2 * time.Minute
	)

	deadline := time.Now().Add(maxWait)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			file.Close()

			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}

		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > staleAfter {
			_ = os.Remove(path)

			continue
		}

		if time.Now().After(deadline) {
			return nil, errors.New("flake journal lock held too long; remove " + path + " if no assay is running")
		}
		time.Sleep(retryEvery)
	}
}
