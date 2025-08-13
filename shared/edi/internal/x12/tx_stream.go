package x12

import (
	"io"
)

// TxScanner streams 204 transactions (ST..SE) from a segment stream.
type TxScanner struct {
	seg *SegmentScanner
	cur TxBlock
	err error
}

func NewTxScanner(r io.Reader, d Delimiters) *TxScanner {
	return &TxScanner{seg: NewSegmentScanner(r, d)}
}

func (t *TxScanner) Next() bool {
	if t.err != nil {
		return false
	}
	ctrl := ""
	setID := ""
	for t.seg.Next() {
		s := t.seg.Segment()
		if s.Tag == "ST" || s.Tag == "st" {
			if len(s.Elements) > 0 && len(s.Elements[0]) > 0 {
				setID = s.Elements[0][0]
			}
			if len(s.Elements) > 1 && len(s.Elements[1]) > 0 {
				ctrl = s.Elements[1][0]
			}
			// start collection
			buf := make([]Segment, 0, 64)
			buf = append(buf, s)
			// collect until matching SE
			for t.seg.Next() {
				seg := t.seg.Segment()
				buf = append(buf, seg)
				if seg.Tag == "SE" || seg.Tag == "se" {
					if ctrl == "" ||
						(len(seg.Elements) > 1 && len(seg.Elements[1]) > 0 && seg.Elements[1][0] == ctrl) {
						t.cur = TxBlock{
							Segs:    buf,
							STIndex: buf[0].Index,
							SEIndex: seg.Index,
							Control: ctrl,
							SetID:   setID,
						}
						return true
					}
				}
			}

			t.cur = TxBlock{
				Segs:    buf,
				STIndex: buf[0].Index,
				SEIndex: -1,
				Control: ctrl,
				SetID:   setID,
			}
			return true
		}
	}
	t.err = t.seg.Err()
	return false
}

func (t *TxScanner) Tx() TxBlock { return t.cur }
func (t *TxScanner) Err() error  { return t.err }
