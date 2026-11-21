package objectstoretest

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type Store struct {
	kernelports.Client

	mu       sync.Mutex
	objects  map[string][]byte
	metadata map[string]map[string]string
	types    map[string]string
	attempts int
	versions int
}

func New() *Store {
	return &Store{
		objects:  map[string][]byte{},
		metadata: map[string]map[string]string{},
		types:    map[string]string{},
	}
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
	s.types[params.Key] = params.ContentType
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
		Body:        io.NopCloser(bytes.NewReader(body)),
		ContentType: s.types[key],
		Size:        int64(len(body)),
		Metadata:    s.metadata[key],
	}, nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	delete(s.metadata, key)
	delete(s.types, key)
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
	s.PutTyped(key, "", body)
}

func (s *Store) PutTyped(key, contentType string, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = body
	s.types[key] = contentType
}

func (s *Store) ContentType(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.types[key]
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

func (s *Store) ListPrefix(
	_ context.Context,
	prefix string,
	fn func(kernelports.ListedObject) error,
) error {
	s.mu.Lock()
	listed := make([]kernelports.ListedObject, 0)
	for key, body := range s.objects {
		if strings.HasPrefix(key, prefix) {
			listed = append(listed, kernelports.ListedObject{Key: key, Size: int64(len(body))})
		}
	}
	s.mu.Unlock()
	slices.SortFunc(listed, func(a, b kernelports.ListedObject) int {
		return strings.Compare(a.Key, b.Key)
	})
	for _, object := range listed {
		if err := fn(object); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeletePrefix(_ context.Context, prefix string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for key := range s.objects {
		if strings.HasPrefix(key, prefix) {
			delete(s.objects, key)
			delete(s.metadata, key)
			delete(s.types, key)
			n++
		}
	}
	return n, nil
}

func (s *Store) GetFileInfo(_ context.Context, key string) (*kernelports.FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, ok := s.objects[key]
	if !ok {
		return nil, kernelports.ErrObjectNotFound
	}
	return &kernelports.FileInfo{
		Key:         key,
		Size:        int64(len(body)),
		ContentType: s.types[key],
		Metadata:    s.metadata[key],
	}, nil
}

func (s *Store) Attempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempts
}
