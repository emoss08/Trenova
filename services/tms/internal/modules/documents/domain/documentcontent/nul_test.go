package documentcontent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestBeforeAppendModelStripsNULFromExtractedText(t *testing.T) {
	t.Parallel()

	content := &Content{ContentText: "Rate\x00 Confirmation", FailureMessage: "page\x00 3"}
	require.NoError(t, content.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))
	assert.Equal(t, "Rate Confirmation", content.ContentText)
	assert.Equal(t, "page 3", content.FailureMessage)

	page := &Page{ExtractedText: "\x00BOL 4471\x00"}
	require.NoError(t, page.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil)))
	assert.Equal(t, "BOL 4471", page.ExtractedText)
}
