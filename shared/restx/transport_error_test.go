package restx_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

func TestSanitizeTransportErrorDropsTheURLAndItsQuery(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://api.example.com/v2?api_key=the-secret",
		http.NoBody,
	)
	require.NoError(t, err)

	_, err = (&http.Client{Transport: failingTransport{}}).Do(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "the-secret")

	assert.EqualError(t, restx.SanitizeTransportError(err), "connection refused")
}
