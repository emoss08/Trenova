package minio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	publicClient *minio.Client
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

	var publicClient *minio.Client
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
	}

	client := &Client{
		client:       minioClient,
		publicClient: publicClient,
		bucket:       cfg.Bucket,
		l:            p.Logger.Named("infrastructure.minio"),
	}

	if err = client.EnsureBucket(context.Background()); err != nil {
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
	log := c.l.With(
		zap.String("operation", "Delete"),
		zap.String("key", key),
	)

	err := c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
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

	return &storage.FileInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		Metadata:     stat.UserMetadata,
	}, nil
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
