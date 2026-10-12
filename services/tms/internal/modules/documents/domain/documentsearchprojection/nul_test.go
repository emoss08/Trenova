package documentsearchprojection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestBeforeAppendModelStripsNULFromSearchText(t *testing.T) {
	t.Parallel()

	p := &Projection{
		FileName:     "pod\x00.pdf",
		OriginalName: "POD\x00.pdf",
		Description:  "signed\x00 copy",
		ContentText:  "received\x00 in good order",
	}
	require.NoError(t, p.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))
	assert.Equal(t, "pod.pdf", p.FileName)
	assert.Equal(t, "POD.pdf", p.OriginalName)
	assert.Equal(t, "signed copy", p.Description)
	assert.Equal(t, "received in good order", p.ContentText)
}
