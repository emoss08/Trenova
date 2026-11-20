package sim

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/shared/smime"
)

type peerState struct {
	Trenova     trenovaConfig `json:"trenova"`
	Interchange int64         `json:"interchange"`
	Group       int64         `json:"group"`
	Originals   []original    `json:"originals"`
	Script      []scriptEntry `json:"script"`
	Overrides   []AckOverride `json:"overrides,omitempty"`
}

type savedState struct {
	SavedAt  time.Time            `json:"savedAt"`
	Journal  journal.Snapshot     `json:"journal"`
	Runs     runlog.Snapshot      `json:"runs"`
	Partners map[string]peerState `json:"partners"`
}

func (s *Server) restore() error {
	if s.options.StatePath == "" {
		return nil
	}
	raw, err := os.ReadFile(s.options.StatePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the saved state: %w", err)
	}
	var state savedState
	if err := sonic.Unmarshal(raw, &state); err != nil {
		return fmt.Errorf("read the saved state %s: %w", s.options.StatePath, err)
	}
	s.journal.Restore(state.Journal)
	s.runs.Restore(state.Runs)
	for name, saved := range state.Partners {
		p, ok := s.peers[name]
		if !ok {
			continue
		}
		if saved.Trenova.CertificatePEM != "" {
			certificate, certErr := smime.ParseCertificate([]byte(saved.Trenova.CertificatePEM))
			if certErr != nil {
				return fmt.Errorf("the saved Trenova certificate for %s: %w", name, certErr)
			}
			saved.Trenova.certificate = certificate
		}
		if saved.Trenova.AS2ID == "" {
			saved.Trenova.AS2ID = p.def.Partner.Trenova.ID
		}
		p.trenova = saved.Trenova
		p.script = saved.Script
		p.overrides = saved.Overrides
		p.originals.reset(saved.Originals)
		p.controls = partner.Controls{
			Interchange: partner.NewCounter(max(saved.Interchange, p.controls.Interchange.Peek())),
			Group:       partner.NewCounter(max(saved.Group, p.controls.Group.Peek())),
		}
	}
	s.saved.Store(s.version())
	return nil
}

func (s *Server) reschedule() {
	for _, m := range s.journal.Pending() {
		if p, ok := s.peers[m.Partner]; ok {
			s.scheduleDelivery(p, m)
		}
	}
}

func (s *Server) version() uint64 {
	return s.journal.Version() + s.changes.Load()
}

func (s *Server) persistLoop(ctx context.Context) {
	if s.options.StatePath == "" {
		<-ctx.Done()
		return
	}
	ticker := time.NewTicker(s.options.SnapshotEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.persist(); err != nil {
				s.logger.Error("the state could not be saved", "error", err)
			}
		}
	}
}

func (s *Server) persist() error {
	if s.options.StatePath == "" {
		return nil
	}
	version := s.version()
	if version == s.saved.Load() {
		return nil
	}
	state := savedState{
		SavedAt:  s.clock.Now().UTC(),
		Journal:  s.journal.Snapshot(),
		Runs:     s.runs.Snapshot(),
		Partners: make(map[string]peerState, len(s.peers)),
	}
	for name, p := range s.peers {
		p.mu.Lock()
		state.Partners[name] = peerState{
			Trenova:     p.trenova,
			Interchange: p.controls.Interchange.Peek(),
			Group:       p.controls.Group.Peek(),
			Script:      append([]scriptEntry(nil), p.script...),
			Overrides:   append([]AckOverride(nil), p.overrides...),
		}
		p.mu.Unlock()
		saved := state.Partners[name]
		saved.Originals = p.originals.list()
		state.Partners[name] = saved
	}
	encoded, err := sonic.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode the state: %w", err)
	}
	dir := filepath.Dir(s.options.StatePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	temp, err := os.CreateTemp(dir, ".edi-sim-state-*")
	if err != nil {
		return fmt.Errorf("save the state: %w", err)
	}
	if _, err := temp.Write(encoded); err != nil {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
		return fmt.Errorf("save the state: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(temp.Name())
		return fmt.Errorf("save the state: %w", err)
	}
	if err := os.Rename(temp.Name(), s.options.StatePath); err != nil {
		_ = os.Remove(temp.Name())
		return fmt.Errorf("save the state: %w", err)
	}
	s.saved.Store(version)
	return nil
}
