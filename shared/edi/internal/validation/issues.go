package validation

import "github.com/emoss08/trenova/shared/edi/internal/x12"

type Severity string

const (
	Error   Severity = "ERROR"
	Warning Severity = "WARN"
)

// Issue represents a validation finding.
type Issue struct {
	Severity        Severity `json:"severity"`
	Code            string   `json:"code"`
	Message         string   `json:"message"`
	SegmentIndex    int      `json:"segment_index"`
	Tag             string   `json:"tag"`
	ElementIndex    int      `json:"element_index,omitempty"`   // 1-based when applicable
	ComponentIndex  int      `json:"component_index,omitempty"` // 1-based when applicable
	Hint            string   `json:"hint,omitempty"`
	// Additional fields for 999 acknowledgments
	Level           string   `json:"level,omitempty"`           // "segment" or "element"
	LoopID          string   `json:"loop_id,omitempty"`         // Loop identifier
	ElementPosition int      `json:"element_position,omitempty"` // Position in segment
	ElementRef      string   `json:"element_ref,omitempty"`     // Element reference number
	BadValue        string   `json:"bad_value,omitempty"`       // Copy of bad data
	Context         string   `json:"context,omitempty"`         // Additional context information
}

// Helper: safe get element/component
func get(s x12.Segment, i, j int) string {
	if i < 0 || i >= len(s.Elements) {
		return ""
	}
	if j < 0 || j >= len(s.Elements[i]) {
		return ""
	}
	return s.Elements[i][j]
}
