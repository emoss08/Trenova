package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
)

type snapshot struct {
	execDir    string
	backupDir  string
	modelsPath string
	models     []byte
	hadModels  bool
	moved      []string
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

	names, err := generatedEntries(execDir)
	if err != nil {
		return nil, errors.Join(err, os.RemoveAll(backupDir))
	}
	for _, name := range names {
		if err = os.Rename(filepath.Join(execDir, name), filepath.Join(backupDir, name)); err != nil {
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
		if err = os.Rename(filepath.Join(s.backupDir, name), filepath.Join(s.execDir, name)); err != nil {
			return fmt.Errorf("restore %s from %s: %w", name, s.backupDir, err)
		}
	}

	if s.modelsPath != "" {
		if s.hadModels {
			err = os.WriteFile(s.modelsPath, s.models, 0o644)
		} else {
			err = os.Remove(s.modelsPath)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		}
		if err != nil {
			return fmt.Errorf("restore models: %w", err)
		}
	}

	return s.discard()
}

func (s *snapshot) discard() error {
	if err := os.RemoveAll(s.backupDir); err != nil {
		return fmt.Errorf("remove backup %s: %w", s.backupDir, err)
	}
	return nil
}
