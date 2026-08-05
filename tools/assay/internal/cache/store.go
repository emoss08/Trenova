package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	entrySuffix = ".entry"

	// GraphCapacity bounds the graph cache: a handful of recent workspace states is
	// plenty. Unbounded stores pass 0.
	GraphCapacity = 32
)

type Store struct {
	dir      string
	capacity int
}

func NewStore(dir string) (*Store, error) {
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("resolve cache directory: %w", err)
		}
		dir = filepath.Join(base, "assay")
	}

	return newStore(dir, GraphCapacity)
}

// Namespace returns a sibling store under its own subdirectory with its own
// capacity. A capacity of 0 means unbounded, which is what the coverage index
// needs: it holds one entry per package, so a small cap would silently evict
// records as fast as they were written.
func (s *Store) Namespace(name string, capacity int) (*Store, error) {
	if s == nil {
		return nil, errors.New("cannot namespace a nil store")
	}

	return newStore(filepath.Join(s.dir, name), capacity)
}

func newStore(dir string, capacity int) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	return &Store{dir: dir, capacity: capacity}, nil
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
	if s.capacity <= 0 {
		return
	}

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

	if len(kept) <= s.capacity {
		return
	}

	sort.Slice(kept, func(i, j int) bool { return kept[i].modTime > kept[j].modTime })
	for _, entry := range kept[s.capacity:] {
		os.Remove(filepath.Join(s.dir, entry.name))
	}
}

var ErrDisabled = errors.New("cache disabled")
