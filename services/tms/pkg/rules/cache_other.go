//go:build !unix

package rules

import "io/fs"

func ownedByCurrentUser(fs.FileInfo) bool {
	return true
}
