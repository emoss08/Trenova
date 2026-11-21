package archivebucket

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/stretchr/testify/require"
)

func TestTheArchiveDialRefusesAnAddressInsideTheServer(t *testing.T) {
	t.Parallel()

	for _, target := range []string{"https://localhost:9/", "https://127.0.0.1:9/"} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody)
		require.NoError(t, err)
		resp, err := guardedTransport(httpsafe.Policy{}).RoundTrip(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		require.ErrorIs(t, err, httpsafe.ErrBlockedAddress, target)
	}
}
