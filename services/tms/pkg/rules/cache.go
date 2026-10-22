package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"
)

const (
	CacheDirEnv   = "TRENOVA_RULES_CACHE_DIR"
	cacheDirName  = "trenova-rules"
	cacheDirPerms = 0o700
)

var errUnsafeCacheDir = errors.New(
	"rules: the compilation cache directory is not private to this user",
)

func cacheBaseDir() string {
	if dir := os.Getenv(CacheDirEnv); dir != "" {
		return dir
	}
	return filepath.Join(os.TempDir(), cacheDirName)
}

func compilationCache(moduleSum string) (wazero.CompilationCache, string, error) {
	base := cacheBaseDir()
	dir := filepath.Join(base, moduleSum)
	if err := os.MkdirAll(dir, cacheDirPerms); err != nil {
		return wazero.NewCompilationCache(), dir, fmt.Errorf("create %s: %w", dir, err)
	}
	for _, path := range []string{base, dir} {
		if err := checkPrivateDir(path); err != nil {
			return wazero.NewCompilationCache(), dir, fmt.Errorf("%s: %w", path, err)
		}
	}
	cache, err := wazero.NewCompilationCacheWithDir(dir)
	if err != nil {
		return wazero.NewCompilationCache(), dir, fmt.Errorf("open %s: %w", dir, err)
	}
	return cache, dir, nil
}

func checkPrivateDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: not a plain directory", errUnsafeCacheDir)
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("%w: writable by other users", errUnsafeCacheDir)
	}
	if !ownedByCurrentUser(info) {
		return fmt.Errorf("%w: owned by another user", errUnsafeCacheDir)
	}
	return nil
}
