package flake

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/zeebo/blake3"
)

const (
	journalName = "journal.jsonl"
	lockName    = "journal.lock"

	// compactThreshold is the journal size that triggers compaction on the next
	// append. Evidence is only useful in the recent past — a verdict from a
	// thousand runs ago describes code that no longer exists.
	compactThreshold = 4 << 20

	// maxPerTest bounds how many observations compaction keeps per (package,
	// test), newest first. Enough to show a pattern, small enough to stay flat.
	maxPerTest = 128
)

// Journal is the append-only observation log for one workspace. It lives under
// the machine-local cache — evidence is derived data — while the curated
// quarantine file lives in the repository. Appends from concurrent assay
// processes are serialised by a lock file, so a watch session and a mutation
// run can both record without tearing each other's lines.
type Journal struct {
	dir string
}

// OpenJournal places the workspace's journal under cacheDir, keyed by the
// workspace root so two repositories never mix evidence.
func OpenJournal(cacheDir, root string) (*Journal, error) {
	if cacheDir == "" {
		return nil, fmt.Errorf("flake journal needs a cache directory")
	}
	sum := blake3.Sum256([]byte(filepath.Clean(root)))
	dir := filepath.Join(cacheDir, "flakes", hex.EncodeToString(sum[:8]))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create flake journal directory: %w", err)
	}

	return &Journal{dir: dir}, nil
}

func (j *Journal) path() string     { return filepath.Join(j.dir, journalName) }
func (j *Journal) lockPath() string { return filepath.Join(j.dir, lockName) }

// Record appends observations, compacting first when the journal has grown
// past the threshold. Nil-safe: a nil journal records nothing, so callers
// running with --no-cache need no branching.
func (j *Journal) Record(observations []Observation) error {
	if j == nil || len(observations) == 0 {
		return nil
	}

	unlock, err := lockFile(j.lockPath())
	if err != nil {
		return fmt.Errorf("lock flake journal: %w", err)
	}
	defer unlock()

	if info, statErr := os.Stat(j.path()); statErr == nil && info.Size() > compactThreshold {
		if compactErr := j.compactLocked(); compactErr != nil {
			return compactErr
		}
	}

	var buf bytes.Buffer
	for _, o := range observations {
		line, marshalErr := marshalObservation(o)
		if marshalErr != nil {
			return fmt.Errorf("encode observation: %w", marshalErr)
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}

	file, err := os.OpenFile(j.path(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open flake journal: %w", err)
	}
	if _, err := file.Write(buf.Bytes()); err != nil {
		file.Close()

		return fmt.Errorf("append flake journal: %w", err)
	}

	return file.Close()
}

// Load reads every observation. Malformed lines are skipped rather than fatal:
// a torn line from a crashed process must not make the whole history
// unreadable.
func (j *Journal) Load() ([]Observation, error) {
	if j == nil {
		return nil, nil
	}

	unlock, err := lockFile(j.lockPath())
	if err != nil {
		return nil, fmt.Errorf("lock flake journal: %w", err)
	}
	defer unlock()

	return j.loadLocked()
}

func (j *Journal) loadLocked() ([]Observation, error) {
	file, err := os.Open(j.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("open flake journal: %w", err)
	}
	defer file.Close()

	var out []Observation
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		o, parseErr := unmarshalObservation(line)
		if parseErr != nil {
			continue
		}
		out = append(out, o)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("read flake journal: %w", scanErr)
	}

	return out, nil
}

// compactLocked rewrites the journal keeping the newest maxPerTest
// observations per (package, test). Caller holds the lock.
func (j *Journal) compactLocked() error {
	observations, err := j.loadLocked()
	if err != nil {
		return err
	}

	type key struct{ pkg, test string }
	grouped := make(map[key][]Observation)
	for _, o := range observations {
		k := key{o.Package, o.Test}
		grouped[k] = append(grouped[k], o)
	}

	kept := make([]Observation, 0, len(observations))
	for _, group := range grouped {
		sort.SliceStable(group, func(i, j int) bool { return group[i].Unix > group[j].Unix })
		if len(group) > maxPerTest {
			group = group[:maxPerTest]
		}
		kept = append(kept, group...)
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Unix < kept[j].Unix })

	var buf bytes.Buffer
	for _, o := range kept {
		line, marshalErr := marshalObservation(o)
		if marshalErr != nil {
			return fmt.Errorf("encode observation: %w", marshalErr)
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}

	tmp, err := os.CreateTemp(j.dir, ".compact-*")
	if err != nil {
		return fmt.Errorf("compact flake journal: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmpName)

		return fmt.Errorf("compact flake journal: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)

		return fmt.Errorf("compact flake journal: %w", err)
	}

	return os.Rename(tmpName, j.path())
}
