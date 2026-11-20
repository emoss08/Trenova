package oracle

import (
	"fmt"
	"strings"
)

type elementCheck struct {
	report *Report
	seg    *Segment
}

func validateSegment(report *Report, def *segmentDef, seg *Segment) {
	c := &elementCheck{report: report, seg: seg}
	for pos := 1; pos <= seg.ElementCount(); pos++ {
		if def.byPos[pos] != nil || seg.Element(pos) == "" {
			continue
		}
		ref := fmt.Sprintf("%s-%02d", def.id, pos)
		if pos > def.lastPos {
			c.add(codeTooManyElements, pos, 0, seg.Element(pos),
				fmt.Sprintf("%s has %d elements", def.id, def.lastPos),
				fmt.Sprintf("%s holds more elements than the guide lists", def.id))
			continue
		}
		c.add(codeNotUsedElementPresent, pos, 0, seg.Element(pos), ref+" not used",
			fmt.Sprintf("%s is not used by this partner but holds a value", ref))
	}
	for _, e := range def.Elements {
		c.element(e)
	}
	for _, note := range def.notes {
		for _, b := range note.check(func(p int) bool { return seg.Element(p) != "" }) {
			c.add(b.code, b.pos, 0, seg.Element(b.pos),
				fmt.Sprintf("%s syntax %s", def.id, note.text),
				fmt.Sprintf("%s-%02d %s (%s)", def.id, b.pos, b.msg, note.text))
		}
	}
}

func (c *elementCheck) add(code string, pos, comp int, value, rule, msg string) {
	f := errorAt(LevelElement, code, rule, msg)
	f.SegmentID, f.SegmentIndex, f.Offset, f.LoopID = c.seg.ID, c.seg.Index, c.seg.Offset, c.seg.LoopID
	f.ElementPos, f.ComponentPos, f.Value = pos, comp, value
	c.report.add(f)
}

func (c *elementCheck) element(e *elementDef) {
	v := c.seg.Element(e.Pos)
	if !c.usage(e, e.Pos, 0, v) {
		return
	}
	reps := c.seg.splitRepetitions(v)
	if len(reps) > e.repeatLimit() {
		c.add(
			codeTooManyRepetitions,
			e.Pos,
			0,
			v,
			fmt.Sprintf("%s repeat %d", e.ref, e.repeatLimit()),
			fmt.Sprintf("%s repeats %d times, more than %d", e.ref, len(reps), e.repeatLimit()),
		)
	}
	for _, rep := range reps {
		if rep == "" {
			continue
		}
		if e.byPos != nil {
			c.composite(e, rep)
			continue
		}
		if strings.IndexByte(rep, c.seg.seps.component) >= 0 {
			c.add(codeTooManyComponents, e.Pos, 0, rep, e.ref+" is not a composite",
				fmt.Sprintf("%s holds components but is a simple element", e.ref))
			continue
		}
		c.value(e, e.Pos, 0, rep)
	}
}

func (c *elementCheck) usage(e *elementDef, pos, comp int, v string) bool {
	if v == "" {
		if e.Usage == "M" {
			c.add(codeElementMissing, pos, comp, "", e.ref+" usage M",
				fmt.Sprintf("mandatory element %s is missing", e.ref))
		}
		return false
	}
	if e.Usage == "N" {
		c.add(codeNotUsedElementPresent, pos, comp, v, e.ref+" usage N",
			fmt.Sprintf("%s is not used by this partner but holds a value", e.ref))
		return false
	}
	return true
}

func (c *elementCheck) composite(e *elementDef, value string) {
	parts := c.seg.splitComponents(value)
	if len(parts) > e.lastPos {
		c.add(
			codeTooManyComponents,
			e.Pos,
			0,
			value,
			fmt.Sprintf("%s components %d", e.ref, e.lastPos),
			fmt.Sprintf("%s has %d components, more than %d", e.ref, len(parts), e.lastPos),
		)
	}
	for i, part := range parts {
		if part != "" && e.byPos[i+1] == nil && i < e.lastPos {
			c.add(
				codeNotUsedElementPresent,
				e.Pos,
				i+1,
				part,
				fmt.Sprintf("%s-%d not used", e.ref, i+1),
				fmt.Sprintf("%s-%d is not used by this partner but holds a value", e.ref, i+1),
			)
		}
	}
	for _, comp := range e.Components {
		v := ""
		if comp.Pos <= len(parts) {
			v = parts[comp.Pos-1]
		}
		if c.usage(comp, e.Pos, comp.Pos, v) {
			c.value(comp, e.Pos, comp.Pos, v)
		}
	}
}

func (c *elementCheck) value(e *elementDef, pos, comp int, v string) {
	if code, why := e.dataType.check(v); code != "" {
		c.add(code, pos, comp, v, fmt.Sprintf("%s type %s", e.ref, e.Type),
			fmt.Sprintf("%s %q %s", e.ref, v, why))
		return
	}
	if n := e.dataType.length(v); n < e.Min {
		c.add(codeTooShort, pos, comp, v, fmt.Sprintf("%s min %d", e.ref, e.Min),
			fmt.Sprintf("%s %q is %d long, shorter than %d", e.ref, v, n, e.Min))
	} else if n > e.Max {
		c.add(codeTooLong, pos, comp, v, fmt.Sprintf("%s max %d", e.ref, e.Max),
			fmt.Sprintf("%s %q is %d long, longer than %d", e.ref, v, n, e.Max))
	}
	if e.Fixed != nil && v != string(*e.Fixed) {
		c.add(codeInvalidCode, pos, comp, v, fmt.Sprintf("%s fixed %s", e.ref, *e.Fixed),
			fmt.Sprintf("%s must be %q, not %q", e.ref, *e.Fixed, v))
		return
	}
	if e.codeSet != nil && !e.codeSet[v] {
		c.add(codeInvalidCode, pos, comp, v,
			fmt.Sprintf("%s codes [%s]", e.ref, scalarsText(e.Codes)),
			fmt.Sprintf("%s %q is not one of the partner's codes", e.ref, v))
	}
}
