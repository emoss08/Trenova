package documentsearchprojection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeforeCreateStripsNULFromSearchText(t *testing.T) {
	t.Parallel()

	p := &Projection{
		FileName:     "pod\x00.pdf",
		OriginalName: "POD\x00.pdf",
		Description:  "signed\x00 copy",
		ContentText:  "received\x00 in good order",
	}
	require.NoError(t, p.BeforeCreate(t.Context()))
	assert.Equal(t, "pod.pdf", p.FileName)
	assert.Equal(t, "POD.pdf", p.OriginalName)
	assert.Equal(t, "signed copy", p.Description)
	assert.Equal(t, "received in good order", p.ContentText)
}
