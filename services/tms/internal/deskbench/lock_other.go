//go:build !unix

package deskbench

import "os"

func tryLock(*os.File) error { return nil }

func unlock(*os.File) error { return nil }
