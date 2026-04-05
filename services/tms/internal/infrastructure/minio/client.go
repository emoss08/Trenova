package minio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Client struct {
	client       *minio.Client
	core         *minio.Core
	publicClient *minio.Client
	publicCore   *minio.Core
	bucket       string
	l            *zap.Logger
}

func New(p Params) (storage.Client, error) {
	cfg := p.Config.GetStorageConfig()

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, cfg.SessionToken),
		Secure: cfg.UseSSL,
	}

	if cfg.Region != "" {
		opts.Region = cfg.Region
	}

	minioClient, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	minioCore, err := minio.NewCore(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio core client: %w", err)
	}

	var publicClient *minio.Client
	var publicCore *minio.Core
	if cfg.PublicEndpoint != "" {
		publicEndpoint, publicUseSSL := parsePublicEndpoint(cfg.PublicEndpoint)
		publicOpts := &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, cfg.SessionToken),
			Secure: publicUseSSL,
		}
		if cfg.Region != "" {
			publicOpts.Region = cfg.Region
		}
		publicClient, err = minio.New(publicEndpoint, publicOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to create public minio client: %w", err)
		}

		publicCore, err = minio.NewCore(publicEndpoint, publicOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to create public minio core client: %w", err)
		}
	}

	client := &Client{
		client:       minioClient,
		core:         minioCore,
		publicClient: publicClient,
		publicCore:   publicCore,
		bucket:       cfg.Bucket,
		l:            p.Logger.Named("infrastructure.minio"),
	}

	ensureCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err = client.EnsureBucket(ensureCtx); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	p.Logger.Info("minio bucket ready", zap.String("bucket", cfg.Bucket))

	return client, nil
}

func parsePublicEndpoint(publicEndpoint string) (endpoint string, useSSL bool) {
	if after, ok := strings.CutPrefix(publicEndpoint, "https://"); ok {
		return after, true
	}

	return strings.TrimPrefix(publicEndpoint, "http://"), false
}

func (c *Client) Upload(
	ctx context.Context,
	params *storage.UploadParams,
) (*storage.FileInfo, error) {
	log := c.l.With(
		zap.String("operation", "Upload"),
		zap.String("key", params.Key),
		zap.Int64("size", params.Size),
	)

	opts := minio.PutObjectOptions{
		ContentType: params.ContentType,
	}

	if len(params.Metadata) > 0 {
		opts.UserMetadata = params.Metadata
	}

	info, err := c.client.PutObject(ctx, c.bucket, params.Key, params.Body, params.Size, opts)
	if err != nil {
		log.Error("failed to upload file", zap.Error(err))
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &storage.FileInfo{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  params.ContentType,
		LastModified: time.Now(),
		Metadata:     params.Metadata,
		VersionID:    info.VersionID,
	}, nil
}

func (c *Client) Download(ctx context.Context, key string) (*storage.DownloadResult, error) {
	log := c.l.With(
		zap.String("operation", "Download"),
		zap.String("key", key),
	)

	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		log.Error("failed to get object", zap.Error(err))
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	stat, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		log.Error("failed to stat object", zap.Error(err))
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &storage.DownloadResult{
		Body:        obj,
		ContentType: stat.ContentType,
		Size:        stat.Size,
		Metadata:    stat.UserMetadata,
	}, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	return c.DeleteObject(ctx, &storage.DeleteObjectParams{Key: key})
}

func (c *Client) DeleteObject(
	ctx context.Context,
	params *storage.DeleteObjectParams,
) error {
	log := c.l.With(
		zap.String("operation", "Delete"),
		zap.String("key", params.Key),
		zap.String("versionId", params.VersionID),
	)

	err := c.client.RemoveObject(ctx, c.bucket, params.Key, minio.RemoveObjectOptions{
		VersionID: params.VersionID,
	})
	if err != nil {
		log.Error("failed to delete object", zap.Error(err))
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (c *Client) GetPresignedURL(
	ctx context.Context,
	params *storage.PresignedURLParams,
) (string, error) {
	log := c.l.With(
		zap.String("operation", "GetPresignedURL"),
		zap.String("key", params.Key),
	)

	reqParams := make(url.Values)
	if params.ContentDisposition != "" {
		reqParams.Set("response-content-disposition", params.ContentDisposition)
	}

	client := c.client
	if c.publicClient != nil {
		client = c.publicClient
	}

	presignedURL, err := client.PresignedGetObject(
		ctx,
		c.bucket,
		params.Key,
		params.Expiry,
		reqParams,
	)
	if err != nil {
		log.Error("failed to generate presigned URL", zap.Error(err))
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return presignedURL.String(), nil
}

func (c *Client) GetPresignedUploadURL(
	ctx context.Context,
	params *storage.PresignedUploadURLParams,
) (string, error) {
	client := c.client
	if c.publicClient != nil {
		client = c.publicClient
	}

	u, err := client.PresignedPutObject(ctx, c.bucket, params.Key, params.Expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return u.String(), nil
}

func (c *Client) InitiateMultipartUpload(
	ctx context.Context,
	params *storage.MultipartUploadParams,
) (string, error) {
	opts := minio.PutObjectOptions{
		ContentType: params.ContentType,
	}
	if len(params.Metadata) > 0 {
		opts.UserMetadata = params.Metadata
	}

	uploadID, err := c.core.NewMultipartUpload(ctx, c.bucket, params.Key, opts)
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	return uploadID, nil
}

func (c *Client) GetMultipartUploadPartURL(
	ctx context.Context,
	params *storage.MultipartUploadPartURLParams,
) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("partNumber", fmt.Sprintf("%d", params.PartNumber))
	reqParams.Set("uploadId", params.UploadID)

	client := c.client
	if c.publicClient != nil {
		client = c.publicClient
	}

	u, err := client.Presign(ctx, http.MethodPut, c.bucket, params.Key, params.Expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate multipart part upload URL: %w", err)
	}

	return u.String(), nil
}

func (c *Client) CompleteMultipartUpload(
	ctx context.Context,
	params *storage.CompleteMultipartUploadParams,
) error {
	parts := make([]minio.CompletePart, 0, len(params.Parts))
	for _, part := range params.Parts {
		parts = append(parts, minio.CompletePart{
			PartNumber: part.PartNumber,
			ETag:       strings.Trim(part.ETag, "\""),
		})
	}

	slices.SortFunc(parts, func(a, b minio.CompletePart) int {
		return a.PartNumber - b.PartNumber
	})

	if _, err := c.core.CompleteMultipartUpload(
		ctx,
		c.bucket,
		params.Key,
		params.UploadID,
		parts,
		minio.PutObjectOptions{},
	); err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

func (c *Client) AbortMultipartUpload(
	ctx context.Context,
	params *storage.AbortMultipartUploadParams,
) error {
	if err := c.core.AbortMultipartUpload(ctx, c.bucket, params.Key, params.UploadID); err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	return nil
}

func (c *Client) ListMultipartUploadParts(
	ctx context.Context,
	params *storage.ListMultipartUploadPartsParams,
) ([]storage.UploadedPart, error) {
	result, err := c.core.ListObjectParts(ctx, c.bucket, params.Key, params.UploadID, 0, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to list multipart upload parts: %w", err)
	}

	parts := make([]storage.UploadedPart, 0, len(result.ObjectParts))
	for _, part := range result.ObjectParts {
		parts = append(parts, storage.UploadedPart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
			Size:       part.Size,
		})
	}

	return parts, nil
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

func (c *Client) GetFileInfo(ctx context.Context, key string) (*storage.FileInfo, error) {
	log := c.l.With(
		zap.String("operation", "GetFileInfo"),
		zap.String("key", key),
	)

	stat, err := c.client.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		log.Error("failed to stat object", zap.Error(err))
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	info := &storage.FileInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		Metadata:     stat.UserMetadata,
		VersionID:    stat.VersionID,
	}

	return info, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = c.client.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}
