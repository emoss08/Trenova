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
