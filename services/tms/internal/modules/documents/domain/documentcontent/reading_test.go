package documentcontent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/stretchr/testify/assert"
)

const readAt int64 = 1_800_000_000

func readContent(status documentcontent.Status, acceptance string) *documentcontent.Content {
	extracted := readAt
	content := &documentcontent.Content{Status: status, LastExtractedAt: &extracted}
	if acceptance != "" {
		content.StructuredData = map[string]any{
			"aiDiagnostics": map[string]any{"acceptanceStatus": acceptance},
		}
	}

	return content
}

/*
A rate confirmation's text lands before the model has read it, and its findings
are the rule-based ones until the model's arrive. An agent that took the text as
the whole reading drafted a shipment from stop names like "7,000"; the reading
is over only once the model's has landed, or been given up on.
*/
func TestStillBeingRead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content *documentcontent.Content
		now     int64
		want    bool
	}{
		{
			"text still being extracted",
			readContent(documentcontent.StatusExtracting, ""),
			readAt,
			true,
		},
		{
			"text in, no model reading",
			readContent(documentcontent.StatusIndexed, ""),
			readAt,
			false,
		},
		{
			"text in, model reading under way",
			readContent(documentcontent.StatusIndexed, "pending"),
			readAt + 60,
			true,
		},
		{
			"text in, model reading landed",
			readContent(documentcontent.StatusIndexed, "accepted"),
			readAt + 60,
			false,
		},
		{
			"text in, model reading turned down",
			readContent(documentcontent.StatusIndexed, "rejected"),
			readAt + 60,
			false,
		},
		{
			"model reading lost along the way",
			readContent(documentcontent.StatusIndexed, "pending"),
			readAt + documentcontent.ModelReadingWindowSeconds,
			false,
		},
		{
			"could not be read",
			readContent(documentcontent.StatusFailed, "pending"),
			readAt + 60,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.content.StillBeingRead(tt.now))
		})
	}
}
