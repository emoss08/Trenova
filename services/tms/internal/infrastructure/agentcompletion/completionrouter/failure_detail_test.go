package completionrouter

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestFailureDetail(t *testing.T) {
	t.Parallel()

	anthropic := &aiprovider.Provider{Name: "Claude", Kind: aiprovider.KindAnthropicMessages}
	assert.Equal(t, "Anthropic returned 529 twice", failureDetail(anthropic, "Overloaded", 529, 2, time.Second))
	assert.Equal(t, "Anthropic returned 500", failureDetail(anthropic, "Failed", 500, 1, time.Second))
	assert.Equal(t, "No response after 30s", failureDetail(anthropic, "Timed out", 0, 1, 30*time.Second))
	assert.Equal(t, "Anthropic could not be reached 3 times", failureDetail(anthropic, "Unavailable", 0, 3, time.Second))
}

func TestDistinctFailuresCountsAttempts(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("aip_")
	out := distinctFailures([]serviceports.ChatProviderFailure{
		{ProviderID: id, Vendor: "anthropic", Status: "Overloaded", HTTPStatus: 529, Attempts: 1},
		{ProviderID: id, Vendor: "anthropic", Status: "Overloaded", HTTPStatus: 529, Attempts: 1},
	})

	assert.Len(t, out, 1)
	assert.Equal(t, 2, out[0].Attempts)
	assert.Equal(t, "Anthropic returned 529 twice", out[0].Detail)
}
