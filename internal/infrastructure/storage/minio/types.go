package minio

import (
	"github.com/minio/minio-go/v7"
)

// MakeBucketOptions is a wrapper around minio.MakeBucketOptions
type MakeBucketOptions = minio.MakeBucketOptions

// PutObjectOptions is a wrapper around minio.PutObjectOptions
type PutObjectOptions = minio.PutObjectOptions

// RemoveObjectOptions is a wrapper around minio.RemoveObjectOptions
type RemoveObjectOptions = minio.RemoveObjectOptions

// ListObjectsOptions is a wrapper around minio.ListObjectsOptions
type ListObjectsOptions = minio.ListObjectsOptions

// ObjectInfo is a wrapper around minio.ObjectInfo
type ObjectInfo = minio.ObjectInfo

// Object is a wrapper around minio.Object
type Object = minio.Object

// StatObjectOptions is a wrapper around minio.StatObjectOptions
type StatObjectOptions = minio.StatObjectOptions

// GetObjectOptions is a wrapper around minio.GetObjectOptions
type GetObjectOptions = minio.GetObjectOptions

// ChecksumType is a wrapper around minio.ChecksumType
type ChecksumType = minio.ChecksumType

const (
	ChecksumSHA256 ChecksumType = minio.ChecksumSHA256
	ChecksumSHA1   ChecksumType = minio.ChecksumSHA1
	ChecksumCRC32  ChecksumType = minio.ChecksumCRC32
	ChecksumCRC32C ChecksumType = minio.ChecksumCRC32C
)
