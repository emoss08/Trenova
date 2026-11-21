package securefile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"
)

const (
	MaxSize      = 1 << 20
	writableMask = 0o022
	privateMask  = 0o077
)

var ErrInsecure = errors.New(
	"the file is not a plain file, is writable by others, or is a file this process keeps that is not its own and private",
)

func ReadSupplied(path string) ([]byte, error) {
	return read(path, func(info fs.FileInfo) error {
		if info.Mode().Perm()&writableMask != 0 {
			return fmt.Errorf("%w: %s is writable by its group or others", ErrInsecure, path)
		}
		return nil
	})
}

func ReadOwn(path string) ([]byte, error) {
	return readOwn(path, os.Geteuid())
}

func CheckOwnDir(path string) error {
	return checkOwnDir(path, os.Geteuid())
}

func readOwn(path string, uid int) ([]byte, error) {
	return read(path, func(info fs.FileInfo) error {
		if info.Mode().Perm()&privateMask != 0 {
			return fmt.Errorf("%w: %s is readable or writable by others", ErrInsecure, path)
		}
		return ownedBy(info, uid, path)
	})
}

func checkOwnDir(path string, uid int) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s is not a directory", ErrInsecure, path)
	}
	if info.Mode().Perm()&privateMask != 0 {
		return fmt.Errorf("%w: %s is open to others", ErrInsecure, path)
	}
	return ownedBy(info, uid, path)
}

func ownedBy(info fs.FileInfo, uid int, path string) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("%w: the owner of %s cannot be read", ErrInsecure, path)
	}
	if int(st.Uid) != uid {
		return fmt.Errorf(
			"%w: %s belongs to uid %d, not this process's %d",
			ErrInsecure,
			path,
			st.Uid,
			uid,
		)
	}
	return nil
}

func read(path string, allowed func(fs.FileInfo) error) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ELOOP) {
		return nil, fmt.Errorf("%w: %s is a link", ErrInsecure, path)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a plain file", ErrInsecure, path)
	}
	if err = allowed(info); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > MaxSize {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", ErrInsecure, path, MaxSize)
	}
	return body, nil
}
