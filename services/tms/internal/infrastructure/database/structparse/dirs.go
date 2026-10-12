package structparse

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func PackageDirs(pattern string) ([]string, error) {
	roots, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("expand %q: %w", pattern, err)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("no domain directory matches %q", pattern)
	}

	dirs := make([]string, 0, 160)
	for _, root := range roots {
		entries, readErr := os.ReadDir(root)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", root, readErr)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, filepath.Join(root, entry.Name()))
			}
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		return filepath.Base(dirs[i]) < filepath.Base(dirs[j])
	})

	return dirs, nil
}
