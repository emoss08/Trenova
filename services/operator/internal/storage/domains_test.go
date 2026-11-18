package storage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/storage"
	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	first  = "abc.trenova.app"
	custom = "tms.acme-freight.com"
)

func TestTheBucketAdmitsEveryServedOriginAndIsAskedOnlyWhenTheSetChanges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	cors := &stubCORS{}
	s := storage.New(&storage.Params{
		Client: client, ControlPlane: &stubCP{value: fetched()}, CORS: cors, Namespace: namespace, Logger: quiet(),
	})
	annotation := func() string {
		docs, err := client.CoreV1().Secrets(namespace).Get(ctx, values.DocumentsSecret, metav1.GetOptions{})
		require.NoError(t, err)
		return docs.Annotations["trenova.app/cors-origin"]
	}

	steps := []struct {
		hosts []string
		calls int
	}{
		{[]string{first}, 1},
		{[]string{first}, 1},
		{[]string{first, custom}, 2},
		{[]string{first, custom, ""}, 2},
		{[]string{first, "other.acme-freight.com"}, 3},
		{[]string{first}, 4},
		{[]string{first}, 4},
	}
	for i, step := range steps {
		_, err := s.Ensure(ctx, step.hosts, 1)
		require.NoError(t, err, i)
		require.Len(t, cors.origins, step.calls, "step %d", i)
	}
	require.Equal(t, []string{
		"https://" + first,
		"https://" + first + ",https://" + custom,
		"https://" + first + ",https://other.acme-freight.com",
		"https://" + first,
	}, cors.origins)
	require.Equal(t, "https://"+first, annotation())
}

func TestAFailedCORSCallLeavesTheRecordedSetForTheRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	client := fake.NewSimpleClientset()
	cors := &stubCORS{}
	s := storage.New(&storage.Params{
		Client: client, ControlPlane: &stubCP{value: fetched()}, CORS: cors, Namespace: namespace, Logger: quiet(),
		Now: func() time.Time { return now },
	})
	_, err := s.Ensure(ctx, []string{first}, 1)
	require.NoError(t, err)

	cors.mu.Lock()
	cors.err = errors.New("the bucket answered 503")
	cors.mu.Unlock()
	_, err = s.Ensure(ctx, []string{first, custom}, 2)
	require.NoError(t, err, "a CORS failure never holds converging")
	docs, err := client.CoreV1().Secrets(namespace).Get(ctx, values.DocumentsSecret, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "https://"+first, docs.Annotations["trenova.app/cors-origin"])

	cors.mu.Lock()
	cors.err = nil
	cors.mu.Unlock()
	now = now.Add(6 * time.Minute)
	_, err = s.Ensure(ctx, []string{first, custom}, 2)
	require.NoError(t, err)
	docs, err = client.CoreV1().Secrets(namespace).Get(ctx, values.DocumentsSecret, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "https://"+first+",https://"+custom, docs.Annotations["trenova.app/cors-origin"])
	require.Len(t, cors.origins, 3)
}

func TestOneCORSDocumentCarriesEveryOrigin(t *testing.T) {
	t.Parallel()
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := storage.S3CORS{Client: srv.Client()}

	require.NoError(t, c.Allow(context.Background(), storage.Bucket{
		Endpoint: srv.URL, Region: "bhs", Name: "trenova-abc-docs", AccessKeyID: "k", SecretAccessKey: "s",
	}, []string{"https://" + first, "https://" + custom}))

	doc := string(body)
	require.Equal(t, 1, strings.Count(doc, "<CORSRule>"), "a PUT ?cors replaces the whole configuration")
	require.Contains(t, doc, "<CORSRule><AllowedOrigin>https://"+first+"</AllowedOrigin>"+
		"<AllowedOrigin>https://"+custom+"</AllowedOrigin><AllowedMethod>GET</AllowedMethod>")
	require.Error(t, c.Allow(context.Background(), storage.Bucket{Endpoint: srv.URL, Name: "b"}, nil))
}
