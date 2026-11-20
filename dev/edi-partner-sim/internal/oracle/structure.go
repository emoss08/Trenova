package oracle

import "fmt"

type frame struct {
	loop   *entry
	body   []*entry
	cursor int
	counts []int
}

type walker struct {
	g      *guide
	stack  []*frame
	report *Report
	noSE   bool
}

func walkStructure(g *guide, report *Report) []*segmentDef {
	w := &walker{g: g, report: report}
	w.stack = []*frame{{body: g.Structure, cursor: -1, counts: make([]int, len(g.Structure))}}
	placed := make([]*segmentDef, len(report.Segments))
	for i := range report.Segments {
		s := &report.Segments[i]
		if use := w.place(s); use != nil {
			placed[i] = use.segment(g)
		}
	}
	at := Segment{}
	if n := len(report.Segments); n > 0 {
		at = report.Segments[n-1]
	}
	w.noSE = at.ID != "SE"
	for len(w.stack) > 0 {
		w.pop(&at)
	}
	return placed
}

func (f *frame) loopID() string {
	if f.loop == nil {
		return transactionLoopID
	}
	return f.loop.Loop
}

func (f *frame) find(id string) int {
	start := max(f.cursor, 0)
	if f.loop != nil {
		start = max(start, 1)
	}
	full := -1
	for k := start; k < len(f.body); k++ {
		e := f.body[k]
		if e.triggerID() != id {
			continue
		}
		if k == f.cursor && f.counts[k] >= e.limit() {
			if full < 0 {
				full = k
			}
			continue
		}
		return k
	}
	return full
}

func (w *walker) place(s *Segment) *entry {
	for depth := len(w.stack) - 1; depth >= 0; depth-- {
		k := w.stack[depth].find(s.ID)
		if k < 0 {
			continue
		}
		for len(w.stack)-1 > depth {
			w.pop(s)
		}
		return w.advance(w.stack[depth], k, s)
	}
	w.unplaced(s)
	return nil
}

func (w *walker) pop(at *Segment) {
	f := w.stack[len(w.stack)-1]
	for k := f.cursor + 1; k < len(f.body); k++ {
		w.missing(f, k, at)
	}
	w.stack = w.stack[:len(w.stack)-1]
}

func (w *walker) advance(f *frame, k int, s *Segment) *entry {
	for j := f.cursor + 1; j < k; j++ {
		w.missing(f, j, s)
	}
	e := f.body[k]
	f.counts[k]++
	f.cursor = k
	if e.Seg != "" {
		s.LoopID = f.loopID()
		if e.Usage == "N" {
			w.segmentFinding(s, codeSegmentUnexpected, e.Seg+" usage N",
				fmt.Sprintf("%s is not used by this partner", s.ID))
		}
		if f.counts[k] > e.limit() {
			w.segmentFinding(s, codeSegmentOverMax, fmt.Sprintf("%s max %d", e.Seg, e.limit()),
				fmt.Sprintf("%s appears more than %d times", s.ID, e.limit()))
		}
		return e
	}
	s.LoopID = e.Loop
	child := &frame{loop: e, body: e.Body, counts: make([]int, len(e.Body))}
	child.counts[0] = 1
	w.stack = append(w.stack, child)
	if f.counts[k] > e.limit() {
		w.segmentFinding(s, codeLoopOverMax, fmt.Sprintf("loop %s max %d", e.Loop, e.limit()),
			fmt.Sprintf("loop %s repeats more than %d times", e.Loop, e.limit()))
	}
	if e.Usage == "N" {
		w.segmentFinding(s, codeSegmentUnexpected, "loop "+e.Loop+" usage N",
			fmt.Sprintf("loop %s is not used by this partner", e.Loop))
	}
	trigger := e.Body[0]
	if trigger.Usage == "N" {
		w.segmentFinding(s, codeSegmentUnexpected, trigger.Seg+" usage N",
			fmt.Sprintf("%s is not used by this partner", s.ID))
	}
	return trigger
}

func (w *walker) missing(f *frame, k int, at *Segment) {
	e := f.body[k]
	if f.counts[k] > 0 || e.Usage != "M" || (w.noSE && e.Seg == "SE") {
		return
	}
	rule := e.Seg + " usage M"
	msg := fmt.Sprintf("mandatory segment %s is missing", e.Seg)
	loopID := f.loopID()
	if e.Loop != "" {
		rule = "loop " + e.Loop + " usage M"
		msg = fmt.Sprintf("mandatory loop %s (%s) is missing", e.Loop, e.triggerID())
		loopID = e.Loop
	}
	fd := errorAt(LevelSegment, codeSegmentMissing, rule, msg)
	fd.SegmentID, fd.SegmentIndex, fd.Offset, fd.LoopID = e.triggerID(), at.Index, at.Offset, loopID
	w.report.add(fd)
}

func (w *walker) unplaced(s *Segment) {
	switch {
	case !validSegmentID(s.ID):
		w.segmentFinding(s, codeSegmentUnrecognized, "segment ID",
			fmt.Sprintf("%q is not a segment ID", s.ID))
	case !w.g.inSet[s.ID]:
		w.segmentFinding(s, codeSegmentNotInSet, "structure of "+string(w.g.Set),
			fmt.Sprintf("%s is not part of the %s guide", s.ID, w.g.Set))
	default:
		w.segmentFinding(s, codeSegmentOutOfSequence, "structure of "+string(w.g.Set),
			fmt.Sprintf("%s cannot appear here", s.ID))
	}
	s.LoopID = w.stack[len(w.stack)-1].loopID()
}

func (w *walker) segmentFinding(s *Segment, code, rule, msg string) {
	f := errorAt(LevelSegment, code, rule, msg)
	f.SegmentID, f.SegmentIndex, f.Offset = s.ID, s.Index, s.Offset
	f.LoopID = w.stack[len(w.stack)-1].loopID()
	w.report.add(f)
}
