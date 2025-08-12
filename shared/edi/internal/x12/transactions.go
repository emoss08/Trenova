package x12

// TxBlock represents a single X12 transaction bounded by ST..SE.
type TxBlock struct {
    Segs     []Segment
    STIndex  int
    SEIndex  int
    Control  string
    SetID    string // ST01 e.g., "204"
}

// SplitTransactions finds ST..SE pairs in the given segments and returns
// a list of transaction blocks. Matching is done via ST02==SE02 when possible;
// otherwise the first SE after ST is used.
func SplitTransactions(segs []Segment) []TxBlock {
    blocks := make([]TxBlock, 0, 2)
    n := len(segs)
    for i := 0; i < n; i++ {
        s := segs[i]
        if s.Tag != "ST" && s.Tag != "st" {
            continue
        }
        setID := ""
        if len(s.Elements) >= 1 && len(s.Elements[0]) > 0 {
            setID = s.Elements[0][0]
        }
        ctrl := ""
        if len(s.Elements) >= 2 && len(s.Elements[1]) > 0 {
            ctrl = s.Elements[1][0]
        }
        // find matching SE
        match := -1
        for j := i + 1; j < n; j++ {
            if segs[j].Tag != "SE" && segs[j].Tag != "se" {
                continue
            }
            if ctrl == "" || (len(segs[j].Elements) >= 2 && len(segs[j].Elements[1]) > 0 && segs[j].Elements[1][0] == ctrl) {
                match = j
                break
            }
        }
        if match >= 0 {
            blocks = append(blocks, TxBlock{
                Segs:    append([]Segment(nil), segs[i:match+1]...),
                STIndex: i,
                SEIndex: match,
                Control: ctrl,
                SetID:   setID,
            })
            i = match
        }
    }
    return blocks
}

