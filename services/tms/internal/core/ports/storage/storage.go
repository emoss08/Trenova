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

type PresignedUploadURLParams struct {
	Key         string
	Expiry      time.Duration
	ContentType string
}

type MultipartUploadParams struct {
	Key         string
	ContentType string
	Metadata    map[string]string
}

type MultipartUploadPartURLParams struct {
	Key        string
	UploadID   string
	PartNumber int
	Expiry     time.Duration
}

type UploadedPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
}

type CompleteMultipartUploadParams struct {
	Key      string
	UploadID string
	Parts    []UploadedPart
}

type AbortMultipartUploadParams struct {
	Key      string
	UploadID string
}

type DeleteObjectParams struct {
	Key       string
	VersionID string
}

type ListMultipartUploadPartsParams struct {
	Key      string
	UploadID string
}

type FileInfo struct {
	Key            string
	Size           int64
	ContentType    string
	LastModified   time.Time
	Metadata       map[string]string
	VersionID      string
	RetentionMode  string
	RetentionUntil *time.Time
	LegalHold      bool
}

type Client interface {
	Upload(ctx context.Context, params *UploadParams) (*FileInfo, error)
	Download(ctx context.Context, key string) (*DownloadResult, error)
	Delete(ctx context.Context, key string) error
	DeleteObject(ctx context.Context, params *DeleteObjectParams) error
	GetPresignedURL(ctx context.Context, params *PresignedURLParams) (string, error)
	GetPresignedUploadURL(ctx context.Context, params *PresignedUploadURLParams) (string, error)
	InitiateMultipartUpload(ctx context.Context, params *MultipartUploadParams) (string, error)
	GetMultipartUploadPartURL(ctx context.Context, params *MultipartUploadPartURLParams) (string, error)
	CompleteMultipartUpload(ctx context.Context, params *CompleteMultipartUploadParams) error
	AbortMultipartUpload(ctx context.Context, params *AbortMultipartUploadParams) error
	ListMultipartUploadParts(
		ctx context.Context,
		params *ListMultipartUploadPartsParams,
	) ([]UploadedPart, error)
	Exists(ctx context.Context, key string) (bool, error)
	GetFileInfo(ctx context.Context, key string) (*FileInfo, error)
}
