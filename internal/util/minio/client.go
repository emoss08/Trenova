// Package minio provides a client for managing media storage using Minio.
package minio

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/pkg/errors"
)

// Client represents a client to interact with the Minio server.
type Client struct {
	client   *minio.Client  // client is the Minio client.
	options  *minio.Options // options are the configurations for the Minio client.
	endpoint string         // endpoint is the Minio server URL.
}

// NewClient initializes a new Minio client.
// It panics if the Minio client cannot be created.
func NewClient(endpoint string, opts *minio.Options) *Client {
	mClient := &Client{
		options:  opts,
		endpoint: endpoint,
	}
	mClient.initialize()
	return mClient
}

// initialize sets up the Minio client. It panics if the client cannot be created.
func (c *Client) initialize() {
	client, err := minio.New(c.endpoint, c.options)
	if err != nil {
		panic(errors.Wrap(err, "failed to create Minio client"))
	}
	c.client = client
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.ListBuckets(ctx)
	return err
}

// CreateBucket ensures a bucket is available for storing media.
//
// It creates the bucket if it does not already exist.
func (c *Client) CreateBucket(ctx context.Context, bucketName, location string) error {
	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return errors.Wrap(err, "checking if bucket exists failed")
	}
	if !exists {
		if err = c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location}); err != nil {
			return errors.Wrap(err, "creating bucket failed")
		}
	}
	return nil
}

// UploadMedia uploads a file to the specified bucket and object name.
//
// It returns UploadInfo or an error if the upload fails.
func (c *Client) UploadMedia(ctx context.Context, bucketName, filePath, objectName, contentType string) (minio.UploadInfo, error) {
	// Check if the bucket exists, if not then create it
	if err := c.CreateBucket(ctx, bucketName, "us-east-1"); err != nil {
		return minio.UploadInfo{}, errors.Wrap(err, "failed to create bucket")
	}

	ui, err := c.client.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return minio.UploadInfo{}, errors.Wrap(err, "upload failed")
	}
	return ui, nil
}

func (c *Client) SaveFile(ctx context.Context, bucketName, objectName, contentType string, fileData []byte) (string, error) {
	// Check if the bucket exists, if not then create it
	if err := c.CreateBucket(ctx, bucketName, "us-east-1"); err != nil {
		log.Printf("Failed to create bucketname %s to %s\n", objectName, bucketName)
		return "", errors.Wrap(err, "failed to create bucket")
	}

	_, err := c.client.PutObject(
		ctx,
		bucketName,
		objectName,
		bytes.NewReader(fileData),
		int64(len(fileData)),
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		log.Printf("Failed to upload %s to %s\n", objectName, bucketName)
		return "", errors.Wrap(err, "upload failed")
	}

	// Generate a public URL
	fileURL := fmt.Sprintf("http://%s/%s/%s", c.endpoint, bucketName, objectName)
	return fileURL, nil
}

func (c *Client) GetPresignedURL(ctx context.Context, bucketName, objectName string, expiry int64) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", "attachment; filename=\""+objectName+"\"")
	reqParams.Set("response-content-type", "application/octet-stream")

	presignedURL, err := c.client.PresignedGetObject(ctx, bucketName, objectName, time.Duration(expiry)*time.Second, reqParams)
	if err != nil {
		return "", errors.Wrap(err, "failed to get presigned URL")
	}
	return presignedURL.String(), nil
}
