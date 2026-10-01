package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
)

type snapshot struct {
	execDir     string
	backupDir   string
	modelsPath  string
	models      []byte
	hadModels   bool
	moved       []string
	resolverDir string
	resolvers   map[string][]byte
}

func takeSnapshot(cfg *config.Config) (*snapshot, error) {
	execDir := cfg.Exec.Dir()
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", execDir, err)
	}

	backupDir, err := os.MkdirTemp(filepath.Dir(execDir), ".gqlexec-backup-*")
	if err != nil {
		return nil, fmt.Errorf("create backup: %w", err)
	}
	snap := &snapshot{execDir: execDir, backupDir: backupDir}

	if cfg.Model.IsDefined() {
		snap.modelsPath = cfg.Model.Filename
		snap.models, err = os.ReadFile(snap.modelsPath)
		switch {
		case err == nil:
			snap.hadModels = true
		case !errors.Is(err, os.ErrNotExist):
			return nil, errors.Join(fmt.Errorf("read models: %w", err), os.RemoveAll(backupDir))
		}
	}

	snap.resolverDir = cfg.Resolver.Dir()
	if snap.resolvers, err = readGoFiles(snap.resolverDir); err != nil {
		return nil, errors.Join(err, os.RemoveAll(backupDir))
	}

	names, err := generatedEntries(execDir)
	if err != nil {
		return nil, errors.Join(err, os.RemoveAll(backupDir))
	}
	for _, name := range names {
		if err = os.Rename(
			filepath.Join(execDir, name),
			filepath.Join(backupDir, name),
		); err != nil {
			return nil, errors.Join(fmt.Errorf("back up %s: %w", name, err), snap.restore())
		}
		snap.moved = append(snap.moved, name)
	}

	return snap, nil
}

func generatedEntries(execDir string) ([]string, error) {
	entries, err := os.ReadDir(execDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", execDir, err)
	}
	var names []string
	for _, e := range entries {
		if (e.IsDir() && e.Name() == shardSuffix) ||
			(!e.IsDir() && strings.HasSuffix(e.Name(), ".generated.go")) {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (s *snapshot) restore() error {
	names, err := generatedEntries(s.execDir)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err = os.RemoveAll(filepath.Join(s.execDir, name)); err != nil {
			return fmt.Errorf("remove partial %s: %w", name, err)
		}
	}
	for _, name := range s.moved {
		if err = os.Rename(
			filepath.Join(s.backupDir, name),
			filepath.Join(s.execDir, name),
		); err != nil {
			return fmt.Errorf("restore %s from %s: %w", name, s.backupDir, err)
		}
	}

	if err = s.restoreResolvers(); err != nil {
		return err
	}

	if err = s.restoreModels(); err != nil {
		return err
	}

	return s.discard()
}

func (s *snapshot) restoreModels() error {
	if s.modelsPath == "" {
		return nil
	}
	if s.hadModels {
		return writeFile(s.modelsPath, s.models)
	}
	if err := os.Remove(s.modelsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("restore models: %w", err)
	}
	return nil
}

func (s *snapshot) discard() error {
	if err := os.RemoveAll(s.backupDir); err != nil {
		return fmt.Errorf("remove backup %s: %w", s.backupDir, err)
	}
	return nil
}

func readGoFiles(dir string) (map[string][]byte, error) {
	root, err := os.OpenRoot(dir)
	if errors.Is(err, os.ErrNotExist) {
		return map[string][]byte{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dir, err)
	}
	defer root.Close()

	files := map[string][]byte{}
	tree := root.FS()
	err = fs.WalkDir(tree, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, readErr := fs.ReadFile(tree, path)
		if readErr != nil {
			return readErr
		}
		files[filepath.Join(dir, filepath.FromSlash(path))] = content
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read resolvers: %w", err)
	}
	return files, nil
}

func (s *snapshot) restoreResolvers() error {
	current, err := readGoFiles(s.resolverDir)
	if err != nil {
		return err
	}
	for path, content := range current {
		original, existed := s.resolvers[path]
		switch {
		case !existed:
			if err = os.Remove(path); err != nil {
				return fmt.Errorf("remove %s: %w", path, err)
			}
			if entries, readErr := os.ReadDir(
				filepath.Dir(path),
			); readErr == nil &&
				len(entries) == 0 {
				if err = os.Remove(filepath.Dir(path)); err != nil {
					return fmt.Errorf("remove %s: %w", filepath.Dir(path), err)
				}
			}
		case !bytes.Equal(original, content):
			if err = writeFile(path, original); err != nil {
				return err
			}
		}
	}
	for path, original := range s.resolvers {
		if _, ok := current[path]; ok {
			continue
		}
		if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err = writeFile(path, original); err != nil {
			return err
		}
	}
	return nil
}
