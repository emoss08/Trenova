package ediconnectiondraft

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/x12field"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
)

type candidate struct {
	value      string
	confidence float64
	source     ediconnectiondraftports.DraftSource
	status     ediconnectiondraftports.ValueStatus
	problem    string
}

func (c *candidate) public() ediconnectiondraftports.DraftCandidate {
	return ediconnectiondraftports.DraftCandidate{
		Value:      c.value,
		Confidence: c.confidence,
		Source:     c.source,
		Problem:    c.problem,
	}
}

type evidence struct {
	byField     map[string][]*candidate
	resolutions map[string]ediconnectiondraftports.DraftResolutionInput
}

func newEvidence() *evidence {
	return &evidence{
		byField:     make(map[string][]*candidate),
		resolutions: make(map[string]ediconnectiondraftports.DraftResolutionInput),
	}
}

func (e *evidence) add(field string, c *candidate) {
	e.byField[field] = append(e.byField[field], c)
}

func readValue(
	pages *pageIndex,
	spec *fieldSpec,
	input *ediconnectiondraftports.DraftValueInput,
) *candidate {
	raw := input.Value
	c := &candidate{value: strings.TrimSpace(raw), confidence: input.Confidence}

	check := pages.locate(input.Citation)
	page := check.page
	if page == 0 {
		page = input.Citation.Page
	}
	c.source = documentSource(page, input.Citation, check.problem)

	normalized, err := spec.normalize(raw)
	switch {
	case !check.ok:
		c.status = ediconnectiondraftports.ValueStatusUnverified
		c.problem = check.problem
	case err != nil:
		c.status = ediconnectiondraftports.ValueStatusInvalid
		c.problem = fmt.Sprintf("%s %s", spec.label, err.Error())
	case !valueInQuote(spec, normalized, input.Citation.Quote) &&
		!valueInQuote(spec, strings.TrimSpace(raw), input.Citation.Quote):
		c.value = normalized
		c.status = ediconnectiondraftports.ValueStatusUnverified
		c.problem = "The quote does not contain this value"
	case input.Confidence < confidenceFloor:
		c.value = normalized
		c.status = ediconnectiondraftports.ValueStatusHeld
		c.problem = fmt.Sprintf(
			"Confidence %.2f is below %.2f",
			input.Confidence,
			confidenceFloor,
		)
	default:
		c.value = normalized
		c.status = ediconnectiondraftports.ValueStatusAccepted
	}
	if c.status == ediconnectiondraftports.ValueStatusInvalid ||
		c.status == ediconnectiondraftports.ValueStatusUnverified {
		c.value = x12field.RedactSecrets(c.value)
	}

	return c
}

type fallback struct {
	value  string
	source ediconnectiondraftports.DraftSource
	status ediconnectiondraftports.ValueStatus
}

func resolveField(
	spec *fieldSpec,
	e *evidence,
	current string,
	alternative *fallback,
	required bool,
) ediconnectiondraftports.DraftValue {
	out := &ediconnectiondraftports.DraftValue{
		Field:    spec.key,
		Label:    spec.label,
		Current:  current,
		Required: required,
	}
	candidates := e.byField[spec.key]
	for _, c := range candidates {
		out.Candidates = append(out.Candidates, c.public())
	}

	switch resolution, resolved := e.resolutions[spec.key]; {
	case resolved:
		applyResolution(spec, out, candidates, &resolution)
	case len(candidates) > 0:
		resolveCandidates(out, candidates)
	default:
		resolveAbsent(out, alternative, required)
	}
	return *out
}

func resolveCandidates(out *ediconnectiondraftports.DraftValue, candidates []*candidate) {
	verified := make([]*candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.status == ediconnectiondraftports.ValueStatusAccepted ||
			c.status == ediconnectiondraftports.ValueStatusHeld {
			verified = append(verified, c)
		}
	}
	values := distinctValues(verified)

	switch {
	case len(values) > 1:
		out.Status = ediconnectiondraftports.ValueStatusHeld
		out.Reason = fmt.Sprintf(
			"The document gives %d different values: %s",
			len(values),
			quoteValues(values),
		)
		keepCurrent(out)
	case len(values) == 1:
		best := strongest(verified)
		if best.status != ediconnectiondraftports.ValueStatusAccepted && out.Current != "" &&
			best.value == out.Current {
			out.Value = out.Current
			out.Status = ediconnectiondraftports.ValueStatusKept
			out.Confidence = best.confidence
			out.Source = best.source
			out.Reason = "The guide's value is the one already saved, so nothing changes"
			out.Change = ediconnectiondraftports.ChangeKindSame
			return
		}
		if best.status != ediconnectiondraftports.ValueStatusAccepted {
			out.Status = ediconnectiondraftports.ValueStatusHeld
			out.Confidence = best.confidence
			out.Reason = best.problem
			keepCurrent(out)
			return
		}
		out.Value = best.value
		out.Status = ediconnectiondraftports.ValueStatusAccepted
		out.Confidence = best.confidence
		out.Source = best.source
		if len(candidates) == 1 {
			out.Candidates = nil
		}
		withChange(out)
	default:
		first := candidates[0]
		out.Status = first.status
		out.Reason = first.problem
		keepCurrent(out)
	}
}

func resolveAbsent(out *ediconnectiondraftports.DraftValue, alternative *fallback, required bool) {
	out.Candidates = nil
	switch {
	case out.Current != "":
		out.Value = out.Current
		out.Status = ediconnectiondraftports.ValueStatusKept
		out.Source = ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindConfiguration,
			Note: "The partner's current setting; the document does not mention it",
		}
		out.Change = ediconnectiondraftports.ChangeKindSame
	case alternative != nil && alternative.value != "":
		out.Value = alternative.value
		out.Status = alternative.status
		out.Source = alternative.source
		out.Change = ediconnectiondraftports.ChangeKindAdd
	case required:
		out.Status = ediconnectiondraftports.ValueStatusMissing
		out.Reason = "The document does not give it"
		out.Change = ediconnectiondraftports.ChangeKindNone
	default:
		out.Status = ediconnectiondraftports.ValueStatusMissing
		out.Change = ediconnectiondraftports.ChangeKindNone
	}
}

func applyResolution(
	spec *fieldSpec,
	out *ediconnectiondraftports.DraftValue,
	candidates []*candidate,
	resolution *ediconnectiondraftports.DraftResolutionInput,
) {
	normalized, err := spec.normalize(resolution.Value)
	if err != nil {
		out.Status = ediconnectiondraftports.ValueStatusInvalid
		out.Reason = fmt.Sprintf("The chosen value is not valid: %s %s", spec.label, err)
		keepCurrent(out)
		return
	}

	switch resolution.Source {
	case ediconnectiondraftports.ResolutionSourceCandidate:
		for _, c := range candidates {
			if c.value != normalized ||
				(c.status != ediconnectiondraftports.ValueStatusAccepted && c.status != ediconnectiondraftports.ValueStatusHeld) {
				continue
			}
			out.Value = normalized
			out.Status = ediconnectiondraftports.ValueStatusAccepted
			out.Confidence = c.confidence
			out.Source = c.source
			out.Source.Note = joinNotes(c.source.Note, "Chosen by the person")
			withChange(out)
			return
		}
		out.Status = ediconnectiondraftports.ValueStatusInvalid
		out.Reason = fmt.Sprintf(
			"%q is not one of the values the document gives for %s",
			normalized,
			spec.label,
		)
		keepCurrent(out)
	case ediconnectiondraftports.ResolutionSourcePerson:
		out.Value = normalized
		out.Status = ediconnectiondraftports.ValueStatusAccepted
		out.Source = ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindPerson,
			Note: joinNotes(
				"Given by the person in this conversation",
				strings.TrimSpace(resolution.Note),
			),
		}
		withChange(out)
	default:
		out.Status = ediconnectiondraftports.ValueStatusInvalid
		out.Reason = "A resolution must say whether it picks a cited value or one the person gave"
		keepCurrent(out)
	}
}

func keepCurrent(out *ediconnectiondraftports.DraftValue) {
	if out.Current != "" {
		out.Value = out.Current
		out.Change = ediconnectiondraftports.ChangeKindSame
		return
	}
	out.Value = ""
	out.Change = ediconnectiondraftports.ChangeKindNone
}

func withChange(out *ediconnectiondraftports.DraftValue) {
	switch out.Current {
	case "":
		out.Change = ediconnectiondraftports.ChangeKindAdd
	case out.Value:
		out.Change = ediconnectiondraftports.ChangeKindSame
	default:
		out.Change = ediconnectiondraftports.ChangeKindChange
	}
}

func distinctValues(candidates []*candidate) []string {
	values := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if !slices.Contains(values, c.value) {
			values = append(values, c.value)
		}
	}
	return values
}

func strongest(candidates []*candidate) *candidate {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if rank(c) > rank(best) {
			best = c
		}
	}
	return best
}

func rank(c *candidate) float64 {
	score := c.confidence
	if c.status == ediconnectiondraftports.ValueStatusAccepted {
		score += 2
	}
	if c.source.Kind == ediconnectiondraftports.SourceKindDocument {
		score += 0.5
	}
	return score
}

func quoteValues(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}

func joinNotes(notes ...string) string {
	kept := make([]string, 0, len(notes))
	for _, note := range notes {
		if note = strings.TrimSpace(note); note != "" {
			kept = append(kept, note)
		}
	}
	return strings.Join(kept, ". ")
}
