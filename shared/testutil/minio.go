package testutil

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	sharedMinioContainer *MinioContainer
	sharedMinioOnce      sync.Once
	sharedMinioErr       error
)

type MinioContainer struct {
	container testcontainers.Container
	endpoint  string
	accessKey string
	secretKey string
	bucket    string
	client    *minio.Client
}

type minioContainerWrapper struct {
	container testcontainers.Container
}

func (w *minioContainerWrapper) Terminate(ctx context.Context) error {
	return w.container.Terminate(ctx)
}

type MinioOptions struct {
	Image     string
	AccessKey string
	SecretKey string
	Bucket    string
}

func DefaultMinioOptions() MinioOptions {
	return MinioOptions{
		Image:     "minio/minio:latest",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Bucket:    "test-bucket",
	}
}

func SetupMinio(
	t *testing.T,
	tc *TestContext,
	opts ...func(*MinioOptions),
) *MinioContainer {
	t.Helper()

	options := DefaultMinioOptions()
	for _, opt := range opts {
		opt(&options)
	}

	req := testcontainers.ContainerRequest{
		Image:        options.Image,
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     options.AccessKey,
			"MINIO_ROOT_PASSWORD": options.SecretKey,
		},
		Cmd: []string{"server", "/data"},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
			wait.ForListeningPort("9000/tcp"),
		).WithDeadline(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(
		tc.Ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	require.NoError(t, err, "failed to start minio container")

	tc.AddContainer(&minioContainerWrapper{container: container})

	host, err := container.Host(tc.Ctx)
	require.NoError(t, err, "failed to get minio host")

	port, err := container.MappedPort(tc.Ctx, "9000/tcp")
	require.NoError(t, err, "failed to get minio port")

	endpoint := fmt.Sprintf("%s:%s", host, port.Port())

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(options.AccessKey, options.SecretKey, ""),
		Secure: false,
	})
	require.NoError(t, err, "failed to create minio client")

	err = client.MakeBucket(tc.Ctx, options.Bucket, minio.MakeBucketOptions{})
	require.NoError(t, err, "failed to create test bucket")

	return &MinioContainer{
		container: container,
		endpoint:  endpoint,
		accessKey: options.AccessKey,
		secretKey: options.SecretKey,
		bucket:    options.Bucket,
		client:    client,
	}
}

func (m *MinioContainer) Client() *minio.Client {
	return m.client
}

func (m *MinioContainer) Endpoint() string {
	return m.endpoint
}

func (m *MinioContainer) AccessKey() string {
	return m.accessKey
}

func (m *MinioContainer) SecretKey() string {
	return m.secretKey
}

func (m *MinioContainer) Bucket() string {
	return m.bucket
}

func (m *MinioContainer) Terminate(ctx context.Context) error {
	if m.container != nil {
		return m.container.Terminate(ctx)
	}
	return nil
}

func (m *MinioContainer) ClearBucket(ctx context.Context) error {
	objectsCh := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{Recursive: true})
	for obj := range objectsCh {
		if obj.Err != nil {
			return obj.Err
		}
		err := m.client.RemoveObject(ctx, m.bucket, obj.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func WithMinioImage(image string) func(*MinioOptions) {
	return func(o *MinioOptions) {
		o.Image = image
	}
}

func WithMinioCredentials(accessKey, secretKey string) func(*MinioOptions) {
	return func(o *MinioOptions) {
		o.AccessKey = accessKey
		o.SecretKey = secretKey
	}
}

func WithMinioBucket(bucket string) func(*MinioOptions) {
	return func(o *MinioOptions) {
		o.Bucket = bucket
	}
}

func getSharedMinio() (*MinioContainer, error) {
	sharedMinioOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		options := DefaultMinioOptions()

		req := testcontainers.ContainerRequest{
			Name:         "trenova-test-minio",
			Image:        options.Image,
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     options.AccessKey,
				"MINIO_ROOT_PASSWORD": options.SecretKey,
			},
			Cmd: []string{"server", "/data"},
			WaitingFor: wait.ForAll(
				wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
				wait.ForListeningPort("9000/tcp"),
			).WithDeadline(60 * time.Second),
		}

		container, err := testcontainers.GenericContainer(
			ctx,
			testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
				Reuse:            true,
			},
		)
		if err != nil {
			sharedMinioErr = fmt.Errorf("failed to start minio container: %w", err)
			return
		}

		host, err := container.Host(ctx)
		if err != nil {
			sharedMinioErr = fmt.Errorf("failed to get minio host: %w", err)
			return
		}

		port, err := container.MappedPort(ctx, "9000/tcp")
		if err != nil {
			sharedMinioErr = fmt.Errorf("failed to get minio port: %w", err)
			return
		}

		endpoint := fmt.Sprintf("%s:%s", host, port.Port())

		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(options.AccessKey, options.SecretKey, ""),
			Secure: false,
		})
		if err != nil {
			sharedMinioErr = fmt.Errorf("failed to create minio client: %w", err)
			return
		}

		err = client.MakeBucket(ctx, options.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			errResponse := minio.ToErrorResponse(err)
			if errResponse.Code != "BucketAlreadyOwnedByYou" {
				sharedMinioErr = fmt.Errorf("failed to create test bucket: %w", err)
				return
			}
		}

		sharedMinioContainer = &MinioContainer{
			container: container,
			endpoint:  endpoint,
			accessKey: options.AccessKey,
			secretKey: options.SecretKey,
			bucket:    options.Bucket,
			client:    client,
		}
	})

	return sharedMinioContainer, sharedMinioErr
}

func SetupTestMinio(t *testing.T) (*TestContext, *MinioContainer) {
	t.Helper()
	RequireIntegration(t)

	mc, err := getSharedMinio()
	require.NoError(t, err, "failed to get shared minio container")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := mc.ClearBucket(ctx); err != nil {
		t.Logf("Warning: failed to clear bucket: %v", err)
	}

	tc := &TestContext{
		T:          t,
		Ctx:        ctx,
		Cancel:     cancel,
		Containers: make([]Container, 0),
	}
	t.Cleanup(func() {
		cancel()
	})

	return tc, mc
}
