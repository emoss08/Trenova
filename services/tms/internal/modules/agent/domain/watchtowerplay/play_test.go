package watchtowerplay_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
)

// Each kind of item has the first move a person makes about it, and a kind
// with no move that follows from the record has none.
func TestOfIsTheFirstMoveForTheKind(t *testing.T) {
	t.Parallel()

	assert.Equal(t, watchtowerplay.AskForPOD, watchtowerplay.Of(watchtower.SourceKindPODMissing))
	for _, kind := range []watchtower.SourceKind{
		watchtower.SourceKindMissedPickup, watchtower.SourceKindMissedDelivery, watchtower.SourceKindEtaAtRisk, watchtower.SourceKindNoCheckCall,
	} {
		assert.Equal(t, watchtowerplay.AskForStatus, watchtowerplay.Of(kind), string(kind))
	}
	assert.Equal(
		t,
		watchtowerplay.TenderAgain,
		watchtowerplay.Of(watchtower.SourceKindTenderExhausted),
	)

	for _, kind := range []watchtower.SourceKind{
		watchtower.SourceKindAgentException, watchtower.SourceKindBillingException, watchtower.SourceKindServiceFailure, watchtower.SourceKindInsight,
	} {
		assert.Equal(t, watchtowerplay.None, watchtowerplay.Of(kind), string(kind))
	}
}
