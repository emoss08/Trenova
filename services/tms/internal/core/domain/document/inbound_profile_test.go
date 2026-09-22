package document_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SupportsIntelligence is the only gate on extraction. A profile that is valid
// but missing from it produces a document that uploads cleanly and is never
// read — which looks like success everywhere except the thing that mattered.
func TestInboundAttachmentProfile_Extracts(t *testing.T) {
	t.Parallel()

	profile := document.ProcessingProfileInboundAttachment

	require.True(t, profile.IsValid())
	assert.True(t, profile.SupportsIntelligence(),
		"an email attachment that is never extracted is an inbox that does no work")
}

func TestProcessingProfile_NormalizesAndRefuses(t *testing.T) {
	t.Parallel()

	normalized, err := document.NormalizeProcessingProfile("inbound_attachment")
	require.NoError(t, err)
	assert.Equal(t, document.ProcessingProfileInboundAttachment, normalized)

	_, err = document.NormalizeProcessingProfile("inbound_attachments")
	require.Error(t, err, "a near-miss must not quietly become a valid profile")
}

/*
IntelligenceProcessingProfiles is what the reconcile sweep filters on, so it
has to agree exactly with SupportsIntelligence — the gate the enqueue uses.

The two disagreeing is how the profile stopped being a gate at all: the sweep
had no profile filter, started the pipeline directly, and ran OCR and a model
over every document that had not been extracted yet, ten minutes after it
landed, whatever its profile said.
*/
func TestIntelligenceProfiles_AgreeWithTheGate(t *testing.T) {
	t.Parallel()

	reading := document.IntelligenceProcessingProfiles()

	for _, profile := range document.AllProcessingProfiles() {
		if profile.SupportsIntelligence() {
			assert.Containsf(t, reading, profile,
				"%s is read on upload but the sweep would skip it", profile)

			continue
		}
		assert.NotContainsf(t, reading, profile,
			"%s says not to read the document, and the sweep would read it anyway", profile)
	}
}

// None must never be in the set. It is the profile every document defaults to,
// so including it would mean reading everything ever uploaded.
func TestIntelligenceProfiles_ExcludeNone(t *testing.T) {
	t.Parallel()

	assert.NotContains(t,
		document.IntelligenceProcessingProfiles(), document.ProcessingProfileNone)
	assert.False(t, document.ProcessingProfileNone.SupportsIntelligence())
}

// Every profile that exists is in the list, or a new one silently never gets
// read and never gets swept.
func TestAllProcessingProfiles_IsComplete(t *testing.T) {
	t.Parallel()

	for _, profile := range document.AllProcessingProfiles() {
		assert.Truef(t, profile.IsValid(), "%s is listed but not valid", profile)
	}
	assert.Len(t, document.AllProcessingProfiles(), 4)
}
