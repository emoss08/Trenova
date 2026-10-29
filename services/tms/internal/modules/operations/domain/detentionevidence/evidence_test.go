package detentionevidence_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildChain(occurrenceID pulid.ID, count int) []*detentionevidence.DetentionEvidence {
	entries := make([]*detentionevidence.DetentionEvidence, 0, count)
	prev := detentionevidence.GenesisHash

	for i := range count {
		entry := &detentionevidence.DetentionEvidence{
			ID:                    pulid.MustNew("dte_"),
			DetentionOccurrenceID: occurrenceID,
			Sequence:              int32(i),
			Kind:                  detentionevidence.EvidenceKindArrival,
			Source:                detentionevidence.EvidenceSourceGeofence,
			Summary:               "geofence crossing recorded",
			ObservedAt:            int64(1000 + i),
			RecordedAt:            int64(1000 + i),
			PrevHash:              prev,
		}
		entry.Seal()
		prev = entry.Hash
		entries = append(entries, entry)
	}

	return entries
}

func TestEvidence_SealAndVerify(t *testing.T) {
	t.Parallel()

	entry := &detentionevidence.DetentionEvidence{
		DetentionOccurrenceID: pulid.MustNew("dto_"),
		Sequence:              0,
		Kind:                  detentionevidence.EvidenceKindArrival,
		Source:                detentionevidence.EvidenceSourceTelematics,
		Summary:               "arrived at facility",
		ObservedAt:            1000,
		RecordedAt:            1000,
		PrevHash:              detentionevidence.GenesisHash,
	}

	entry.Seal()

	assert.NotEmpty(t, entry.Hash)
	assert.Len(t, entry.Hash, 64)
	assert.True(t, entry.Verify())
}

func TestEvidence_TamperingBreaksTheHash(t *testing.T) {
	t.Parallel()

	entry := buildChain(pulid.MustNew("dto_"), 1)[0]
	require.True(t, entry.Verify())

	entry.ObservedAt = 999

	assert.False(t, entry.Verify(),
		"editing a recorded timestamp must be detectable")
}

func TestEvidence_IntactChainVerifies(t *testing.T) {
	t.Parallel()

	chain := buildChain(pulid.MustNew("dto_"), 5)

	assert.Equal(t, -1, detentionevidence.VerifyChain(chain))
}

func TestEvidence_ChainDetectsMiddleTampering(t *testing.T) {
	t.Parallel()

	chain := buildChain(pulid.MustNew("dto_"), 5)
	chain[2].Summary = "backdated arrival"

	assert.Equal(t, 2, detentionevidence.VerifyChain(chain),
		"the first broken link is reported")
}

func TestEvidence_ChainDetectsResealedTampering(t *testing.T) {
	t.Parallel()

	chain := buildChain(pulid.MustNew("dto_"), 5)

	chain[2].Summary = "backdated arrival"
	chain[2].Seal()

	assert.Equal(t, 3, detentionevidence.VerifyChain(chain),
		"resealing a link orphans every successor, so the break still surfaces")
}

func TestEvidence_ChainDetectsDeletion(t *testing.T) {
	t.Parallel()

	chain := buildChain(pulid.MustNew("dto_"), 5)
	shortened := append(chain[:2], chain[3:]...)

	assert.Equal(t, 2, detentionevidence.VerifyChain(shortened),
		"removing an inconvenient record breaks the chain")
}

func TestEvidence_ChainDetectsReordering(t *testing.T) {
	t.Parallel()

	chain := buildChain(pulid.MustNew("dto_"), 4)
	chain[1], chain[2] = chain[2], chain[1]

	assert.NotEqual(t, -1, detentionevidence.VerifyChain(chain))
}

func TestEvidence_EmptyChainIsIntact(t *testing.T) {
	t.Parallel()

	assert.Equal(t, -1, detentionevidence.VerifyChain(nil))
	assert.Equal(t, -1, detentionevidence.VerifyChain([]*detentionevidence.DetentionEvidence{}))
}

func TestEvidence_NilEntryIsABrokenLink(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, detentionevidence.VerifyChain([]*detentionevidence.DetentionEvidence{nil}))
}

// Length-delimited hashing prevents two different field splits from colliding.
func TestEvidence_FieldBoundariesCannotCollide(t *testing.T) {
	t.Parallel()

	occurrenceID := pulid.MustNew("dto_")

	a := &detentionevidence.DetentionEvidence{
		DetentionOccurrenceID: occurrenceID,
		Kind:                  detentionevidence.EvidenceKindArrival,
		Source:                detentionevidence.EvidenceSourceManual,
		Summary:               "abc",
		ObservedAt:            1000,
		RecordedAt:            1000,
		PrevHash:              detentionevidence.GenesisHash,
	}
	b := &detentionevidence.DetentionEvidence{
		DetentionOccurrenceID: occurrenceID,
		Kind:                  detentionevidence.EvidenceKindArrival,
		Source:                detentionevidence.EvidenceSourceManual,
		Summary:               "ab",
		ObservedAt:            1000,
		RecordedAt:            1000,
		PrevHash:              detentionevidence.GenesisHash,
	}

	assert.NotEqual(t, a.ComputeHash(), b.ComputeHash())
}

func TestEvidence_SourceWeighting(t *testing.T) {
	t.Parallel()

	assert.Greater(t,
		detentionevidence.EvidenceSourceGeofence.Weight(),
		detentionevidence.EvidenceSourceManual.Weight(),
		"a geofence crossing is far more defensible than a dispatcher keystroke")

	assert.Greater(t,
		detentionevidence.EvidenceSourceEDI.Weight(),
		detentionevidence.EvidenceSourceDriverApp.Weight())

	assert.Equal(t, 0, detentionevidence.EvidenceSource("Rumor").Weight())
}

func TestEvidence_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*detentionevidence.DetentionEvidence)
		wantField string
	}{
		{
			name:      "missing occurrence",
			mutate:    func(e *detentionevidence.DetentionEvidence) { e.DetentionOccurrenceID = pulid.Nil },
			wantField: "detentionOccurrenceId",
		},
		{
			name:      "missing summary",
			mutate:    func(e *detentionevidence.DetentionEvidence) { e.Summary = "" },
			wantField: "summary",
		},
		{
			name:      "unknown kind",
			mutate:    func(e *detentionevidence.DetentionEvidence) { e.Kind = detentionevidence.EvidenceKind("Vibes") },
			wantField: "kind",
		},
		{
			name:      "unknown source",
			mutate:    func(e *detentionevidence.DetentionEvidence) { e.Source = detentionevidence.EvidenceSource("Vibes") },
			wantField: "source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := buildChain(pulid.MustNew("dto_"), 1)[0]
			tt.mutate(entry)

			multiErr := errortypes.NewMultiError()
			entry.Validate(multiErr)

			require.True(t, multiErr.HasErrors())
			assert.Contains(t, errorFields(multiErr), tt.wantField)
		})
	}
}

func errorFields(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	return fields
}
