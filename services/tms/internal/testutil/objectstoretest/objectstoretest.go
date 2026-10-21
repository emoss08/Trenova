package objectstoretest

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type Store struct {
	kernelports.Client

	mu       sync.Mutex
	objects  map[string][]byte
	metadata map[string]map[string]string
	attempts int
	versions int
}

func New() *Store {
	return &Store{objects: map[string][]byte{}, metadata: map[string]map[string]string{}}
}

func (s *Store) Upload(
	_ context.Context,
	params *kernelports.UploadParams,
) (*kernelports.FileInfo, error) {
	s.mu.Lock()
	s.attempts++
	s.mu.Unlock()

	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	if params.Size >= 0 && int64(len(body)) != params.Size {
		return nil, io.ErrUnexpectedEOF
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.versions++
	s.objects[params.Key] = body
	s.metadata[params.Key] = params.Metadata
	return &kernelports.FileInfo{
		Key:          params.Key,
		Size:         int64(len(body)),
		ContentType:  params.ContentType,
		LastModified: time.Now(),
		Metadata:     params.Metadata,
		VersionID:    "v" + strconv.Itoa(s.versions),
	}, nil
}

func (s *Store) Download(_ context.Context, key string) (*kernelports.DownloadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, ok := s.objects[key]
	if !ok {
		return nil, kernelports.ErrObjectNotFound
	}
	return &kernelports.DownloadResult{
		Body:     io.NopCloser(bytes.NewReader(body)),
		Size:     int64(len(body)),
		Metadata: s.metadata[key],
	}, nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

func (s *Store) DeleteObject(_ context.Context, params *kernelports.DeleteObjectParams) error {
	return s.Delete(context.Background(), params.Key)
}

func (s *Store) Exists(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objects[key]
	return ok, nil
}

func (s *Store) Put(key string, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = body
}

func (s *Store) Object(key string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, ok := s.objects[key]
	return body, ok
}

func (s *Store) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.objects))
	for key := range s.objects {
		keys = append(keys, key)
	}
	return keys
}

func (s *Store) Attempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempts
}
