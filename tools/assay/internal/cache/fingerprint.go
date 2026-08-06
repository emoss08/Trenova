package cache

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/sourcegraph/conc/pool"
	"github.com/zeebo/blake3"
)

const schemaVersion = "assay-graph-v1"

type Fingerprint [32]byte

func (f Fingerprint) String() string {
	return hex.EncodeToString(f[:])
}

type Digest [32]byte

type FileDigest struct {
	AbsPath string
	RelPath string
	Digest  Digest
}

type Manifest struct {
	Root   string
	Key    Fingerprint
	Files  []FileDigest
	byPath map[string]Digest
}

func (m *Manifest) DigestFor(absPath string) (Digest, bool) {
	d, ok := m.byPath[filepath.Clean(absPath)]

	return d, ok
}

// Digests renders the manifest as the rel-path digest map the index
// fingerprinter consumes, describing the working tree as it is right now.
func (m *Manifest) Digests() map[string]string {
	out := make(map[string]string, len(m.Files))
	for _, file := range m.Files {
		out[file.RelPath] = fmt.Sprintf("%x", file.Digest)
	}

	return out
}

type Inputs struct {
	Root string
	Tags []string
	Env  []string
}

var skippedDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
}

var moduleFiles = map[string]struct{}{
	"go.mod":      {},
	"go.sum":      {},
	"go.work":     {},
	"go.work.sum": {},
}

var toolchainEnv = []string{"GOOS", "GOARCH", "GOFLAGS", "GOEXPERIMENT", "CGO_ENABLED", "GOWORK"}

func Compute(ctx context.Context, in Inputs) (Fingerprint, error) {
	manifest, err := Scan(ctx, in)
	if err != nil {
		return Fingerprint{}, err
	}

	return manifest.Key, nil
}

func Scan(ctx context.Context, in Inputs) (*Manifest, error) {
	root, err := filepath.Abs(in.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	paths, err := collect(root)
	if err != nil {
		return nil, err
	}

	digests, err := hashFiles(ctx, paths)
	if err != nil {
		return nil, err
	}

	manifest := &Manifest{
		Root:   root,
		Files:  make([]FileDigest, 0, len(paths)),
		byPath: make(map[string]Digest, len(paths)),
	}

	h := blake3.New()
	writeField(h, schemaVersion)
	writeField(h, BuildIdentity())
	writeField(h, root)
	writeField(h, runtime.Version())

	tags := append([]string(nil), in.Tags...)
	sort.Strings(tags)
	for _, tag := range tags {
		writeField(h, tag)
	}

	for _, key := range toolchainEnv {
		writeField(h, key+"="+lookupEnv(in.Env, key))
	}

	for i, path := range paths {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil, fmt.Errorf("relativise %s: %w", path, relErr)
		}
		rel = filepath.ToSlash(rel)

		writeField(h, rel)
		h.Write(digests[i][:])

		manifest.Files = append(manifest.Files, FileDigest{
			AbsPath: path,
			RelPath: rel,
			Digest:  digests[i],
		})
		manifest.byPath[path] = digests[i]
	}

	copy(manifest.Key[:], h.Sum(nil))

	return manifest, nil
}

func BuildIdentity() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown" + executableIdentity()
	}

	parts := []string{info.Main.Path, info.Main.Version}
	var revision string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision", "vcs.modified":
			parts = append(parts, setting.Key+"="+setting.Value)
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
		}
	}

	identity := strings.Join(parts, " ")
	if revision == "" || info.Main.Version == "" || info.Main.Version == "(devel)" {
		identity += executableIdentity()
	}

	return identity
}

func executableIdentity() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	return " exe=" + strconv.FormatInt(info.Size(), 10) + ":" + strconv.FormatInt(info.ModTime().UnixNano(), 10)
}

func writeField(h io.Writer, value string) {
	h.Write([]byte(value))
	h.Write([]byte{0})
}

func lookupEnv(env []string, key string) string {
	prefix := key + "="
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(env[i], prefix) {
			return env[i][len(prefix):]
		}
	}
	if len(env) > 0 {
		return ""
	}

	return os.Getenv(key)
}

func collect(root string) ([]string, error) {
	var out []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root {
				return walkErr
			}
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}

			return nil
		}

		if entry.IsDir() {
			if path == root {
				return nil
			}
			name := entry.Name()
			if _, skip := skippedDirs[name]; skip || strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}

			return nil
		}

		if isRelevant(entry.Name()) || underTestdata(path) {
			out = append(out, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}

	sort.Strings(out)

	return out, nil
}

func isRelevant(name string) bool {
	if _, ok := moduleFiles[name]; ok {
		return true
	}

	return strings.HasSuffix(name, ".go")
}

// underTestdata includes test fixtures in the fingerprint. A test binary bakes
// testdata in twice over — tests read it at runtime and //go:embed compiles it
// in — so a fixture edit that moved no .go digest used to leave a stale warm
// binary passing against data that no longer exists. Embedded assets outside
// testdata directories remain outside the fingerprint; that residual is
// documented rather than chased.
func underTestdata(path string) bool {
	sep := string(filepath.Separator)

	return strings.Contains(path, sep+"testdata"+sep)
}

// hashers keeps a blake3 state and a read buffer per live worker rather than per
// file. Allocating them per file cost 107 MB of garbage on this workspace against
// 2.7 MB reusing them, so the pool is load-bearing, not tidiness.
var hashers = sync.Pool{
	New: func() any {
		return &hashState{hasher: blake3.New(), buf: make([]byte, 64<<10)}
	},
}

type hashState struct {
	hasher *blake3.Hasher
	buf    []byte
}

func hashFiles(ctx context.Context, paths []string) ([]Digest, error) {
	digests := make([]Digest, len(paths))
	if len(paths) == 0 {
		return digests, nil
	}

	workers := min(runtime.GOMAXPROCS(0), len(paths))

	hashing := pool.New().
		WithMaxGoroutines(workers).
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	for i := range paths {
		hashing.Go(func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			state, _ := hashers.Get().(*hashState)
			defer hashers.Put(state)

			return hashFile(state.hasher, state.buf, paths[i], &digests[i])
		})
	}

	if err := hashing.Wait(); err != nil {
		return nil, err
	}

	return digests, nil
}

func hashFile(hasher *blake3.Hasher, buf []byte, path string, out *Digest) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	hasher.Reset()
	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			hasher.Write(buf[:n])
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}

			return fmt.Errorf("read %s: %w", path, readErr)
		}
	}

	copy(out[:], hasher.Sum(out[:0]))

	return nil
}
