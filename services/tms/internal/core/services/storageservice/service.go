package storageservice

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger      *zap.Logger
	MinIOClient *minio.Client
}

type Service struct {
	logger *zap.Logger
	client *minio.Client
}

func NewService(p ServiceParams) *Service {
	return &Service{
		logger: p.Logger.Named("service.storage"),
		client: p.MinIOClient,
	}
}

// EnsureBucket creates a bucket if it doesn't exist
func (s *Service) EnsureBucket(ctx context.Context, bucketName string) error {
	exists, err := s.client.BucketExists(ctx, bucketName)
	if err != nil {
		s.logger.Error("failed to check bucket existence",
			zap.Error(err),
			zap.String("bucket", bucketName),
		)
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			s.logger.Error("failed to create bucket",
				zap.Error(err),
				zap.String("bucket", bucketName),
			)
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		s.logger.Info("created bucket", zap.String("bucket", bucketName))
	}

	return nil
}

// UploadFile uploads a file to MinIO storage
func (s *Service) UploadFile(
	ctx context.Context,
	bucketName string,
	objectName string,
	data []byte,
	contentType string,
) error {
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(
		ctx,
		bucketName,
		objectName,
		reader,
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		s.logger.Error("failed to upload file",
			zap.Error(err),
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
		return fmt.Errorf("failed to upload file: %w", err)
	}

	s.logger.Info("uploaded file",
		zap.String("bucket", bucketName),
		zap.String("object", objectName),
		zap.Int("size", len(data)),
	)

	return nil
}

// DownloadFile downloads a file from MinIO storage
func (s *Service) DownloadFile(
	ctx context.Context,
	bucketName string,
	objectName string,
) ([]byte, error) {
	object, err := s.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		s.logger.Error("failed to get object",
			zap.Error(err),
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		s.logger.Error("failed to read object",
			zap.Error(err),
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// GetFileInfo retrieves file information from MinIO storage
func (s *Service) GetFileInfo(
	ctx context.Context,
	bucketName string,
	objectName string,
) (*FileInfo, error) {
	object, err := s.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		s.logger.Error("failed to get object for stat",
			zap.Error(err),
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	stat, err := object.Stat()
	if err != nil {
		s.logger.Error("failed to stat object",
			zap.Error(err),
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	return &FileInfo{
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}, nil
}

// StreamFile returns a reader for streaming a file from MinIO
func (s *Service) StreamFile(
	ctx context.Context,
	bucketName string,
	objectName string,
) (*minio.Object, error) {
	object, err := s.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		s.logger.Error("failed to get object for streaming",
			zap.Error(err),
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return object, nil
}
