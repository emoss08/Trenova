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
