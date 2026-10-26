package shapetest

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/keel"
)

const Env = "KEEL_SHAPES"

var (
	sharedOnce     sync.Once
	sharedRecorder keel.Observer
	sharedSink     *sink
	sharedErr      error
)

type sink struct {
	file *os.File
	mu   sync.Mutex
	err  error
}

func (s *sink) Write(p []byte) (int, error) {
	n, err := s.file.Write(p)
	if err != nil {
		s.mu.Lock()
		if s.err == nil {
			s.err = err
		}
		s.mu.Unlock()
	}

	return n, err
}

func (s *sink) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.err
}

func openRecorder(path string) (keel.Observer, *sink, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open the statement shapes file %s: %w", path, err)
	}
	out := &sink{file: file}

	return keel.RecordShapes(out), out, nil
}

func Path() string {
	return strings.TrimSpace(os.Getenv(Env))
}

func Observers(t testing.TB) []keel.Observer {
	t.Helper()

	path := Path()
	if path == "" {
		return nil
	}

	sharedOnce.Do(func() {
		sharedRecorder, sharedSink, sharedErr = openRecorder(path)
	})
	if sharedErr != nil {
		t.Fatal(sharedErr)
	}

	t.Cleanup(func() {
		if err := sharedSink.Err(); err != nil {
			t.Errorf("record statement shapes to %s: %v", path, err)
		}
	})

	return []keel.Observer{sharedRecorder}
}
