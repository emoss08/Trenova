package oracle

import (
	"cmp"
	"fmt"
	"slices"
)

func (s *GuideSet) Judge(raw []byte) (*Judgement, error) {
	j, err := readFile(raw)
	if err != nil {
		return nil, err
	}
	for _, ic := range j.Interchanges {
		for _, g := range ic.Groups {
			for _, r := range g.Reports {
				s.judgeReport(g, r)
			}
		}
	}
	return j, nil
}

func (s *GuideSet) judgeReport(g *Group, r *Report) {
	gd := s.pick(r.Set, r.Version)
	if gd == nil {
		f := errorAt(LevelTransaction, codeSetNotSupported, "guide for the set",
			fmt.Sprintf("no guide covers set %s version %s", r.Set, r.Version))
		f.SegmentID, f.SegmentIndex, f.ElementPos, f.Value = "ST", 1, 1, r.Set
		if len(r.Segments) > 0 {
			f.Offset = r.Segments[0].Offset
		}
		r.add(f)
		return
	}
	r.Guide, r.GuideFile = gd.file, gd.file
	if gd.Title != "" {
		r.Guide = gd.Title
	}
	if g.FunctionalID != gd.Group {
		r.add(Finding{
			Severity:   SeverityWarning,
			Level:      LevelGroup,
			SegmentID:  "GS",
			ElementPos: 1,
			Value:      g.FunctionalID,
			Rule:       "GS01 " + gd.Group,
			Message: fmt.Sprintf(
				"set %s belongs in group %s, not %s",
				r.Set,
				gd.Group,
				g.FunctionalID,
			),
		})
	}
	placed := walkStructure(gd, r)
	for i, def := range placed {
		if def != nil {
			validateSegment(r, def, &r.Segments[i])
		}
	}
	slices.SortStableFunc(r.Findings, func(a, b Finding) int {
		return cmp.Compare(a.SegmentIndex, b.SegmentIndex)
	})
}
