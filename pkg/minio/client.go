package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog"
)

// MinioClient defines the interface for Minio client operations.
type MinioClient interface {
	Ping(ctx context.Context) error
	CreateBucket(ctx context.Context, bucketName, location string) error
	UploadMedia(ctx context.Context, params UploadMediaOptions) (minio.UploadInfo, error)
	SaveFile(ctx context.Context, params SaveFileOptions) (string, error)
	UploadTemporaryFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error)
	MoveFile(ctx context.Context, tempBucket, tempObject, permBucket, permObject string) (string, error)
}

type Client struct {
	client   *minio.Client
	options  *minio.Options
	endpoint string
	logger   *zerolog.Logger
}

func NewClient(endpoint string, logger *zerolog.Logger, opts *minio.Options) *Client {
	mClient := &Client{
		endpoint: endpoint,
		options:  opts,
		logger:   logger,
	}

	mClient.initialize()
	return mClient
}

func (mc *Client) initialize() {
	client, err := minio.New(mc.endpoint, mc.options)
	if err != nil {
		mc.logger.Fatal().Err(err).Msg("MinioClient: Error initializing Minio client")
	}

	mc.client = client
}

func (mc *Client) Ping(ctx context.Context) error {
	_, err := mc.client.ListBuckets(ctx)
	return err
}

func (mc *Client) CreateBucket(ctx context.Context, bucketName, location string) error {
	exists, err := mc.client.BucketExists(ctx, bucketName)
	if err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error checking if bucket exists")
		return err
	}

	if !exists {
		if err = mc.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location}); err != nil {
			mc.logger.Error().Err(err).Msg("MinioClient: Error creating bucket")
			return err
		}
	}

	return nil
}

func (mc *Client) UploadMedia(ctx context.Context, params UploadMediaOptions) (minio.UploadInfo, error) {
	if err := mc.CreateBucket(ctx, params.BucketName, "us-east-1"); err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error creating bucket")
		return minio.UploadInfo{}, err
	}

	ui, err := mc.client.FPutObject(ctx, params.BucketName, params.ObjectName, params.FilePath, minio.PutObjectOptions{ContentType: params.ContentType})
	if err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error uploading media")
		return minio.UploadInfo{}, err
	}

	return ui, nil
}

func (mc *Client) SaveFile(ctx context.Context, params SaveFileOptions) (string, error) {
	if err := mc.CreateBucket(ctx, params.BucketName, "us-east-1"); err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error creating bucket")
		return "", err
	}

	_, err := mc.client.PutObject(
		ctx,
		params.BucketName,
		params.ObjectName,
		bytes.NewReader(params.FileData),
		int64(len(params.FileData)),
		minio.PutObjectOptions{ContentType: params.ContentType},
	)
	if err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error saving file")
		return "", err
	}

	fileURL := fmt.Sprintf("http://%s/%s/%s", mc.endpoint, params.BucketName, params.ObjectName)
	return fileURL, nil
}

func (mc *Client) UploadTemporaryFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	tempObjectName := fmt.Sprintf("temp-uploads/%s", header.Filename)
	tempBucketName := TemporaryBucket.String()

	fileData, err := io.ReadAll(file)
	if err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error reading file data")
		return "", err
	}

	fileURL, err := mc.SaveFile(ctx, SaveFileOptions{
		BucketName:  tempBucketName,
		ObjectName:  tempObjectName,
		FileData:    fileData,
		ContentType: header.Header.Get("Content-Type"),
	})
	if err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error saving file")
		return "", err
	}

	return fileURL, nil
}

func (mc *Client) MoveFile(ctx context.Context, tempBucket, tempObject, permBucket, permObject string) (string, error) {
	// Ensure the permanent bucket exists
	if err := mc.CreateBucket(ctx, permBucket, "us-east-1"); err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error creating permanent bucket")
		return "", err
	}

	// Copy the object from the temporary bucket to the permanent bucket
	if _, err := mc.client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: permBucket,
		Object: permObject,
	}, minio.CopySrcOptions{
		Bucket: tempBucket,
		Object: tempObject,
	}); err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error copying object to permanent bucket")
		return "", err
	}

	// Remove the object from the temporary bucket
	if err := mc.client.RemoveObject(ctx, tempBucket, tempObject, minio.RemoveObjectOptions{}); err != nil {
		mc.logger.Error().Err(err).Msg("MinioClient: Error removing object from temporary bucket")
		return "", err
	}

	permanentURL := fmt.Sprintf("http://%s/%s/%s", mc.endpoint, permBucket, permObject)
	return permanentURL, nil
}
