package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/storage"
)

type MockStorageClient struct {
	mu      sync.RWMutex
	files   map[string]*MockFile
	uploads []UploadCall
	deletes []string

	UploadFunc          func(ctx context.Context, params *storage.UploadParams) (*storage.FileInfo, error)
	DownloadFunc        func(ctx context.Context, key string) (*storage.DownloadResult, error)
	DeleteFunc          func(ctx context.Context, key string) error
	GetPresignedURLFunc func(ctx context.Context, params *storage.PresignedURLParams) (string, error)
	ExistsFunc          func(ctx context.Context, key string) (bool, error)
	GetFileInfoFunc     func(ctx context.Context, key string) (*storage.FileInfo, error)
}

type MockFile struct {
	Key         string
	Data        []byte
	ContentType string
	Size        int64
	Metadata    map[string]string
	CreatedAt   time.Time
}

type UploadCall struct {
	Key         string
	ContentType string
	Size        int64
	Metadata    map[string]string
}

func NewMockStorageClient() *MockStorageClient {
	return &MockStorageClient{
		files:   make(map[string]*MockFile),
		uploads: make([]UploadCall, 0),
		deletes: make([]string, 0),
	}
}

func (m *MockStorageClient) Upload(
	ctx context.Context,
	params *storage.UploadParams,
) (*storage.FileInfo, error) {
	if m.UploadFunc != nil {
		return m.UploadFunc(ctx, params)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	m.files[params.Key] = &MockFile{
		Key:         params.Key,
		Data:        data,
		ContentType: params.ContentType,
		Size:        params.Size,
		Metadata:    params.Metadata,
		CreatedAt:   time.Now(),
	}

	m.uploads = append(m.uploads, UploadCall{
		Key:         params.Key,
		ContentType: params.ContentType,
		Size:        params.Size,
		Metadata:    params.Metadata,
	})

	return &storage.FileInfo{
		Key:          params.Key,
		Size:         params.Size,
		ContentType:  params.ContentType,
		LastModified: time.Now(),
		Metadata:     params.Metadata,
	}, nil
}

func (m *MockStorageClient) Download(
	ctx context.Context,
	key string,
) (*storage.DownloadResult, error) {
	if m.DownloadFunc != nil {
		return m.DownloadFunc(ctx, key)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	file, exists := m.files[key]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", key)
	}

	return &storage.DownloadResult{
		Body:        io.NopCloser(bytes.NewReader(file.Data)),
		ContentType: file.ContentType,
		Size:        file.Size,
		Metadata:    file.Metadata,
	}, nil
}

func (m *MockStorageClient) Delete(ctx context.Context, key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.files, key)
	m.deletes = append(m.deletes, key)

	return nil
}

func (m *MockStorageClient) GetPresignedURL(
	ctx context.Context,
	params *storage.PresignedURLParams,
) (string, error) {
	if m.GetPresignedURLFunc != nil {
		return m.GetPresignedURLFunc(ctx, params)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.files[params.Key]; !exists {
		return "", fmt.Errorf("file not found: %s", params.Key)
	}

	return fmt.Sprintf(
		"https://mock-storage.example.com/%s?expires=%d",
		params.Key,
		time.Now().Add(params.Expiry).Unix(),
	), nil
}

func (m *MockStorageClient) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, key)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.files[key]
	return exists, nil
}

func (m *MockStorageClient) GetFileInfo(
	ctx context.Context,
	key string,
) (*storage.FileInfo, error) {
	if m.GetFileInfoFunc != nil {
		return m.GetFileInfoFunc(ctx, key)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	file, exists := m.files[key]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", key)
	}

	return &storage.FileInfo{
		Key:          file.Key,
		Size:         file.Size,
		ContentType:  file.ContentType,
		LastModified: file.CreatedAt,
		Metadata:     file.Metadata,
	}, nil
}

func (m *MockStorageClient) GetUploads() []UploadCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.uploads
}

func (m *MockStorageClient) GetDeletes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.deletes
}

func (m *MockStorageClient) GetFile(key string) (*MockFile, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	file, exists := m.files[key]
	return file, exists
}

func (m *MockStorageClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = make(map[string]*MockFile)
	m.uploads = make([]UploadCall, 0)
	m.deletes = make([]string, 0)
}

func (m *MockStorageClient) AddFile(
	key string,
	data []byte,
	contentType string,
	metadata map[string]string,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[key] = &MockFile{
		Key:         key,
		Data:        data,
		ContentType: contentType,
		Size:        int64(len(data)),
		Metadata:    metadata,
		CreatedAt:   time.Now(),
	}
}
