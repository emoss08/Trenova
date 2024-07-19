// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package testutils

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"testing"

	"github.com/emoss08/trenova/pkg/minio"
	mio "github.com/minio/minio-go/v7"
	"github.com/rs/zerolog"
)

type MockMinioClient struct {
	Buckets map[string]map[string][]byte
	Logger  *zerolog.Logger
}

func NewMockMinioClient(logger *zerolog.Logger) *MockMinioClient {
	return &MockMinioClient{
		Buckets: make(map[string]map[string][]byte),
		Logger:  logger,
	}
}

func (m *MockMinioClient) Ping(ctx context.Context) error {
	return nil
}

func (m *MockMinioClient) CreateBucket(ctx context.Context, bucketName, location string) error {
	if _, exists := m.Buckets[bucketName]; !exists {
		m.Buckets[bucketName] = make(map[string][]byte)
	}
	return nil
}

func (m *MockMinioClient) UploadMedia(ctx context.Context, params minio.UploadMediaOptions) (mio.UploadInfo, error) {
	if _, exists := m.Buckets[params.BucketName]; !exists {
		return mio.UploadInfo{}, errors.New("bucket does not exist")
	}

	m.Buckets[params.BucketName][params.ObjectName] = []byte("mock data")
	return mio.UploadInfo{
		Bucket: params.BucketName,
		Key:    params.ObjectName,
	}, nil
}

func (m *MockMinioClient) SaveFile(ctx context.Context, params minio.SaveFileOptions) (string, error) {
	if _, exists := m.Buckets[params.BucketName]; !exists {
		return "", errors.New("bucket does not exist")
	}

	m.Buckets[params.BucketName][params.ObjectName] = params.FileData
	fileURL := "http://mock-minio/" + params.BucketName + "/" + params.ObjectName
	return fileURL, nil
}

func (m *MockMinioClient) UploadTemporaryFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	if _, exists := m.Buckets[minio.TemporaryBucket.String()]; !exists {
		m.Buckets[minio.TemporaryBucket.String()] = make(map[string][]byte)
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	objectName := header.Filename
	m.Buckets[minio.TemporaryBucket.String()][objectName] = fileData
	fileURL := "http://mock-minio/" + minio.TemporaryBucket.String() + "/" + objectName
	return fileURL, nil
}

func (m *MockMinioClient) MoveFile(ctx context.Context, tempBucket, tempObject, permBucket, permObject string) (string, error) {
	if _, exists := m.Buckets[tempBucket]; !exists {
		m.Buckets[tempBucket] = make(map[string][]byte)
	}

	if _, exists := m.Buckets[permBucket]; !exists {
		m.Buckets[permBucket] = make(map[string][]byte)
	}

	fileData := m.Buckets[tempBucket][tempObject]
	m.Buckets[permBucket][permObject] = fileData
	delete(m.Buckets[tempBucket], tempObject)

	return permObject, nil
}

func WithTestMinioClient(t *testing.T, logger *zerolog.Logger) *MockMinioClient {
	t.Helper()
	return NewMockMinioClient(logger)
}
