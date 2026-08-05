package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	entrySuffix = ".graph"
	maxEntries  = 32
)

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("resolve cache directory: %w", err)
		}
		dir = filepath.Join(base, "assay")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	return &Store{dir: dir}, nil
}

func (s *Store) Dir() string { return s.dir }

func (s *Store) Get(key Fingerprint) ([]byte, bool) {
	payload, err := os.ReadFile(s.path(key))
	if err != nil {
		return nil, false
	}

	return payload, true
}

func (s *Store) Put(key Fingerprint, payload []byte) error {
	target := s.path(key)

	tmp, err := os.CreateTemp(s.dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create cache temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		os.Remove(tmpName)

		return fmt.Errorf("write cache entry: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)

		return fmt.Errorf("close cache entry: %w", err)
	}

	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)

		return fmt.Errorf("commit cache entry: %w", err)
	}

	s.prune()

	return nil
}

func (s *Store) path(key Fingerprint) string {
	return filepath.Join(s.dir, key.String()+entrySuffix)
}

func (s *Store) prune() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}

	type aged struct {
		name    string
		modTime int64
	}

	kept := make([]aged, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != entrySuffix {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		kept = append(kept, aged{name: entry.Name(), modTime: info.ModTime().UnixNano()})
	}

	if len(kept) <= maxEntries {
		return
	}

	sort.Slice(kept, func(i, j int) bool { return kept[i].modTime > kept[j].modTime })
	for _, entry := range kept[maxEntries:] {
		os.Remove(filepath.Join(s.dir, entry.name))
	}
}

var ErrDisabled = errors.New("cache disabled")
