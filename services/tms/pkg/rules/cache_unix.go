//go:build unix

package rules

import (
	"io/fs"
	"os"
	"syscall"
)

func ownedByCurrentUser(info fs.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return int(stat.Uid) == os.Geteuid()
}
