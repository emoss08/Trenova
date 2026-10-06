package documentcontent

import (
	"math"

	"github.com/emoss08/trenova/shared/jsonutils"
)

// MetadataLines is the page metadata key that holds where each line of the
// page's text sits, so a value read off the page can be pointed at.
const MetadataLines = "lines"

// MaxLayoutLines bounds what one page keeps: enough for a dense rate
// confirmation, and a ceiling on what a scan full of noise can store.
const MaxLayoutLines = 400

// LayoutLine is one line of a page's text and its box, each coordinate a
// fraction of the page so it draws on a page of any size.
type LayoutLine struct {
	Text string  `json:"t"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

// LinesOf reads the layout a page's metadata holds; none when it was
// extracted before layouts were kept or the reader could not place its text.
func LinesOf(metadata map[string]any) []LayoutLine {
	raw, ok := metadata[MetadataLines]
	if !ok || raw == nil {
		return nil
	}
	var lines []LayoutLine
	if err := jsonutils.Convert(raw, &lines); err != nil {
		return nil
	}

	return lines
}

// Fraction rounds a coordinate to a fraction of the page, kept to four places
// so a page's layout stays small.
func Fraction(value, of float64) float64 {
	if of <= 0 {
		return 0
	}
	fraction := math.Max(0, math.Min(1, value/of))

	return math.Round(fraction*10000) / 10000
}
