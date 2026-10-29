//go:build !unix

package benchnamespace

import "os"

func tryLock(*os.File) error { return nil }

func unlock(*os.File) error { return nil }
