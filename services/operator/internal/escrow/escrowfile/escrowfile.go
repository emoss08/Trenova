package escrowfile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/operator/internal/atomicfile"
	"github.com/emoss08/trenova/operator/internal/escrow"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const dirMode = 0o700

type Store struct {
	dir    string
	cp     escrow.ControlPlane
	logger *slog.Logger
}

type Params struct {
	Dir          string
	ControlPlane escrow.ControlPlane
	Logger       *slog.Logger
}

func New(p *Params) (*Store, error) {
	switch {
	case p.Dir == "":
		return nil, errors.New("escrowfile: the secrets directory is required")
	case p.ControlPlane == nil:
		return nil, errors.New("escrowfile: the control plane is required")
	case p.Logger == nil:
		return nil, errors.New("escrowfile: a logger is required")
	}
	return &Store{dir: p.Dir, cp: p.ControlPlane, logger: p.Logger}, nil
}

func (s *Store) Ensure(ctx context.Context) error {
	held, err := s.read()
	if err != nil {
		return err
	}
	problem := escrow.Validate(escrow.FromData(held))
	if problem == nil {
		return nil
	}
	if len(held) > 0 {
		s.logger.Warn(
			"the escrowed secrets on disk are incomplete or invalid; restoring them from the control plane",
			"dir",
			s.dir,
			"reason",
			problem.Error(),
		)
	}
	fetched, err := s.cp.Secrets(ctx)
	if err != nil {
		return fmt.Errorf("fetch the escrowed secrets: %w", err)
	}
	if err = escrow.Validate(fetched); err != nil {
		return fmt.Errorf("the control plane's escrowed secrets: %w", err)
	}
	if err = s.write(fetched); err != nil {
		return err
	}
	s.logger.Info("escrowed secrets written", "dir", s.dir)
	return nil
}

func (s *Store) read() (map[string][]byte, error) {
	out := make(map[string][]byte, len(escrow.Keys))
	for _, key := range escrow.Keys {
		body, err := os.ReadFile(filepath.Join(s.dir, key))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read the escrowed secret %s: %w", key, err)
		}
		out[key] = []byte(strings.TrimSpace(string(body)))
	}
	return out, nil
}

func (s *Store) write(v instanceapi.InstanceSecrets) error {
	if err := os.MkdirAll(s.dir, dirMode); err != nil {
		return fmt.Errorf("make the secrets directory: %w", err)
	}
	if err := os.Chmod(s.dir, dirMode); err != nil {
		return fmt.Errorf("restrict the secrets directory: %w", err)
	}
	for key, value := range escrow.ToData(nil, v) {
		if err := atomicfile.Write(filepath.Join(s.dir, key), value); err != nil {
			return err
		}
	}
	return nil
}
