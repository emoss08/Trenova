package storageservice

import "time"

// FileInfo represents metadata about a file in storage
type FileInfo struct {
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}
