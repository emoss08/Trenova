package archivebucket_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/adapters/objectstore/archivebucket"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buckets(allowed ...string) archivebucket.Buckets {
	return archivebucket.New(archivebucket.Params{Managed: &managedconfig.Config{
		Archive: managedconfig.Archive{AllowedEndpoints: allowed},
	}}).(archivebucket.Buckets)
}

func open(b archivebucket.Buckets, endpoint string) error {
	_, err := b.Open(
		&tenantarchive.Location{Endpoint: endpoint, Bucket: "archives", Prefix: "tenants/x/"},
		&tenantarchive.Credentials{AccessKey: "test-access", SecretKey: "test-secret"},
	)
	return err
}

func TestAnArchiveEndpointMustBeAnAllowedHTTPSObjectStore(t *testing.T) {
	t.Parallel()

	allowed := buckets(".r2.cloudflarestorage.com", "s3.us-east-1.amazonaws.com")
	for _, endpoint := range []string{
		"https://acct.r2.cloudflarestorage.com",
		"acct.r2.cloudflarestorage.com",
		"https://s3.us-east-1.amazonaws.com:443",
	} {
		require.NoError(t, open(allowed, endpoint), endpoint)
	}
	for _, endpoint := range []string{
		"http://acct.r2.cloudflarestorage.com",
		"https://r2.cloudflarestorage.com.evil.example",
		"https://evil.example",
		"https://user@acct.r2.cloudflarestorage.com",
		"https://acct.r2.cloudflarestorage.com/path",
		"https://127.0.0.1:9000",
		"https://169.254.169.254",
		"https://[::1]:9000",
		"ftp://acct.r2.cloudflarestorage.com",
	} {
		err := open(allowed, endpoint)
		refusal, ok := tenantarchive.RefusalOf(err)
		require.True(t, ok, "%s: %v", endpoint, err)
		assert.Equal(t, tenantarchive.RefusalLocation, refusal.Code, endpoint)
	}
}

func TestTheAllowListMatchesHostsAndSuffixesOnly(t *testing.T) {
	t.Parallel()

	archive := managedconfig.Archive{
		AllowedEndpoints: []string{".r2.cloudflarestorage.com", "objects.example.com"},
	}
	assert.True(t, archive.EndpointAllowed("acct.r2.cloudflarestorage.com"))
	assert.True(t, archive.EndpointAllowed("OBJECTS.example.com."))
	assert.False(t, archive.EndpointAllowed("r2.cloudflarestorage.com"))
	assert.False(t, archive.EndpointAllowed("xr2.cloudflarestorage.com"))
	assert.False(t, archive.EndpointAllowed("objects.example.com.evil.test"))
	assert.False(t, archive.EndpointAllowed(""))
	assert.True(t, (&managedconfig.Archive{}).EndpointAllowed("any.example.com"),
		"a dedicated instance with no list reaches any public host")
}

func TestTheConfiguredBackupEndpointIsTheOnlyHostTheImportMayReach(t *testing.T) {
	t.Parallel()

	production := archivebucket.ForConfiguredEndpoint(archivebucket.ConfiguredEndpoint{
		Endpoint: "https://s3.bhs.io.cloud.ovh.net",
	}).(archivebucket.Buckets)
	require.NoError(t, open(production, "https://s3.bhs.io.cloud.ovh.net"))
	for _, endpoint := range []string{
		"https://s3.gra.io.cloud.ovh.net",
		"http://s3.bhs.io.cloud.ovh.net",
		"https://10.0.0.5",
	} {
		_, ok := tenantarchive.RefusalOf(open(production, endpoint))
		assert.True(t, ok, endpoint)
	}

	private := archivebucket.ForConfiguredEndpoint(archivebucket.ConfiguredEndpoint{
		Endpoint: "http://10.0.0.5:9000",
	}).(archivebucket.Buckets)
	_, ok := tenantarchive.RefusalOf(open(private, "http://10.0.0.5:9000"))
	assert.True(t, ok, "a private or plain http store is refused outside development")

	development := archivebucket.ForConfiguredEndpoint(archivebucket.ConfiguredEndpoint{
		Endpoint:    "http://minio:9000",
		Development: true,
	}).(archivebucket.Buckets)
	require.NoError(t, open(development, "http://minio:9000"),
		"a development install reaches its own store")
	_, ok = tenantarchive.RefusalOf(open(development, "http://elsewhere:9000"))
	assert.True(t, ok, "even in development only the configured host")
}

func TestTheNightlyStorageEndpointIsCheckedByTheSameRules(t *testing.T) {
	t.Parallel()

	allowed := buckets(".r2.cloudflarestorage.com")
	require.NoError(t, allowed.CheckEndpoint("https://acct.r2.cloudflarestorage.com"))
	for endpoint, rule := range map[string]string{
		"https://evil.example":                 "managed.archive.allowedEndpoints",
		"https://10.0.0.5":                     "private",
		"https://169.254.169.254":              "metadata",
		"http://acct.r2.cloudflarestorage.com": "https URL",
	} {
		err := allowed.CheckEndpoint(endpoint)
		refusal, ok := tenantarchive.RefusalOf(err)
		require.True(t, ok, "%s: %v", endpoint, err)
		assert.Equal(t, tenantarchive.RefusalLocation, refusal.Code, endpoint)
		assert.Contains(t, refusal.Reason, rule, endpoint)
	}
}

func refusingStore(t *testing.T, status int, code string) archiveports.Bucket {
	t.Helper()
	store := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>` + code +
			`</Code><Message>answered for the test</Message></Error>`))
	}))
	t.Cleanup(store.Close)
	development := archivebucket.ForConfiguredEndpoint(archivebucket.ConfiguredEndpoint{
		Endpoint: store.URL, Development: true,
	})
	bucket, err := development.Open(
		&tenantarchive.Location{
			Endpoint: store.URL, Region: "us-east-1", Bucket: "archives", Prefix: "tenants/x/",
		},
		&tenantarchive.Credentials{AccessKey: "test-access", SecretKey: "test-secret"},
	)
	require.NoError(t, err)
	return bucket
}

func TestABucketThatRefusesTheCredentialIsAStorageRefusal(t *testing.T) {
	t.Parallel()

	for code, status := range map[string]int{
		"AccessDenied":          http.StatusForbidden,
		"InvalidAccessKeyId":    http.StatusForbidden,
		"SignatureDoesNotMatch": http.StatusForbidden,
		"NoSuchBucket":          http.StatusNotFound,
	} {
		bucket := refusingStore(t, status, code)
		err := bucket.List(t.Context(), "tenants/x/", func(string, int64) error { return nil })
		refusal, ok := tenantarchive.RefusalOf(err)
		require.True(t, ok, "%s: %v", code, err)
		assert.Equal(t, tenantarchive.RefusalStorage, refusal.Code, code)
		assert.Contains(t, refusal.Reason, code)
		assert.NotContains(t, refusal.Reason, "test-secret")
		assert.NotContains(t, refusal.Reason, "test-access")

		if status != http.StatusForbidden {
			continue
		}
		_, err = bucket.Get(t.Context(), "tenants/x/manifest.json")
		refusal, ok = tenantarchive.RefusalOf(err)
		require.True(t, ok, "a refused read of one object, %s: %v", code, err)
		assert.Equal(t, tenantarchive.RefusalStorage, refusal.Code, code)
		assert.NotErrorIs(t, err, archiveports.ErrMissing, code)
	}
}

func TestAMissingObjectIsNotACredentialRefusal(t *testing.T) {
	t.Parallel()

	_, err := refusingStore(t, http.StatusNotFound, "NoSuchKey").
		Get(t.Context(), "tenants/x/manifest.json")
	require.ErrorIs(t, err, archiveports.ErrMissing)
	_, ok := tenantarchive.RefusalOf(err)
	assert.False(t, ok)
}
