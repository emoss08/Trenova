package ediconnectiondraft

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
)

type candidate struct {
	value      string
	confidence float64
	source     ediports.DraftSource
	status     ediports.ValueStatus
	problem    string
}

func (c *candidate) public() ediports.DraftCandidate {
	return ediports.DraftCandidate{
		Value:      c.value,
		Confidence: c.confidence,
		Source:     c.source,
		Problem:    c.problem,
	}
}

type evidence struct {
	byField     map[string][]*candidate
	resolutions map[string]ediports.DraftResolutionInput
}

func newEvidence() *evidence {
	return &evidence{
		byField:     make(map[string][]*candidate),
		resolutions: make(map[string]ediports.DraftResolutionInput),
	}
}

func (e *evidence) add(field string, c *candidate) {
	e.byField[field] = append(e.byField[field], c)
}

func readValue(pages *pageIndex, spec fieldSpec, input ediports.DraftValueInput) *candidate {
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
		c.status = ediports.ValueStatusUnverified
		c.problem = check.problem
	case err != nil:
		c.status = ediports.ValueStatusInvalid
		c.problem = fmt.Sprintf("%s %s", spec.label, err.Error())
	case !valueInQuote(spec, normalized, input.Citation.Quote) &&
		!valueInQuote(spec, strings.TrimSpace(raw), input.Citation.Quote):
		c.value = normalized
		c.status = ediports.ValueStatusUnverified
		c.problem = "The quote does not contain this value"
	case input.Confidence < confidenceFloor:
		c.value = normalized
		c.status = ediports.ValueStatusHeld
		c.problem = fmt.Sprintf(
			"Confidence %.2f is below %.2f",
			input.Confidence,
			confidenceFloor,
		)
	default:
		c.value = normalized
		c.status = ediports.ValueStatusAccepted
	}
	if c.status == ediports.ValueStatusInvalid || c.status == ediports.ValueStatusUnverified {
		c.value = redact(c.value)
	}

	return c
}

type fallback struct {
	value  string
	source ediports.DraftSource
	status ediports.ValueStatus
}

func resolveField(
	spec fieldSpec,
	e *evidence,
	current string,
	alternative *fallback,
	required bool,
) ediports.DraftValue {
	out := ediports.DraftValue{
		Field:    spec.key,
		Label:    spec.label,
		Current:  current,
		Required: required,
	}
	candidates := e.byField[spec.key]
	for _, c := range candidates {
		out.Candidates = append(out.Candidates, c.public())
	}

	if resolution, ok := e.resolutions[spec.key]; ok {
		return applyResolution(spec, out, candidates, resolution)
	}

	verified := make([]*candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.status == ediports.ValueStatusAccepted || c.status == ediports.ValueStatusHeld {
			verified = append(verified, c)
		}
	}
	values := distinctValues(verified)

	switch {
	case len(values) > 1:
		out.Status = ediports.ValueStatusHeld
		out.Reason = fmt.Sprintf(
			"The document gives %d different values: %s",
			len(values),
			quoteValues(values),
		)
		return keepCurrent(out)
	case len(values) == 1:
		best := strongest(verified)
		if best.status != ediports.ValueStatusAccepted {
			out.Status = ediports.ValueStatusHeld
			out.Confidence = best.confidence
			out.Reason = best.problem
			return keepCurrent(out)
		}
		out.Value = best.value
		out.Status = ediports.ValueStatusAccepted
		out.Confidence = best.confidence
		out.Source = best.source
		if len(candidates) == 1 {
			out.Candidates = nil
		}
		return withChange(out)
	case len(candidates) > 0:
		first := candidates[0]
		out.Status = first.status
		out.Reason = first.problem
		return keepCurrent(out)
	}

	out.Candidates = nil
	switch {
	case current != "":
		out.Value = current
		out.Status = ediports.ValueStatusKept
		out.Source = ediports.DraftSource{
			Kind: ediports.SourceKindConfiguration,
			Note: "The partner's current setting; the document does not mention it",
		}
		out.Change = ediports.ChangeKindSame
	case alternative != nil && alternative.value != "":
		out.Value = alternative.value
		out.Status = alternative.status
		out.Source = alternative.source
		out.Change = ediports.ChangeKindAdd
	case required:
		out.Status = ediports.ValueStatusMissing
		out.Reason = "The document does not give it"
		out.Change = ediports.ChangeKindNone
	default:
		out.Status = ediports.ValueStatusMissing
		out.Change = ediports.ChangeKindNone
	}
	return out
}

func applyResolution(
	spec fieldSpec,
	out ediports.DraftValue,
	candidates []*candidate,
	resolution ediports.DraftResolutionInput,
) ediports.DraftValue {
	normalized, err := spec.normalize(resolution.Value)
	if err != nil {
		out.Status = ediports.ValueStatusInvalid
		out.Reason = fmt.Sprintf("The chosen value is not valid: %s %s", spec.label, err)
		return keepCurrent(out)
	}

	switch resolution.Source {
	case ediports.ResolutionSourceCandidate:
		for _, c := range candidates {
			if c.value == normalized &&
				(c.status == ediports.ValueStatusAccepted || c.status == ediports.ValueStatusHeld) {
				out.Value = normalized
				out.Status = ediports.ValueStatusAccepted
				out.Confidence = c.confidence
				out.Source = c.source
				out.Source.Note = joinNotes(c.source.Note, "Chosen by the person")
				return withChange(out)
			}
		}
		out.Status = ediports.ValueStatusInvalid
		out.Reason = fmt.Sprintf(
			"%q is not one of the values the document gives for %s",
			normalized,
			spec.label,
		)
		return keepCurrent(out)
	case ediports.ResolutionSourcePerson:
		out.Value = normalized
		out.Status = ediports.ValueStatusAccepted
		out.Source = ediports.DraftSource{
			Kind: ediports.SourceKindPerson,
			Note: joinNotes("Given by the person in this conversation", strings.TrimSpace(resolution.Note)),
		}
		return withChange(out)
	default:
		out.Status = ediports.ValueStatusInvalid
		out.Reason = "A resolution must say whether it picks a cited value or one the person gave"
		return keepCurrent(out)
	}
}

func keepCurrent(out ediports.DraftValue) ediports.DraftValue {
	if out.Current != "" {
		out.Value = out.Current
		out.Change = ediports.ChangeKindSame
		return out
	}
	out.Value = ""
	out.Change = ediports.ChangeKindNone
	return out
}

func withChange(out ediports.DraftValue) ediports.DraftValue {
	switch {
	case out.Current == "":
		out.Change = ediports.ChangeKindAdd
	case out.Current == out.Value:
		out.Change = ediports.ChangeKindSame
	default:
		out.Change = ediports.ChangeKindChange
	}
	return out
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
	if c.status == ediports.ValueStatusAccepted {
		score += 2
	}
	if c.source.Kind == ediports.SourceKindDocument {
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
