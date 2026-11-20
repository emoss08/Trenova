package oracle

import (
	"fmt"
	"maps"
	"slices"
)

const maxLoopDepth = 16

type guideChecker struct {
	g     *guide
	errs  []error
	loops map[string]bool
}

func (c *guideChecker) fail(at, format string, args ...any) {
	c.errs = append(c.errs, &GuideError{
		File:    c.g.file,
		Path:    at,
		Message: fmt.Sprintf(format, args...),
	})
}

func (c *guideChecker) check() {
	g := c.g
	required := [][2]string{
		{"set", string(g.Set)},
		{"version", string(g.Version)},
		{"group", g.Group},
	}
	for _, kv := range required {
		if kv[1] == "" {
			c.fail(kv[0], "is required")
		}
	}
	if len(g.Segments) == 0 {
		c.fail("segments", "is required")
	}
	for _, id := range slices.Sorted(maps.Keys(g.Segments)) {
		c.checkSegment(id, g.Segments[id])
	}
	c.loops = make(map[string]bool)
	g.inSet = make(map[string]bool)
	c.checkBody("structure", g.Structure, 0)
	if g.Set != "" && !digitsWithin(string(g.Set), 3, 3) {
		c.fail("set", "%q is not a three-digit transaction set ID", g.Set)
	}
	n := len(g.Structure)
	if n == 0 || g.Structure[0].Seg != "ST" || g.Structure[n-1].Seg != "SE" {
		c.fail("structure", "must start with ST and end with SE")
	}
}

func (c *guideChecker) checkSegment(id string, def *segmentDef) {
	at := "segments." + id
	if def == nil {
		c.fail(at, "has no definition")
		return
	}
	if !validSegmentID(id) {
		c.fail(at, "%q is not a segment ID", id)
	}
	def.id = id
	def.byPos = make(map[int]*elementDef)
	c.checkElements(at+".elements", id, def.Elements, def.byPos, &def.lastPos, false)
	for i, text := range def.Syntax {
		noteAt := fmt.Sprintf("%s.syntax[%d]", at, i)
		note, err := parseSyntaxNote(text)
		if err != nil {
			c.fail(noteAt, "%v", err)
			continue
		}
		for _, p := range note.positions {
			if def.byPos[p] == nil {
				c.fail(noteAt, "%s names element %02d, which the segment does not list", text, p)
			}
		}
		def.notes = append(def.notes, note)
	}
}

func (c *guideChecker) checkElements(
	at, ref string,
	defs []*elementDef,
	byPos map[int]*elementDef,
	lastPos *int,
	component bool,
) {
	for i, e := range defs {
		elAt := fmt.Sprintf("%s[%d]", at, i)
		if e == nil {
			c.fail(elAt, "is empty")
			continue
		}
		if e.Pos < 1 {
			c.fail(elAt+".pos", "must be 1 or more")
			continue
		}
		if byPos[e.Pos] != nil {
			c.fail(elAt+".pos", "position %d is listed twice", e.Pos)
			continue
		}
		if e.Pos < *lastPos {
			c.fail(elAt+".pos", "position %d is out of order", e.Pos)
		}
		byPos[e.Pos] = e
		*lastPos = max(*lastPos, e.Pos)
		if component {
			e.ref = fmt.Sprintf("%s-%d", ref, e.Pos)
		} else {
			e.ref = fmt.Sprintf("%s-%02d", ref, e.Pos)
		}
		c.checkElement(elAt, e, component)
	}
}

func (c *guideChecker) checkElement(at string, e *elementDef, component bool) {
	if !validUsage(e.Usage) {
		c.fail(at+".usage", "%q is not one of M, O, X, N", e.Usage)
	}
	if e.Repeat < 0 || (component && e.Repeat > 1) {
		c.fail(at+".repeat", "%d is not allowed here", e.Repeat)
	}
	if len(e.Components) > 0 {
		if component {
			c.fail(at+".components", "a component cannot have components")
			return
		}
		e.byPos = make(map[int]*elementDef)
		c.checkElements(at+".components", e.ref, e.Components, e.byPos, &e.lastPos, true)
		return
	}
	t, ok := parseDataType(e.Type)
	if !ok {
		c.fail(at+".type", "%q is not one of AN, ID, DT, TM, N0..N9, R", e.Type)
		return
	}
	e.dataType = t
	if e.Min == 0 {
		e.Min = 1
	}
	if e.Max < 1 {
		c.fail(at+".max", "is required and must be 1 or more")
	}
	if e.Min > e.Max {
		c.fail(at+".min", "min %d is greater than max %d", e.Min, e.Max)
	}
	if t.name == "DT" && e.Max != 6 && e.Max != 8 {
		c.fail(at+".max", "a date is 6 or 8 long, not %d", e.Max)
	}
	if t.name == "TM" && (e.Min < 4 || e.Max > 8) {
		c.fail(at, "a time is 4 to 8 long")
	}
	c.checkCodes(at, e)
}

func (c *guideChecker) checkCodes(at string, e *elementDef) {
	if e.Fixed != nil && *e.Fixed == "" {
		e.Fixed = nil
	}
	if len(e.Codes) > 0 {
		e.codeSet = make(map[string]bool, len(e.Codes))
	}
	for i, code := range e.Codes {
		if l := e.dataType.length(string(code)); code == "" || l < e.Min || l > e.Max {
			c.fail(fmt.Sprintf("%s.codes[%d]", at, i), "%q does not fit min %d max %d",
				code, e.Min, e.Max)
		}
		e.codeSet[string(code)] = true
	}
	if e.Fixed != nil && e.codeSet != nil && !e.codeSet[string(*e.Fixed)] {
		c.fail(at+".fixed", "%q is not one of the codes", *e.Fixed)
	}
}

func (c *guideChecker) checkBody(at string, body []*entry, depth int) {
	if depth > maxLoopDepth {
		c.fail(at, "loops nest deeper than %d", maxLoopDepth)
		return
	}
	for i, e := range body {
		entryAt := fmt.Sprintf("%s[%d]", at, i)
		if e == nil {
			c.fail(entryAt, "is empty")
			continue
		}
		if !validUsage(e.Usage) {
			c.fail(entryAt+".usage", "%q is not one of M, O, X, N", e.Usage)
		}
		if e.Max < 0 {
			c.fail(entryAt+".max", "must be 1 or more")
		}
		switch {
		case e.Seg != "" && e.Loop != "":
			c.fail(entryAt, "is both a segment and a loop")
		case e.Seg != "":
			if len(e.Body) > 0 {
				c.fail(entryAt+".body", "a segment use has no body")
			}
			if base := c.g.Segments[e.Seg]; base == nil {
				c.fail(entryAt+".seg", "segment %s is not defined under segments", e.Seg)
			} else {
				c.narrow(entryAt, e, base)
			}
			c.g.inSet[e.Seg] = true
		case e.Loop != "":
			if len(e.Elements) > 0 {
				c.fail(entryAt+".elements", "a loop has no elements to override")
			}
			c.checkLoop(entryAt, e, depth)
		default:
			c.fail(entryAt, "names neither a segment nor a loop")
		}
	}
}

func (c *guideChecker) checkLoop(at string, e *entry, depth int) {
	if c.loops[e.Loop] {
		c.fail(at+".loop", "loop %s is defined twice", e.Loop)
	}
	c.loops[e.Loop] = true
	if len(e.Body) == 0 || e.Body[0] == nil || e.Body[0].Seg == "" {
		c.fail(at+".body", "a loop's first entry must be its trigger segment")
		return
	}
	c.checkBody(at+".body", e.Body, depth+1)
}

func validUsage(u string) bool {
	return u == "M" || u == "O" || u == "X" || u == "N"
}

func validSegmentID(id string) bool {
	if len(id) < 2 || len(id) > 3 || id[0] < 'A' || id[0] > 'Z' {
		return false
	}
	for i := 1; i < len(id); i++ {
		b := id[i]
		if (b < 'A' || b > 'Z') && (b < '0' || b > '9') {
			return false
		}
	}
	return true
}

func (e *entry) segment(g *guide) *segmentDef {
	if e.def != nil {
		return e.def
	}
	return g.Segments[e.Seg]
}

func (e *entry) limit() int {
	return max(e.Max, 1)
}

func (e *entry) triggerID() string {
	if e.Seg != "" {
		return e.Seg
	}
	return e.Body[0].Seg
}

func (e *elementDef) repeatLimit() int {
	return max(e.Repeat, 1)
}
