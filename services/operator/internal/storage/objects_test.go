package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/values"
)

type bucketStub struct {
	mu      sync.Mutex
	objects map[string][]byte
	meta    map[string]http.Header
	auth    []string
	status  int
}

func (b *bucketStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.auth = append(b.auth, r.Header.Get("Authorization"))
	if b.status != 0 {
		w.WriteHeader(b.status)
		return
	}
	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		b.objects[r.URL.Path] = body
		b.meta[r.URL.Path] = r.Header.Clone()
	case http.MethodHead:
		body, ok := b.objects[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		for k, v := range b.meta[r.URL.Path] {
			if strings.HasPrefix(k, metaPrefix) {
				w.Header()[k] = v
			}
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	w.WriteHeader(http.StatusOK)
}

func documents(t *testing.T, withSecret bool) (*bucketStub, *Documents) {
	t.Helper()
	stub := &bucketStub{objects: map[string][]byte{}, meta: map[string]http.Header{}}
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)
	kube := fake.NewClientset()
	if withSecret {
		_, err := kube.CoreV1().Secrets("trenova").Create(t.Context(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: values.DocumentsSecret, Namespace: "trenova"},
			Data: map[string][]byte{
				KeyAccessKey: []byte("documents-access-key-0001"),
				KeySecretKey: []byte("documents-secret-key-0001"),
				KeyEndpoint:  []byte("https://s3.bhs.io.cloud.ovh.net"),
				KeyRegion:    []byte("bhs"),
				KeyBucket:    []byte("trenova-x-docs"),
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}
	return stub, NewDocuments(&DocumentsParams{
		Client: srv.Client(), Kube: kube, Namespace: "trenova", PrivateEndpoint: srv.URL,
	})
}

func TestAnObjectIsPutSignedIntoTheDocumentsBucketAndFoundAgain(t *testing.T) {
	t.Parallel()
	stub, docs := documents(t, true)
	body := []byte("zip bytes")
	sum := sha256.Sum256(body)

	require.NoError(t, docs.Put(t.Context(), &Object{
		Key: "support/logs/b.zip", Body: bytes.NewReader(body), Size: int64(len(body)),
		SHA256: hex.EncodeToString(sum[:]), ContentType: "application/zip",
		Metadata: map[string]string{"components": "api:10:0"},
	}))
	info, ok, err := docs.Head(t.Context(), "support/logs/b.zip")
	require.NoError(t, err)
	require.True(t, ok)
	_, missing, err := docs.Head(t.Context(), "support/logs/other.zip")
	require.NoError(t, err)

	assert.Equal(t, body, stub.objects["/trenova-x-docs/support/logs/b.zip"])
	assert.Equal(t, "api:10:0", info.Metadata["components"])
	assert.False(t, missing)
	for _, a := range stub.auth {
		assert.Contains(t, a, "AWS4-HMAC-SHA256 Credential=documents-access-key-0001/")
		assert.Contains(t, a, "/bhs/s3/aws4_request")
		assert.NotContains(t, a, "documents-secret-key-0001")
	}
}

func TestWithoutTheDocumentsSecretThereIsNoBucket(t *testing.T) {
	t.Parallel()
	_, docs := documents(t, false)

	_, _, err := docs.Head(t.Context(), "support/logs/b.zip")

	require.ErrorIs(t, err, ErrNoDocumentsBucket)
}

func TestAServerErrorIsRetryableAndARefusalIsNot(t *testing.T) {
	t.Parallel()
	stub, docs := documents(t, true)
	stub.status = http.StatusServiceUnavailable
	put := func() error {
		return docs.Put(t.Context(), &Object{Key: "k", Body: bytes.NewReader(nil), SHA256: emptySHA256})
	}

	var oe *ObjectError
	require.ErrorAs(t, put(), &oe)
	assert.True(t, oe.Retryable())
	stub.status = http.StatusForbidden
	require.ErrorAs(t, put(), &oe)
	assert.False(t, oe.Retryable())
}
