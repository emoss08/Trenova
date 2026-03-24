package minio_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	minioadapter "github.com/emoss08/trenova/internal/infrastructure/minio"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestClient(t *testing.T, mc *testutil.MinioContainer) storage.Client {
	t.Helper()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:  mc.Endpoint(),
			AccessKey: mc.AccessKey(),
			SecretKey: mc.SecretKey(),
			Bucket:    mc.Bucket(),
			UseSSL:    false,
		},
	}

	logger := zap.NewNop()

	client, err := minioadapter.New(minioadapter.Params{
		Config: cfg,
		Logger: logger,
	})
	require.NoError(t, err)

	return client
}

func TestClient_Upload(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	tests := []struct {
		name        string
		key         string
		content     []byte
		contentType string
		metadata    map[string]string
		wantErr     bool
	}{
		{
			name:        "upload text file",
			key:         "test/file.txt",
			content:     []byte("Hello, World!"),
			contentType: "text/plain",
			metadata:    map[string]string{"custom": "value"},
			wantErr:     false,
		},
		{
			name:        "upload binary file",
			key:         "test/binary.bin",
			content:     []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			contentType: "application/octet-stream",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "upload pdf",
			key:         "documents/test.pdf",
			content:     []byte("%PDF-1.4 fake pdf content"),
			contentType: "application/pdf",
			metadata:    map[string]string{"original_name": "report.pdf"},
			wantErr:     false,
		},
		{
			name:        "upload with nested path",
			key:         "org123/trailers/uuid123/document.pdf",
			content:     []byte("nested path content"),
			contentType: "application/pdf",
			metadata:    nil,
			wantErr:     false,
		},
		{
			name:        "upload empty file",
			key:         "test/empty.txt",
			content:     []byte{},
			contentType: "text/plain",
			metadata:    nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Upload(tc.Ctx, &storage.UploadParams{
				Key:         tt.key,
				ContentType: tt.contentType,
				Size:        int64(len(tt.content)),
				Body:        bytes.NewReader(tt.content),
				Metadata:    tt.metadata,
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.key, result.Key)
			assert.Equal(t, int64(len(tt.content)), result.Size)
			assert.Equal(t, tt.contentType, result.ContentType)

			exists, err := client.Exists(tc.Ctx, tt.key)
			require.NoError(t, err)
			assert.True(t, exists)
		})
	}
}

func TestClient_Download(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	uploadContent := []byte("download test content")
	uploadKey := "download-test/file.txt"

	_, err := client.Upload(tc.Ctx, &storage.UploadParams{
		Key:         uploadKey,
		ContentType: "text/plain",
		Size:        int64(len(uploadContent)),
		Body:        bytes.NewReader(uploadContent),
		Metadata:    map[string]string{"test": "metadata"},
	})
	require.NoError(t, err)

	t.Run("download existing file", func(t *testing.T) {
		result, err := client.Download(tc.Ctx, uploadKey)
		require.NoError(t, err)
		defer result.Body.Close()

		content, err := io.ReadAll(result.Body)
		require.NoError(t, err)

		assert.Equal(t, uploadContent, content)
		assert.Equal(t, "text/plain", result.ContentType)
		assert.Equal(t, int64(len(uploadContent)), result.Size)
	})

	t.Run("download non-existent file", func(t *testing.T) {
		result, err := client.Download(tc.Ctx, "non-existent/file.txt")

		if err == nil && result != nil {
			_, readErr := io.ReadAll(result.Body)
			result.Body.Close()
			assert.Error(t, readErr)
		}
	})
}

func TestClient_Delete(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	uploadContent := []byte("delete test content")
	uploadKey := "delete-test/file.txt"

	_, err := client.Upload(tc.Ctx, &storage.UploadParams{
		Key:         uploadKey,
		ContentType: "text/plain",
		Size:        int64(len(uploadContent)),
		Body:        bytes.NewReader(uploadContent),
	})
	require.NoError(t, err)

	exists, err := client.Exists(tc.Ctx, uploadKey)
	require.NoError(t, err)
	assert.True(t, exists)

	t.Run("delete existing file", func(t *testing.T) {
		err := client.Delete(tc.Ctx, uploadKey)
		require.NoError(t, err)

		exists, err := client.Exists(tc.Ctx, uploadKey)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete non-existent file (should not error)", func(t *testing.T) {
		err := client.Delete(tc.Ctx, "non-existent/file.txt")
		assert.NoError(t, err)
	})
}

func TestClient_GetPresignedURL(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	uploadContent := []byte("presigned url test content")
	uploadKey := "presigned-test/file.txt"

	_, err := client.Upload(tc.Ctx, &storage.UploadParams{
		Key:         uploadKey,
		ContentType: "text/plain",
		Size:        int64(len(uploadContent)),
		Body:        bytes.NewReader(uploadContent),
	})
	require.NoError(t, err)

	t.Run("generate presigned URL", func(t *testing.T) {
		url, err := client.GetPresignedURL(tc.Ctx, &storage.PresignedURLParams{
			Key:    uploadKey,
			Expiry: 15 * time.Minute,
		})
		require.NoError(t, err)

		assert.NotEmpty(t, url)
		assert.Contains(t, url, uploadKey)
		assert.Contains(t, url, "X-Amz-Signature")
	})

	t.Run("generate presigned URL with content disposition", func(t *testing.T) {
		url, err := client.GetPresignedURL(tc.Ctx, &storage.PresignedURLParams{
			Key:                uploadKey,
			Expiry:             15 * time.Minute,
			ContentDisposition: "attachment; filename=\"download.txt\"",
		})
		require.NoError(t, err)

		assert.NotEmpty(t, url)
		assert.Contains(t, url, "response-content-disposition")
	})
}

func TestClient_Exists(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	existingKey := "exists-test/file.txt"
	_, err := client.Upload(tc.Ctx, &storage.UploadParams{
		Key:         existingKey,
		ContentType: "text/plain",
		Size:        5,
		Body:        strings.NewReader("hello"),
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "existing file",
			key:      existingKey,
			expected: true,
		},
		{
			name:     "non-existent file",
			key:      "non-existent/file.txt",
			expected: false,
		},
		{
			name:     "non-existent directory",
			key:      "completely/different/path/file.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := client.Exists(tc.Ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, exists)
		})
	}
}

func TestClient_GetFileInfo(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	uploadContent := []byte("file info test content")
	uploadKey := "fileinfo-test/file.txt"
	metadata := map[string]string{"custom-key": "custom-value"}

	_, err := client.Upload(tc.Ctx, &storage.UploadParams{
		Key:         uploadKey,
		ContentType: "text/plain",
		Size:        int64(len(uploadContent)),
		Body:        bytes.NewReader(uploadContent),
		Metadata:    metadata,
	})
	require.NoError(t, err)

	t.Run("get file info for existing file", func(t *testing.T) {
		info, err := client.GetFileInfo(tc.Ctx, uploadKey)
		require.NoError(t, err)

		assert.Equal(t, uploadKey, info.Key)
		assert.Equal(t, int64(len(uploadContent)), info.Size)
		assert.Equal(t, "text/plain", info.ContentType)
		assert.False(t, info.LastModified.IsZero())
	})

	t.Run("get file info for non-existent file", func(t *testing.T) {
		_, err := client.GetFileInfo(tc.Ctx, "non-existent/file.txt")
		assert.Error(t, err)
	})
}

func TestClient_LargeFileUpload(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	largeContent := make([]byte, 10*1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	uploadKey := "large-file-test/large.bin"

	t.Run("upload large file", func(t *testing.T) {
		result, err := client.Upload(tc.Ctx, &storage.UploadParams{
			Key:         uploadKey,
			ContentType: "application/octet-stream",
			Size:        int64(len(largeContent)),
			Body:        bytes.NewReader(largeContent),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(len(largeContent)), result.Size)
	})

	t.Run("download and verify large file", func(t *testing.T) {
		result, err := client.Download(tc.Ctx, uploadKey)
		require.NoError(t, err)
		defer result.Body.Close()

		downloadedContent, err := io.ReadAll(result.Body)
		require.NoError(t, err)

		assert.Equal(t, largeContent, downloadedContent)
	})
}

func TestClient_ConcurrentOperations(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	numOperations := 10
	results := make(chan error, numOperations)

	for i := range numOperations {
		go func(idx int) {
			key := "concurrent-test/file-" + string(rune('0'+idx)) + ".txt"
			content := []byte("content for file " + string(rune('0'+idx)))

			_, err := client.Upload(tc.Ctx, &storage.UploadParams{
				Key:         key,
				ContentType: "text/plain",
				Size:        int64(len(content)),
				Body:        bytes.NewReader(content),
			})
			if err != nil {
				results <- err
				return
			}

			downloadResult, err := client.Download(tc.Ctx, key)
			if err != nil {
				results <- err
				return
			}
			downloadResult.Body.Close()

			results <- nil
		}(i)
	}

	for range numOperations {
		err := <-results
		assert.NoError(t, err)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	ctx, cancel := context.WithCancel(tc.Ctx)
	cancel()

	_, err := client.Upload(ctx, &storage.UploadParams{
		Key:         "cancelled/file.txt",
		ContentType: "text/plain",
		Size:        5,
		Body:        strings.NewReader("hello"),
	})
	assert.Error(t, err)
}

type bucketEnsurer interface {
	EnsureBucket(ctx context.Context) error
}

func TestClient_EnsureBucket(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	ensurer, ok := client.(bucketEnsurer)
	require.True(t, ok)

	t.Run("bucket already exists", func(t *testing.T) {
		err := ensurer.EnsureBucket(tc.Ctx)
		require.NoError(t, err)
	})

	t.Run("idempotent call", func(t *testing.T) {
		err := ensurer.EnsureBucket(tc.Ctx)
		require.NoError(t, err)

		err = ensurer.EnsureBucket(tc.Ctx)
		require.NoError(t, err)
	})
}

func TestClient_New_WithRegion(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:  mc.Endpoint(),
			AccessKey: mc.AccessKey(),
			SecretKey: mc.SecretKey(),
			Bucket:    mc.Bucket(),
			UseSSL:    false,
			Region:    "us-east-1",
		},
	}

	logger := zap.NewNop()

	client, err := minioadapter.New(minioadapter.Params{
		Config: cfg,
		Logger: logger,
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestClient_Upload_WithMetadata(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	client := setupTestClient(t, mc)

	metadata := map[string]string{
		"org-id":        "org_123",
		"document-type": "invoice",
		"uploaded-by":   "user_456",
	}

	result, err := client.Upload(tc.Ctx, &storage.UploadParams{
		Key:         "metadata-test/file.txt",
		ContentType: "text/plain",
		Size:        int64(len("hello")),
		Body:        strings.NewReader("hello"),
		Metadata:    metadata,
	})
	require.NoError(t, err)
	assert.Equal(t, "metadata-test/file.txt", result.Key)
}

func TestClient_EnsureBucket_CreateNew(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:  mc.Endpoint(),
			AccessKey: mc.AccessKey(),
			SecretKey: mc.SecretKey(),
			Bucket:    "new-test-bucket-ensure",
			UseSSL:    false,
		},
	}

	logger := zap.NewNop()

	client, err := minioadapter.New(minioadapter.Params{
		Config: cfg,
		Logger: logger,
	})
	require.NoError(t, err)

	ensurer, ok := client.(bucketEnsurer)
	require.True(t, ok)

	err = ensurer.EnsureBucket(tc.Ctx)
	require.NoError(t, err)

	exists, err := client.Exists(tc.Ctx, "nonexistent-key")
	require.NoError(t, err)
	assert.False(t, exists)
}
