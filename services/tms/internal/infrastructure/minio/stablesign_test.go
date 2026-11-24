package minio

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func presignedByMinio(t *testing.T, signedAt string) *url.URL {
	t.Helper()
	u, err := url.Parse("https://storage.test/bucket/org_1/worker/photo/wrk_1/a.jpg" +
		"?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
		"&X-Amz-Credential=AKID%2F20261124%2Fus-east-1%2Fs3%2Faws4_request" +
		"&X-Amz-Date=" + signedAt +
		"&X-Amz-Expires=600&X-Amz-SignedHeaders=host&X-Amz-Signature=abc" +
		"&response-content-disposition=inline")
	require.NoError(t, err)
	return u
}

func TestResign_TheSameWindowGivesTheSameLink(t *testing.T) {
	t.Parallel()

	signer := newStableSigner("AKID", "secret", "")
	signedAt := time.Date(2026, 11, 24, 12, 30, 0, 0, time.UTC)

	first, err := signer.resign(
		t.Context(),
		presignedByMinio(t, "20261124T123412Z"),
		70*time.Minute,
		signedAt,
	)
	require.NoError(t, err)
	second, err := signer.resign(
		t.Context(),
		presignedByMinio(t, "20261124T123958Z"),
		70*time.Minute,
		signedAt,
	)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	parsed, err := url.Parse(first)
	require.NoError(t, err)
	query := parsed.Query()
	assert.Equal(t, "20261124T123000Z", query.Get("X-Amz-Date"))
	assert.Equal(t, "4200", query.Get("X-Amz-Expires"))
	assert.Equal(t, "inline", query.Get("response-content-disposition"))
	assert.NotEqual(t, "abc", query.Get("X-Amz-Signature"))
}

func TestResign_AnonymousStorageKeepsMinioLink(t *testing.T) {
	t.Parallel()

	presigned := presignedByMinio(t, "20261124T123412Z")
	got, err := newStableSigner("", "", "").resign(
		t.Context(),
		presigned,
		time.Hour,
		time.Now(),
	)

	require.NoError(t, err)
	assert.Equal(t, presigned.String(), got)
}

func TestRegionOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "us-east-1", regionOf("AKID/20261124/us-east-1/s3/aws4_request"))
	assert.Empty(t, regionOf("garbage"))
}
