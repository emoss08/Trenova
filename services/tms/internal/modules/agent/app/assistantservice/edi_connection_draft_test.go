package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
)

func TestFamilyOf_AConnectionDraftIsReadFromADocument(t *testing.T) {
	t.Parallel()

	assert.Equal(t, assistantartifact.FamilyExtract,
		assistantartifact.FamilyOf(assistantartifact.KindEDIConnectionDraft, nil))
	assert.NotEmpty(t, assistantartifact.LineageKeyFor(
		assistantartifact.KindEDIConnectionDraft, "edi_draft_connection",
		map[string]any{"documentId": "doc_01"},
	), "drafting the same guide again is a new version of the review")
}
