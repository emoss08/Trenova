package x12

import "testing"

func TestSplitTransactions_Two204s(t *testing.T) {
    segs := []Segment{
        {Tag: "ISA"}, {Tag: "GS"},
        {Tag: "ST", Elements: [][]string{{"204"}, {"0001"}}},
        {Tag: "B2", Elements: [][]string{{""}, {"SCAC"}, {"AAA111"}}},
        {Tag: "SE", Elements: [][]string{{"3"}, {"0001"}}},
        {Tag: "ST", Elements: [][]string{{"204"}, {"0002"}}},
        {Tag: "B2", Elements: [][]string{{""}, {"SCAC"}, {"BBB222"}}},
        {Tag: "SE", Elements: [][]string{{"3"}, {"0002"}}},
        {Tag: "GE"}, {Tag: "IEA"},
    }
    blocks := SplitTransactions(segs)
    if len(blocks) != 2 {
        t.Fatalf("expected 2 transactions, got %d", len(blocks))
    }
    if blocks[0].SetID != "204" || blocks[0].Control != "0001" {
        t.Fatalf("unexpected first tx: %#v", blocks[0])
    }
    if blocks[1].SetID != "204" || blocks[1].Control != "0002" {
        t.Fatalf("unexpected second tx: %#v", blocks[1])
    }
}

