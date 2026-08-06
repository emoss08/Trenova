package watch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/overlay"
	"github.com/emoss08/assay/internal/proc"
)

// BinaryCache keeps one pre-linked test binary per package for the life of a
// watch session.
//
// The savings this buys is the whole point of watch mode: `go test` pays flag
// parsing, cache probing and a relink on every invocation — around a second even
// when nothing changed — while a warm binary starts in milliseconds. Staleness is
// decided by the same fingerprint the index uses, computed over the working
// tree's content digests, so a binary survives exactly as long as its package's
// dependency closure is byte-identical.
type BinaryCache struct {
	dir string

	mu      sync.Mutex
	entries map[string]binEntry
}

type binEntry struct {
	fingerprint cache.Fingerprint
	path        string
}

func NewBinaryCache() (*BinaryCache, error) {
	dir, err := os.MkdirTemp("", "assay-watch-")
	if err != nil {
		return nil, fmt.Errorf("create watch binary directory: %w", err)
	}

	return &BinaryCache{dir: dir, entries: make(map[string]binEntry)}, nil
}

func (c *BinaryCache) Close() {
	if c != nil && c.dir != "" {
		_ = os.RemoveAll(c.dir)
	}
}

type BuildRequest struct {
	Root             string
	Env              []string
	Tags             []string
	ImportPath       string
	Fingerprint      cache.Fingerprint
	BuildParallelism int
}

// Ensure returns a runnable test binary for the package, building one only when
// the cached binary's fingerprint no longer matches the working tree. The empty
// path with a nil error means the package has no test binary at all.
func (c *BinaryCache) Ensure(ctx context.Context, req BuildRequest) (string, bool, error) {
	c.mu.Lock()
	entry, cached := c.entries[req.ImportPath]
	c.mu.Unlock()

	if cached && entry.fingerprint == req.Fingerprint {
		return entry.path, true, nil
	}

	output := filepath.Join(c.dir, binaryName(req.ImportPath))
	staging := output + ".tmp"

	args := []string{"test", "-c", "-o", staging}
	if req.BuildParallelism > 0 {
		args = append(args, "-p", strconv.Itoa(req.BuildParallelism))
	}
	if len(req.Tags) > 0 {
		args = append(args, "-tags", strings.Join(req.Tags, ","))
	}
	args = append(args, req.ImportPath)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = req.Root
	if len(req.Env) > 0 {
		cmd.Env = req.Env
	}
	proc.Isolate(cmd)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("compile %s: %w: %s",
			req.ImportPath, err, strings.TrimSpace(out.String()))
	}

	if _, err := os.Stat(staging); err != nil {
		// A package with no test files builds successfully and produces nothing.
		c.remember(req.ImportPath, req.Fingerprint, "")

		return "", false, nil
	}

	// The rename is what makes a cancelled or failed build harmless: a truncated
	// binary can never sit at the final path where a later fingerprint match
	// would happily reuse it.
	if err := os.Rename(staging, output); err != nil {
		return "", false, fmt.Errorf("publish %s binary: %w", req.ImportPath, err)
	}

	c.remember(req.ImportPath, req.Fingerprint, output)

	return output, false, nil
}

func binaryName(importPath string) string {
	return overlay.UniqueName(importPath) + ".test"
}

func (c *BinaryCache) remember(importPath string, fp cache.Fingerprint, path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[importPath] = binEntry{fingerprint: fp, path: path}
}

// RunResult is one package's test execution within a cycle. Skipped means no
// test binary exists — a package whose tests are all behind a build tag — which
// is a different claim from "ran and passed" and must never be counted as one.
type RunResult struct {
	ImportPath string
	Tests      []string
	Passed     bool
	Skipped    bool
	Output     []byte
	Duration   time.Duration
	Reused     bool
	Err        error
}

type RunRequest struct {
	Binary  string
	Dir     string
	Env     []string
	Pattern string
	Timeout time.Duration
}

// RunTests executes a warm binary against a -test.run pattern. A failing test is
// a result, not an error; errors are reserved for the binary not running at all.
func RunTests(ctx context.Context, req RunRequest) (bool, []byte, error) {
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	args := []string{"-test.count=1"}
	if req.Timeout > 0 {
		// The binary's own timeout fires first, so a hang dies with a goroutine
		// dump instead of a bare SIGKILL from the context deadline.
		args = append(args, "-test.timeout", (req.Timeout * 9 / 10).String())
	}
	if req.Pattern != "" {
		args = append(args, "-test.run", req.Pattern)
	}

	cmd := exec.CommandContext(ctx, req.Binary, args...)
	cmd.Dir = req.Dir
	if len(req.Env) > 0 {
		cmd.Env = req.Env
	}
	proc.Isolate(cmd)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err == nil {
		return true, out.Bytes(), nil
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return false, out.Bytes(), fmt.Errorf("timed out after %s", req.Timeout)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() >= 0 {
		return false, out.Bytes(), nil
	}

	return false, out.Bytes(), err
}
