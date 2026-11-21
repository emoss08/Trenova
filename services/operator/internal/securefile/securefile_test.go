package securefile

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, []byte("secret"), 0o600))
	require.NoError(t, os.Chmod(path, mode))
	return path
}

func TestASuppliedFileReadableByEveryoneIsAccepted(t *testing.T) {
	t.Parallel()
	for _, mode := range []os.FileMode{0o444, 0o400, 0o640, 0o644, 0o600} {
		body, err := ReadSupplied(write(t, mode))
		require.NoError(t, err, "mode %o", mode)
		assert.Equal(t, "secret", string(body))
	}
}

func TestASuppliedFileOwnedBySomeoneElseIsAccepted(t *testing.T) {
	t.Parallel()
	const path = "/etc/passwd"
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Skip("no plain /etc/passwd on this machine")
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(st.Uid) == os.Geteuid() || info.Mode().Perm()&writableMask != 0 {
		t.Skip("/etc/passwd is not another user's read-only file here")
	}
	_, err = ReadSupplied(path)
	require.NoError(t, err)
}

func TestASuppliedFileOthersCanWriteIsRefused(t *testing.T) {
	t.Parallel()
	for _, mode := range []os.FileMode{0o664, 0o646, 0o666, 0o620, 0o602} {
		_, err := ReadSupplied(write(t, mode))
		require.ErrorIs(t, err, ErrInsecure, "mode %o", mode)
	}
}

func TestALinkIsRefusedInEitherClass(t *testing.T) {
	t.Parallel()
	target := write(t, 0o600)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(target, link))

	_, err := ReadSupplied(link)
	require.ErrorIs(t, err, ErrInsecure)
	_, err = ReadOwn(link)
	require.ErrorIs(t, err, ErrInsecure)
}

func TestADirectoryOrPipeIsRefusedInEitherClass(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := ReadSupplied(dir)
	require.ErrorIs(t, err, ErrInsecure)
	_, err = ReadOwn(dir)
	require.ErrorIs(t, err, ErrInsecure)

	pipe := filepath.Join(t.TempDir(), "pipe")
	require.NoError(t, syscall.Mkfifo(pipe, 0o600))
	_, err = ReadSupplied(pipe)
	require.ErrorIs(t, err, ErrInsecure)
	_, err = ReadOwn(pipe)
	require.ErrorIs(t, err, ErrInsecure)
}

func TestAMissingFileIsNotExistInEitherClass(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "absent")
	_, err := ReadSupplied(path)
	require.ErrorIs(t, err, fs.ErrNotExist)
	_, err = ReadOwn(path)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAnOwnFileIsReadOnlyWhenPrivate(t *testing.T) {
	t.Parallel()
	body, err := ReadOwn(write(t, 0o600))
	require.NoError(t, err)
	assert.Equal(t, "secret", string(body))
	_, err = ReadOwn(write(t, 0o400))
	require.NoError(t, err)

	for _, mode := range []os.FileMode{0o640, 0o604, 0o644, 0o444, 0o660} {
		_, err = ReadOwn(write(t, mode))
		require.ErrorIs(t, err, ErrInsecure, "mode %o", mode)
	}
}

func TestAnOwnFileOfAnotherUserIsRefused(t *testing.T) {
	t.Parallel()
	path := write(t, 0o600)
	_, err := readOwn(path, os.Geteuid()+1)
	require.ErrorIs(t, err, ErrInsecure)
	_, err = readOwn(path, os.Geteuid())
	require.NoError(t, err)
}

func TestAnOwnDirectoryMustBePrivateMineAndNotALink(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "state")
	require.NoError(t, os.Mkdir(dir, 0o700))
	require.NoError(t, CheckOwnDir(dir))
	require.ErrorIs(t, checkOwnDir(dir, os.Geteuid()+1), ErrInsecure)

	require.NoError(t, os.Chmod(dir, 0o750))
	require.ErrorIs(t, CheckOwnDir(dir), ErrInsecure)

	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(dir, link))
	require.ErrorIs(t, CheckOwnDir(link), ErrInsecure)
}

func TestAnOversizedFileIsRefused(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "big")
	require.NoError(t, os.WriteFile(path, make([]byte, MaxSize+1), 0o600))
	_, err := ReadSupplied(path)
	require.ErrorIs(t, err, ErrInsecure)
}
