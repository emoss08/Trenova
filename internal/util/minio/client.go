package minio

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	Client *minio.Client
	Config *minio.Options
}

func NewMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: minioClient,
		Config: &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
		},
	}, nil
}

// // CreateMediaBucket creates a new bucket for media.
// func (m *Client) CreateMediaBucket(bucketName, location string) error {
// 	ctx := context.Background()

// 	err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
// 	if err != nil {
// 		// Check to see if we already own this bucket (which happens if the server restarts)
// 		exists, errBucketExists := client.BucketExists(ctx, bucketName)
// 		if errBucketExists == nil && exists {
// 			log.Printf("We already own %s\n", bucketName)
// 		} else {
// 			log.Fatalln(err)
// 		}
// 	} else {
// 		log.Printf("Successfully created %s\n", bucketName)
// 	}

// 	return nil
// }

// // UploadFile uploads a file to the minio server
// // By default, the file will be uploaded to the media bucket.
// func (m *Client) UploadFile(objectName, filePath, bucketName string) error {
// 	contenttype := "application/octet-stream"
// 	ctx := context.Background()

// 	_, err := client.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contenttype})
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// // UploadImage uploads an image to the minio server
// // By default, the image will be uploaded to the media bucket.
// func (m *Client) UploadImage(objectName, filePath, bucketName string) error {
// 	contenttype := "image/webp"
// 	ctx := context.Background()

// 	_, err := client.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contenttype})
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
