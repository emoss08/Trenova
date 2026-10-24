//go:build !unix

package wasmcache

import "io/fs"

func ownedByCurrentUser(fs.FileInfo) bool {
	return true
}
