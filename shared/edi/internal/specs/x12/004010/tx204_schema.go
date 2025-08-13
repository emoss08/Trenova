package _004010

// Package _004010 holds minimal 204 transaction structure metadata for X12 004010.
// This is intentionally lightweight for MVP; it will be expanded to formal usage
// and validation rules as we iterate.

// SegmentRule describes expected occurrence of a segment.
type SegmentRule struct {
    Tag string // e.g., "B2"
    Min int    // minimum occurrences
    Max int    // maximum occurrences (-1 for unbounded)
}

// LoopSpec models a loop: an ordered set of segment rules and nested loops.
type LoopSpec struct {
    Name     string
    Rules    []SegmentRule
    Children []LoopSpec
}

// Tx204Spec provides a minimal shape of the 204 transaction's key loops for MVP.
func Tx204Spec() LoopSpec {
    n1 := LoopSpec{
        Name: "N1_LOOP",
        Rules: []SegmentRule{
            {Tag: "N1", Min: 1, Max: -1},
            {Tag: "N3", Min: 0, Max: -1},
            {Tag: "N4", Min: 0, Max: -1},
            {Tag: "REF", Min: 0, Max: -1},
            {Tag: "G61", Min: 0, Max: -1},
        },
    }

    s5 := LoopSpec{
        Name: "S5_LOOP",
        Rules: []SegmentRule{
            {Tag: "S5", Min: 1, Max: -1},
            {Tag: "DTM", Min: 0, Max: -1},
            {Tag: "N1", Min: 0, Max: -1},
            {Tag: "N3", Min: 0, Max: -1},
            {Tag: "N4", Min: 0, Max: -1},
            {Tag: "NTE", Min: 0, Max: -1},
            {Tag: "AT5", Min: 0, Max: -1},
            {Tag: "LX", Min: 0, Max: -1},
        },
    }

    root := LoopSpec{
        Name: "ST_204",
        Rules: []SegmentRule{
            {Tag: "ST", Min: 1, Max: 1},
            {Tag: "B2", Min: 1, Max: 1},
            {Tag: "B2A", Min: 0, Max: 1},
            {Tag: "L11", Min: 0, Max: -1},
            {Tag: "G61", Min: 0, Max: -1},
            {Tag: "NTE", Min: 0, Max: -1},
            {Tag: "N7", Min: 0, Max: -1},
            {Tag: "L3", Min: 0, Max: 1},
            {Tag: "SE", Min: 1, Max: 1},
        },
        Children: []LoopSpec{n1, s5},
    }
    return root
}

