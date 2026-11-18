package storage_test

import (
	"context"
	"crypto/md5" //nolint:gosec // the test checks the Content-MD5 S3 requires
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/storage"
	"github.com/emoss08/trenova/operator/internal/values"
)

const namespace = "trenova"

type stubCP struct {
	mu    sync.Mutex
	calls int
	value instanceapi.InstanceStorage
	err   error
}

func (s *stubCP) Storage(context.Context) (instanceapi.InstanceStorage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.value, s.err
}

type stubCORS struct {
	mu      sync.Mutex
	buckets []storage.Bucket
	origins []string
	err     error
}

func (s *stubCORS) Allow(_ context.Context, b storage.Bucket, origins []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets = append(s.buckets, b)
	s.origins = append(s.origins, strings.Join(origins, ","))
	return s.err
}

func fetched() instanceapi.InstanceStorage {
	return instanceapi.InstanceStorage{
		Endpoint:      "https://s3.bhs.io.cloud.ovh.net",
		Region:        "bhs",
		BackupsPrefix: "pitr/",
		Documents: instanceapi.StorageBucket{
			Bucket: "trenova-abc-docs", AccessKeyId: "DOCSKEY0123456789", SecretAccessKey: "documents-secret-0123456789",
		},
		Backups: instanceapi.StorageBucket{
			Bucket: "trenova-abc-backups", AccessKeyId: "BACKUPKEY01234567", SecretAccessKey: "backups-secret-0123456789",
		},
	}
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestTheBucketsAreFetchedOnceIntoTwoSecrets(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	cp := &stubCP{value: fetched()}
	cors := &stubCORS{}
	s := storage.New(&storage.Params{Client: client, ControlPlane: cp, CORS: cors, Namespace: namespace, Logger: quiet()})

	loc, err := s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
	require.NoError(t, err)
	require.Equal(t, values.Storage{
		Endpoint: "https://s3.bhs.io.cloud.ovh.net", Region: "bhs", DocumentsBucket: "trenova-abc-docs",
		BackupsPath: "s3://trenova-abc-backups/pitr/",
	}, loc)
	require.True(t, loc.Configured())

	docs, err := client.CoreV1().Secrets(namespace).Get(ctx, values.DocumentsSecret, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "DOCSKEY0123456789", string(docs.Data[storage.KeyAccessKey]))
	require.Equal(t, "documents-secret-0123456789", string(docs.Data[storage.KeySecretKey]))
	// The application's Secret holds the documents credential alone.
	for _, v := range docs.Data {
		require.NotContains(t, string(v), "backups-secret")
	}
	backups, err := client.CoreV1().Secrets(namespace).Get(ctx, values.BackupsSecret, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "BACKUPKEY01234567", string(backups.Data[storage.KeyBackupAccessKeyID]))
	require.Equal(t, "backups-secret-0123456789", string(backups.Data[storage.KeyBackupSecretKey]))
	require.Equal(t, "bhs", string(backups.Data[storage.KeyBackupRegion]))
	require.Equal(t, "s3://trenova-abc-backups/pitr/", string(backups.Data[storage.KeyBackupPath]))

	// The browser may read the documents bucket from the instance's origin,
	// set once with the documents credential.
	require.Equal(t, []string{"https://abc.trenova.app"}, cors.origins)
	require.Equal(t, "trenova-abc-docs", cors.buckets[0].Name)
	require.Equal(t, "DOCSKEY0123456789", cors.buckets[0].AccessKeyID)

	// Held: neither the control plane nor the bucket is asked again.
	again, err := s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
	require.NoError(t, err)
	require.Equal(t, loc, again)
	require.Equal(t, 1, cp.calls)
	require.Len(t, cors.origins, 1)

	// A Secret that lost a key is restored from the control plane.
	delete(backups.Data, storage.KeyBackupSecretKey)
	_, err = client.CoreV1().Secrets(namespace).Update(ctx, backups, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
	require.NoError(t, err)
	require.Equal(t, 2, cp.calls)
	backups, err = client.CoreV1().Secrets(namespace).Get(ctx, values.BackupsSecret, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "backups-secret-0123456789", string(backups.Data[storage.KeyBackupSecretKey]))
}

func TestAnInstanceWithoutStorageKeepsTheChartsStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	cp := &stubCP{err: controlplane.ErrNoStorage}
	s := storage.New(&storage.Params{
		Client: fake.NewSimpleClientset(), ControlPlane: cp, Namespace: namespace, Logger: quiet(),
		Now: func() time.Time { return now },
	})
	loc, err := s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
	require.NoError(t, err)
	require.False(t, loc.Configured())
	_, err = s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
	require.NoError(t, err)
	require.Equal(t, 1, cp.calls)
	now = now.Add(11 * time.Minute)
	_, err = s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
	require.NoError(t, err)
	require.Equal(t, 2, cp.calls)
}

func TestANewRevisionLooksForStorageAgainAtOnce(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	client := fake.NewSimpleClientset()
	cp := &stubCP{err: controlplane.ErrNoStorage}
	s := storage.New(&storage.Params{
		Client: client, ControlPlane: cp, Namespace: namespace, Logger: quiet(),
		Now: func() time.Time { return now },
	})
	loc, err := s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
	require.NoError(t, err)
	require.False(t, loc.Configured())

	cp.mu.Lock()
	cp.err, cp.value = nil, fetched()
	cp.mu.Unlock()
	now = now.Add(2 * time.Minute)
	loc, err = s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
	require.NoError(t, err)
	require.False(t, loc.Configured(), "the same revision waits out the recheck")
	require.Equal(t, 1, cp.calls)

	loc, err = s.Ensure(ctx, []string{"abc.trenova.app"}, 2)
	require.NoError(t, err)
	require.True(t, loc.Configured(), "the next revision asks again")
	require.Equal(t, 2, cp.calls)
	_, err = client.CoreV1().Secrets(namespace).Get(ctx, values.BackupsSecret, metav1.GetOptions{})
	require.NoError(t, err)

	_, err = s.Ensure(ctx, []string{"abc.trenova.app"}, 2)
	require.NoError(t, err)
	require.Equal(t, 2, cp.calls, "the Secrets are held from then on")
}

func TestMalformedStorageIsNeverWritten(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for name, change := range map[string]func(*instanceapi.InstanceStorage){
		"an http endpoint":      func(v *instanceapi.InstanceStorage) { v.Endpoint = "http://s3.example.com" },
		"a shared bucket":       func(v *instanceapi.InstanceStorage) { v.Backups.Bucket = v.Documents.Bucket },
		"a shared credential":   func(v *instanceapi.InstanceStorage) { v.Backups.AccessKeyId = v.Documents.AccessKeyId },
		"no secret":             func(v *instanceapi.InstanceStorage) { v.Documents.SecretAccessKey = "" },
		"a newline in a secret": func(v *instanceapi.InstanceStorage) { v.Backups.SecretAccessKey = "abcdefghijklmnop\nq" },
		"a bucket path":         func(v *instanceapi.InstanceStorage) { v.Documents.Bucket = "a/b" },
	} {
		v := fetched()
		change(&v)
		client := fake.NewSimpleClientset()
		s := storage.New(&storage.Params{Client: client, ControlPlane: &stubCP{value: v}, Namespace: namespace, Logger: quiet()})
		_, err := s.Ensure(ctx, []string{"abc.trenova.app"}, 1)
		require.ErrorIs(t, err, storage.ErrInvalid, name)
		list, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		require.Empty(t, list.Items, name)
	}
}

func TestCORSRulesAreSignedForTheBucket(t *testing.T) {
	t.Parallel()
	var got *http.Request
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(context.Background())
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	at := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	c := storage.S3CORS{Client: srv.Client(), Now: func() time.Time { return at }}
	err := c.Allow(context.Background(), storage.Bucket{
		Endpoint: srv.URL, Region: "bhs", Name: "trenova-abc-docs",
		AccessKeyID: "DOCSKEY0123456789", SecretAccessKey: "documents-secret-0123456789",
	}, []string{"https://abc.trenova.app"})
	require.NoError(t, err)
	require.Equal(t, http.MethodPut, got.Method)
	require.Equal(t, "/trenova-abc-docs", got.URL.Path)
	require.True(t, got.URL.Query().Has("cors"))
	require.Len(t, got.URL.Query(), 1)
	auth := got.Header.Get("Authorization")
	require.True(t, strings.HasPrefix(auth,
		"AWS4-HMAC-SHA256 Credential=DOCSKEY0123456789/20261114/bhs/s3/aws4_request"), auth)
	require.NotContains(t, auth, "documents-secret")
	sum := md5.Sum(body) //nolint:gosec // see the import
	require.Equal(t, base64.StdEncoding.EncodeToString(sum[:]), got.Header.Get("Content-MD5"))
	require.Contains(t, string(body), "<AllowedOrigin>https://abc.trenova.app</AllowedOrigin>")

	refused := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer refused.Close()
	c = storage.S3CORS{Client: refused.Client()}
	require.Error(t, c.Allow(context.Background(), storage.Bucket{
		Endpoint: refused.URL, Region: "bhs", Name: "b", AccessKeyID: "k", SecretAccessKey: "s",
	}, []string{"https://abc.trenova.app"}))
}

func TestAPrivateEndpointIsWhereTheOperatorReachesTheBucket(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	cors := &stubCORS{}
	s := storage.New(&storage.Params{
		PrivateEndpoint: "http://minio.trenova.internal:9000", Client: client, ControlPlane: &stubCP{value: fetched()},
		CORS: cors, Namespace: namespace, Logger: quiet(),
	})
	loc, err := s.Ensure(ctx, []string{"abc.instances.localhost"}, 1)
	require.NoError(t, err)
	require.Equal(t, "https://s3.bhs.io.cloud.ovh.net", loc.Endpoint)
	require.Equal(t, "http://minio.trenova.internal:9000", loc.PrivateEndpoint)
	require.Equal(t, "http://minio.trenova.internal:9000", cors.buckets[0].Endpoint)
	require.Equal(t, []string{"https://abc.instances.localhost"}, cors.origins)

	again, err := s.Ensure(ctx, []string{"abc.instances.localhost"}, 1)
	require.NoError(t, err)
	require.Equal(t, loc, again)

	require.True(t, storage.ValidPrivateEndpoint(""))
	require.True(t, storage.ValidPrivateEndpoint("http://minio.trenova.internal:9000"))
	require.False(t, storage.ValidPrivateEndpoint("ftp://minio:21"))
	require.False(t, storage.ValidPrivateEndpoint("http://minio:9000/path"))
}
