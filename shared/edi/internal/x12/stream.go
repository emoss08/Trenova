package x12

import (
    "bufio"
    "bytes"
    "io"
)

// SegmentScanner iterates segments from an io.Reader using provided delimiters.
type SegmentScanner struct {
    r      *bufio.Reader
    d      Delimiters
    index  int
    seg    Segment
    err    error
}

func NewSegmentScanner(r io.Reader, d Delimiters) *SegmentScanner {
    return &SegmentScanner{r: bufio.NewReader(r), d: d, index: 0}
}

// Next advances to the next segment. It returns false on EOF or error.
func (s *SegmentScanner) Next() bool {
    if s.err != nil {
        return false
    }
    // read until segment terminator
    var buf bytes.Buffer
    for {
        b, err := s.r.ReadByte()
        if err != nil {
            if err == io.EOF {
                // flush any residual token (ignore if empty)
                if buf.Len() == 0 {
                    return false
                }
                // treat as final segment without explicit terminator
                break
            }
            s.err = err
            return false
        }
        if b == s.d.Segment {
            break
        }
        buf.WriteByte(b)
    }
    // Trim whitespace
    data := bytes.TrimSpace(buf.Bytes())
    if len(data) == 0 {
        return s.Next()
    }
    // Split to elements
    elems := splitKeepEmpty(data, s.d.Element)
    if len(elems) == 0 {
        return s.Next()
    }
    tag := string(elems[0])
    elements := make([][]string, 0, len(elems)-1)
    for _, e := range elems[1:] {
        if s.d.Component != 0 && bytes.ContainsRune(e, rune(s.d.Component)) {
            comps := splitKeepEmpty(e, s.d.Component)
            arr := make([]string, len(comps))
            for i := range comps { arr[i] = string(comps[i]) }
            elements = append(elements, arr)
        } else {
            elements = append(elements, []string{string(e)})
        }
    }
    s.seg = Segment{Tag: tag, Elements: elements, Index: s.index}
    s.index++
    return true
}

func (s *SegmentScanner) Segment() Segment { return s.seg }
func (s *SegmentScanner) Err() error       { return s.err }

