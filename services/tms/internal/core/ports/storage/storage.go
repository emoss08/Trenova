package storage

import (
	"context"
	"io"
	"time"
)

type UploadParams struct {
	Key         string
	ContentType string
	Size        int64
	Body        io.Reader
	Metadata    map[string]string
}

type DownloadResult struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
	Metadata    map[string]string
}

type PresignedURLParams struct {
	Key                string
	Expiry             time.Duration
	ContentDisposition string
}

type FileInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	Metadata     map[string]string
}

type Client interface {
	Upload(ctx context.Context, params *UploadParams) (*FileInfo, error)
	Download(ctx context.Context, key string) (*DownloadResult, error)
	Delete(ctx context.Context, key string) error
	GetPresignedURL(ctx context.Context, params *PresignedURLParams) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	GetFileInfo(ctx context.Context, key string) (*FileInfo, error)
}
