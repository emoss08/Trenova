package ediconnectionfield

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/x12field"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
)

type Candidate struct {
	Value      string
	confidence float64
	source     ediconnectiondraftports.DraftSource
	Status     ediconnectiondraftports.ValueStatus
	problem    string
}

func (c *Candidate) public() ediconnectiondraftports.DraftCandidate {
	return ediconnectiondraftports.DraftCandidate{
		Value:      c.Value,
		Confidence: c.confidence,
		Source:     c.source,
		Problem:    c.problem,
	}
}

type Evidence struct {
	ByField     map[string][]*Candidate
	Resolutions map[string]ediconnectiondraftports.DraftResolutionInput
}

func NewEvidence() *Evidence {
	return &Evidence{
		ByField:     make(map[string][]*Candidate),
		Resolutions: make(map[string]ediconnectiondraftports.DraftResolutionInput),
	}
}

func (e *Evidence) Add(field string, c *Candidate) {
	e.ByField[field] = append(e.ByField[field], c)
}

func ReadValue(
	pages *Pages,
	spec *Spec,
	input *ediconnectiondraftports.DraftValueInput,
) *Candidate {
	raw := input.Value
	c := &Candidate{Value: strings.TrimSpace(raw), confidence: input.Confidence}

	check := pages.Locate(input.Citation)
	page := check.Page
	if page == 0 {
		page = input.Citation.Page
	}
	c.source = DocumentSource(page, input.Citation, check.Problem)

	normalized, err := spec.Normalize(raw)
	switch {
	case !check.OK:
		c.Status = ediconnectiondraftports.ValueStatusUnverified
		c.problem = check.Problem
	case err != nil:
		c.Status = ediconnectiondraftports.ValueStatusInvalid
		c.problem = fmt.Sprintf("%s %s", spec.Label, err.Error())
	case !valueInQuote(spec, normalized, input.Citation.Quote) &&
		!valueInQuote(spec, strings.TrimSpace(raw), input.Citation.Quote):
		c.Value = normalized
		c.Status = ediconnectiondraftports.ValueStatusUnverified
		c.problem = "The quote does not contain this value"
	case input.Confidence < ConfidenceFloor:
		c.Value = normalized
		c.Status = ediconnectiondraftports.ValueStatusHeld
		c.problem = fmt.Sprintf(
			"Confidence %.2f is below %.2f",
			input.Confidence,
			ConfidenceFloor,
		)
	default:
		c.Value = normalized
		c.Status = ediconnectiondraftports.ValueStatusAccepted
	}
	if c.Status == ediconnectiondraftports.ValueStatusInvalid ||
		c.Status == ediconnectiondraftports.ValueStatusUnverified {
		c.Value = x12field.RedactSecrets(c.Value)
	}

	return c
}

type Fallback struct {
	Value  string
	Source ediconnectiondraftports.DraftSource
	Status ediconnectiondraftports.ValueStatus
}

func Resolve(
	spec *Spec,
	e *Evidence,
	current string,
	alternative *Fallback,
	required bool,
) ediconnectiondraftports.DraftValue {
	out := &ediconnectiondraftports.DraftValue{
		Field:    spec.Key,
		Label:    spec.Label,
		Current:  current,
		Required: required,
	}
	candidates := e.ByField[spec.Key]
	for _, c := range candidates {
		out.Candidates = append(out.Candidates, c.public())
	}

	switch resolution, resolved := e.Resolutions[spec.Key]; {
	case resolved:
		applyResolution(spec, out, candidates, &resolution)
	case len(candidates) > 0:
		resolveCandidates(out, candidates)
	default:
		resolveAbsent(out, alternative, required)
	}
	return *out
}

func resolveCandidates(out *ediconnectiondraftports.DraftValue, candidates []*Candidate) {
	verified := make([]*Candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.Status == ediconnectiondraftports.ValueStatusAccepted ||
			c.Status == ediconnectiondraftports.ValueStatusHeld {
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
		if best.Status != ediconnectiondraftports.ValueStatusAccepted && out.Current != "" &&
			best.Value == out.Current {
			out.Value = out.Current
			out.Status = ediconnectiondraftports.ValueStatusKept
			out.Confidence = best.confidence
			out.Source = best.source
			out.Reason = "The guide's value is the one already saved, so nothing changes"
			out.Change = ediconnectiondraftports.ChangeKindSame
			return
		}
		if best.Status != ediconnectiondraftports.ValueStatusAccepted {
			out.Status = ediconnectiondraftports.ValueStatusHeld
			out.Confidence = best.confidence
			out.Reason = best.problem
			keepCurrent(out)
			return
		}
		out.Value = best.Value
		out.Status = ediconnectiondraftports.ValueStatusAccepted
		out.Confidence = best.confidence
		out.Source = best.source
		if len(candidates) == 1 {
			out.Candidates = nil
		}
		WithChange(out)
	default:
		first := candidates[0]
		out.Status = first.Status
		out.Reason = first.problem
		keepCurrent(out)
	}
}

func resolveAbsent(out *ediconnectiondraftports.DraftValue, alternative *Fallback, required bool) {
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
	case alternative != nil && alternative.Value != "":
		out.Value = alternative.Value
		out.Status = alternative.Status
		out.Source = alternative.Source
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
	spec *Spec,
	out *ediconnectiondraftports.DraftValue,
	candidates []*Candidate,
	resolution *ediconnectiondraftports.DraftResolutionInput,
) {
	normalized, err := spec.Normalize(resolution.Value)
	if err != nil {
		out.Status = ediconnectiondraftports.ValueStatusInvalid
		out.Reason = fmt.Sprintf("The chosen value is not valid: %s %s", spec.Label, err)
		keepCurrent(out)
		return
	}

	switch resolution.Source {
	case ediconnectiondraftports.ResolutionSourceCandidate:
		for _, c := range candidates {
			if c.Value != normalized ||
				(c.Status != ediconnectiondraftports.ValueStatusAccepted && c.Status != ediconnectiondraftports.ValueStatusHeld) {
				continue
			}
			out.Value = normalized
			out.Status = ediconnectiondraftports.ValueStatusAccepted
			out.Confidence = c.confidence
			out.Source = c.source
			out.Source.Note = JoinNotes(c.source.Note, "Chosen by the person")
			WithChange(out)
			return
		}
		out.Status = ediconnectiondraftports.ValueStatusInvalid
		out.Reason = fmt.Sprintf(
			"%q is not one of the values the document gives for %s",
			normalized,
			spec.Label,
		)
		keepCurrent(out)
	case ediconnectiondraftports.ResolutionSourcePerson:
		out.Value = normalized
		out.Status = ediconnectiondraftports.ValueStatusAccepted
		out.Source = ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindPerson,
			Note: JoinNotes(
				"Given by the person in this conversation",
				strings.TrimSpace(resolution.Note),
			),
		}
		WithChange(out)
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

func WithChange(out *ediconnectiondraftports.DraftValue) {
	switch out.Current {
	case "":
		out.Change = ediconnectiondraftports.ChangeKindAdd
	case out.Value:
		out.Change = ediconnectiondraftports.ChangeKindSame
	default:
		out.Change = ediconnectiondraftports.ChangeKindChange
	}
}

func distinctValues(candidates []*Candidate) []string {
	values := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if !slices.Contains(values, c.Value) {
			values = append(values, c.Value)
		}
	}
	return values
}

func strongest(candidates []*Candidate) *Candidate {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if rank(c) > rank(best) {
			best = c
		}
	}
	return best
}

func rank(c *Candidate) float64 {
	score := c.confidence
	if c.Status == ediconnectiondraftports.ValueStatusAccepted {
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

func JoinNotes(notes ...string) string {
	kept := make([]string, 0, len(notes))
	for _, note := range notes {
		if note = strings.TrimSpace(note); note != "" {
			kept = append(kept, note)
		}
	}
	return strings.Join(kept, ". ")
}

type IdentityHints struct {
	Code         string
	Name         string
	ISAQualifier string
	ISAID        string
	AS2ID        string
}

func (e *Evidence) Hint(field string) string {
	spec, ok := Catalog[field]
	if !ok {
		return ""
	}
	settled := Resolve(spec, e, "", nil, false)
	if settled.Status != ediconnectiondraftports.ValueStatusAccepted {
		return ""
	}
	return settled.Value
}

func (e *Evidence) Hints() *IdentityHints {
	return &IdentityHints{
		Code:         e.Hint(PartnerCode),
		Name:         e.Hint(PartnerName),
		ISAQualifier: e.Hint(PartnerQualifier),
		ISAID:        e.Hint(PartnerID),
		AS2ID:        e.Hint("as2.partnerAs2Id"),
	}
}
