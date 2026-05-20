package exchangerateservice

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNew_UsesDefaultHTTPClientWithTimeout(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger: zap.NewNop(),
	})

	assert.NotNil(t, svc.httpClient)
	assert.Equal(t, 10*time.Second, svc.httpClient.Timeout)
	assert.NotNil(t, svc.httpClient.Transport)
}

func TestNew_UsesInjectedHTTPClient(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: 2 * time.Second}
	svc := New(Params{
		Logger:     zap.NewNop(),
		HTTPClient: client,
	})

	assert.Same(t, client, svc.httpClient)
}
