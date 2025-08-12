package x12

// Delimiters represent the X12 separators used in an interchange.
type Delimiters struct {
	Element    byte `json:"element"`    // typically '*'
	Component  byte `json:"component"`  // typically '>' or ':'
	Repetition byte `json:"repetition"` // often '^' in 5010+; may be 0 in 4010
	Segment    byte `json:"segment"`    // typically '~'
}

// Segment is a parsed X12 segment with its tag and elements.
// Elements are split into components if present.
type Segment struct {
	Tag      string     `json:"tag"`
	Elements [][]string `json:"elements"`
	Index    int        `json:"index"`
}
