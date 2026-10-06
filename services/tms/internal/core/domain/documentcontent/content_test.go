package documentcontent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoorlyRead(t *testing.T) {
	t.Parallel()

	scan := func(confidence ...float64) []*Page {
		pages := make([]*Page, 0, len(confidence))
		for _, c := range confidence {
			pages = append(pages, &Page{SourceKind: SourceKindOCR, OCRConfidence: c})
		}
		return pages
	}

	tests := []struct {
		name    string
		content *Content
		want    bool
	}{
		{"nothing to read", nil, false},
		{"still reading", &Content{Status: StatusPending, Pages: scan(0.1)}, false},
		{"reading failed", &Content{Status: StatusFailed}, false},
		{"a clear scan", &Content{Status: StatusExtracted, Pages: scan(0.92, 0.88)}, false},
		{"indexed after reading", &Content{Status: StatusIndexed, Pages: scan(0.2)}, true},
		{"a blurred photo", &Content{Status: StatusExtracted, Pages: scan(0.31)}, true},
		{"mostly blurred", &Content{Status: StatusExtracted, Pages: scan(0.2, 0.3, 0.9)}, true},
		{
			"native text never counts as poorly read",
			&Content{
				Status: StatusExtracted,
				Pages:  []*Page{{SourceKind: SourceKindNative, OCRConfidence: 0}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.content.PoorlyRead())
		})
	}
}
