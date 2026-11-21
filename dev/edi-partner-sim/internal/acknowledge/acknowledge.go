package acknowledge

import (
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

const (
	SegmentHasElementErrors = "8"
	maxCopyLength           = 99
	acknowledgmentGroup     = "FA"
)

type Decision struct {
	Set      string
	Control  string
	Code     string
	Errors   int
	Warnings int
	Notes    []SegmentNote
}

func Decide(
	rules *partner.Acknowledgment,
	ic *oracle.Interchange,
	g *oracle.Group,
	rep *oracle.Report,
) Decision {
	envelopeErrors, envelopeWarnings := Count(ic.Findings)
	groupErrors, groupWarnings := Count(g.Findings)
	errs, warns := Count(rep.Findings)
	errs += envelopeErrors + groupErrors
	warns += envelopeWarnings + groupWarnings
	return Decision{
		Set:      rep.Set,
		Control:  rep.ControlNumber,
		Code:     Code(rules, rep.Set, errs, warns),
		Errors:   errs,
		Warnings: warns,
		Notes:    Notes(rep.Findings, rules.Sends),
	}
}

func Plan(rules *partner.Acknowledgment, j *oracle.Judgement) []Input {
	if rules.Sends == partner.AckNone {
		return nil
	}
	var inputs []Input
	for _, ic := range j.Interchanges {
		for _, g := range ic.Groups {
			if g.FunctionalID == acknowledgmentGroup || len(g.Reports) == 0 {
				continue
			}
			input := Input{
				GroupID:      g.FunctionalID,
				GroupControl: g.ControlNumber,
				Version:      g.Version,
			}
			for _, rep := range g.Reports {
				decision := Decide(rules, ic, g, rep)
				input.Transactions = append(input.Transactions, InputTransaction{
					Set:     decision.Set,
					Control: decision.Control,
					Code:    decision.Code,
					Notes:   decision.Notes,
				})
			}
			inputs = append(inputs, input)
		}
	}
	return inputs
}

func Count(findings []oracle.Finding) (errs, warns int) {
	for i := range findings {
		switch findings[i].Severity {
		case oracle.SeverityError:
			errs++
		case oracle.SeverityWarning:
			warns++
		}
	}
	return errs, warns
}

func Notes(findings []oracle.Finding, sends partner.AckSet) []SegmentNote {
	var notes []SegmentNote
	index := map[int]int{}
	for i := range findings {
		f := &findings[i]
		if f.Severity != oracle.SeverityError || f.SegmentIndex == 0 ||
			(f.Level != oracle.LevelSegment && f.Level != oracle.LevelElement) {
			continue
		}
		at, ok := index[f.SegmentIndex]
		if !ok {
			at = len(notes)
			index[f.SegmentIndex] = at
			notes = append(notes, SegmentNote{
				ID:       f.SegmentID,
				Position: f.SegmentIndex,
				Loop:     f.LoopID,
				Code:     SegmentHasElementErrors,
			})
		}
		note := &notes[at]
		if f.Level == oracle.LevelSegment {
			note.Code = f.Code
			continue
		}
		note.Elements = append(note.Elements, ElementNote{
			Position:  f.ElementPos,
			Component: f.ComponentPos,
			Code:      elementCode(f.Code, sends),
			Copy:      truncate(f.Value, maxCopyLength),
		})
	}
	return notes
}

func TransactionCode(t partner.Transaction) string {
	for _, segment := range t.Segments {
		if (segment.ID == "AK5" || segment.ID == "IK5") && len(segment.Elements) > 0 &&
			len(segment.Elements[0]) > 0 {
			return segment.Elements[0][0]
		}
	}
	return ""
}

func elementCode(code string, sends partner.AckSet) string {
	if sends != partner.Ack999 && strings.HasPrefix(code, "I") {
		return "3"
	}
	return code
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
