package aimodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileInput_SendsPDFsAsDocumentsAndFourImageFormatsAsImages(t *testing.T) {
	t.Parallel()

	var input FileInput
	assert.Equal(t, FileKindDocument, input.Kind("application/pdf"))
	assert.Equal(t, FileKindImage, input.Kind("Image/PNG; charset=binary"))
	assert.Equal(t, FileKindImage, input.Kind("image/jpeg"))
	assert.Equal(t, FileKindImage, input.Kind("image/gif"))
	assert.Equal(t, FileKindImage, input.Kind("image/webp"))
	assert.Equal(t, FileKindNone, input.Kind("image/tiff"))
	assert.Equal(t, FileKindNone, input.Kind("image/heic"))
	assert.Equal(t, FileKindNone, input.Kind("text/csv"))
	assert.Equal(t, FileKindNone, input.Kind(""))
	assert.Equal(t, "image/png", MediaType(" Image/PNG; charset=binary"))
}
