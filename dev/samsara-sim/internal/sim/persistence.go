package sim

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
)

const (
	persistedStateVersion   = 1
	defaultStateFlushDelay  = 250 * time.Millisecond
	persistRetryDelay       = time.Second
	stateDirectoryPerm      = 0o750
	temporaryStateFileGlob  = ".tmp-*"
	persistedTimestampsForm = time.RFC3339Nano
)

type PersistenceOptions struct {
	Path       string
	FlushDelay time.Duration
	Logger     *slog.Logger
}

type PersistenceStatus struct {
	Path        string
	Restored    bool
	SavedAt     time.Time
	SeedChanged bool
}

type persistedState struct {
	Version         int               `json:"version"`
	SavedAt         string            `json:"savedAt"`
	Revision        uint64            `json:"revision"`
	SeedFingerprint string            `json:"seedFingerprint"`
	Counters        map[idSpace]int64 `json:"counters"`
	State           *Fixture          `json:"state"`
}

type statePersister struct {
	path        string
	flushDelay  time.Duration
	logger      *slog.Logger
	fingerprint string
	fileMu      sync.Mutex
	persisted   uint64
	dirty       chan struct{}
	stop        chan struct{}
	done        chan struct{}
	closeOnce   sync.Once
	closeErr    error
	writes      atomic.Uint64
}

func isTransientResource(resource Resource) bool {
	return resource == ResourceAssetLocation
}

func (p *statePersister) markDirty() {
	select {
	case p.dirty <- struct{}{}:
	default:
	}
}

func (s *Store) EnablePersistence(options PersistenceOptions) (PersistenceStatus, error) {
	path := strings.TrimSpace(options.Path)
	if path == "" {
		return PersistenceStatus{}, ErrStatePathRequired
	}
	if s.currentPersister() != nil {
		return PersistenceStatus{}, ErrPersistenceEnabled
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	flushDelay := options.FlushDelay
	if flushDelay <= 0 {
		flushDelay = defaultStateFlushDelay
	}

	if err := os.MkdirAll(filepath.Dir(path), stateDirectoryPerm); err != nil {
		return PersistenceStatus{}, fmt.Errorf("create state directory: %w", err)
	}
	fingerprint, err := s.seedFingerprint()
	if err != nil {
		return PersistenceStatus{}, err
	}

	status := PersistenceStatus{Path: path}
	loaded, found, err := readStateFile(path)
	if err != nil {
		return status, err
	}
	if found {
		s.restore(&loaded)
		status.Restored = true
		status.SeedChanged = loaded.SeedFingerprint != fingerprint
		if savedAt, parseErr := time.Parse(
			persistedTimestampsForm,
			loaded.SavedAt,
		); parseErr == nil {
			status.SavedAt = savedAt
		}
	}

	persister := &statePersister{
		path:        path,
		flushDelay:  flushDelay,
		logger:      logger,
		fingerprint: fingerprint,
		dirty:       make(chan struct{}, 1),
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}

	s.mu.Lock()
	persister.persisted = s.revision
	s.persister = persister
	s.mu.Unlock()

	go s.runPersister(persister)
	return status, nil
}

func (s *Store) ClosePersistence() error {
	persister := s.currentPersister()
	if persister == nil {
		return nil
	}
	persister.closeOnce.Do(func() {
		close(persister.stop)
		<-persister.done
		persister.closeErr = s.flushState(persister)
	})
	return persister.closeErr
}

func (s *Store) currentPersister() *statePersister {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.persister
}

func (s *Store) runPersister(persister *statePersister) {
	defer close(persister.done)

	timer := time.NewTimer(persister.flushDelay)
	timer.Stop()
	pending := false
	for {
		select {
		case <-persister.stop:
			timer.Stop()
			return
		case <-persister.dirty:
			if !pending {
				timer.Reset(persister.flushDelay)
				pending = true
			}
		case <-timer.C:
			pending = false
			if err := s.flushState(persister); err != nil {
				persister.logger.Error(
					"failed to persist simulator state",
					slog.String("path", persister.path),
					slog.String("error", err.Error()),
				)
				timer.Reset(persistRetryDelay)
				pending = true
			}
		}
	}
}

func (s *Store) flushState(persister *statePersister) error {
	persister.fileMu.Lock()
	defer persister.fileMu.Unlock()

	payload, revision, changed, err := s.encodeState(persister.persisted, persister.fingerprint)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err = writeFileAtomic(persister.path, payload); err != nil {
		return err
	}
	persister.writes.Add(1)
	persister.persisted = revision
	return nil
}

func (s *Store) encodeState(
	persistedRevision uint64,
	fingerprint string,
) (payload []byte, revision uint64, changed bool, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.revision == persistedRevision {
		return nil, s.revision, false, nil
	}

	state := s.state
	state.AssetLocation = nil
	payload, err = sonic.Marshal(persistedState{
		Version:         persistedStateVersion,
		SavedAt:         time.Now().UTC().Format(persistedTimestampsForm),
		Revision:        s.revision,
		SeedFingerprint: fingerprint,
		Counters:        s.counters,
		State:           &state,
	})
	if err != nil {
		return nil, 0, false, fmt.Errorf("encode simulator state: %w", err)
	}
	return payload, s.revision, true, nil
}

func (s *Store) restore(loaded *persistedState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := *loaded.State
	state.AssetLocation = s.state.AssetLocation
	for _, resource := range allResources {
		restored, restoredOK := state.collection(resource)
		seeded, seededOK := s.seed.collection(resource)
		if restoredOK && seededOK && *restored == nil {
			*restored = cloneRecords(*seeded)
		}
	}
	state.normalize()
	s.state = state
	s.rebuildCountersLocked()
	for space, value := range loaded.Counters {
		if value > s.counters[space] {
			s.counters[space] = value
		}
	}
	s.revision = 0
	s.version.Add(1)
}

func (s *Store) seedFingerprint() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seed := s.seed
	seed.AssetLocation = nil
	encoded, err := sonic.ConfigStd.Marshal(&seed)
	if err != nil {
		return "", fmt.Errorf("fingerprint fixture seed: %w", err)
	}
	return fmt.Sprintf("%016x", fnvHash64(string(encoded))), nil
}

func readStateFile(path string) (persistedState, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return persistedState{}, false, nil
	}
	if err != nil {
		return persistedState{}, false, fmt.Errorf("read state file %s: %w", path, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return persistedState{}, false, fmt.Errorf("%w: %s is empty", ErrStateCorrupt, path)
	}

	loaded := persistedState{}
	if err = sonic.Unmarshal(raw, &loaded); err != nil {
		return persistedState{}, false, fmt.Errorf("%w: %s: %w", ErrStateCorrupt, path, err)
	}
	if loaded.Version != persistedStateVersion {
		return persistedState{}, false, fmt.Errorf(
			"%w: %s has version %d, expected %d",
			ErrStateVersion,
			path,
			loaded.Version,
			persistedStateVersion,
		)
	}
	if loaded.State == nil {
		return persistedState{}, false, fmt.Errorf(
			"%w: %s has no state section",
			ErrStateCorrupt,
			path,
		)
	}
	return loaded, true, nil
}

func writeFileAtomic(path string, payload []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+temporaryStateFileGlob)
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}
	temporaryPath := temporary.Name()
	if err = writeAndSync(temporary, payload); err != nil {
		return errors.Join(err, removeIfExists(temporaryPath))
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return errors.Join(
			fmt.Errorf("replace state file: %w", err),
			removeIfExists(temporaryPath),
		)
	}
	return syncDirectory(directory)
}

func writeAndSync(file *os.File, payload []byte) error {
	if _, err := file.Write(payload); err != nil {
		return errors.Join(fmt.Errorf("write temporary state file: %w", err), file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync temporary state file: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary state file: %w", err)
	}
	return nil
}

func syncDirectory(directory string) error {
	handle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open state directory: %w", err)
	}
	if err = handle.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync state directory: %w", err), handle.Close())
	}
	if err = handle.Close(); err != nil {
		return fmt.Errorf("close state directory: %w", err)
	}
	return nil
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

func removeStateFile(path string) error {
	if err := removeIfExists(path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}
