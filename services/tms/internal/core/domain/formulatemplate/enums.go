package formulatemplate

import (
	"errors"
	"slices"
)

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
	StatusDraft    = Status("Draft")
	StatusInReview = Status("InReview")
)

func (s Status) String() string {
	return string(s)
}

func StatusFromString(s string) (Status, error) {
	switch s {
	case "Active":
		return StatusActive, nil
	case "Inactive":
		return StatusInactive, nil
	case "Draft":
		return StatusDraft, nil
	case "InReview":
		return StatusInReview, nil
	default:
		return "", errors.New("invalid status")
	}
}

// allowedTransitions is the review cycle. Active is reachable only from
// InReview: an archived template goes back through review rather than
// straight to Active, because its content may reference tables that have
// since changed and nobody has looked at it in a while.
var allowedTransitions = map[Status][]Status{
	StatusDraft:    {StatusInReview},
	StatusInReview: {StatusActive, StatusInactive, StatusDraft},
	StatusActive:   {StatusInactive, StatusDraft},
	StatusInactive: {StatusInReview},
}

func CanTransition(from, to Status) bool {
	return from == to || slices.Contains(allowedTransitions[from], to)
}

type TemplateType string

const (
	TemplateTypeFreightCharge     = TemplateType("FreightCharge")
	TemplateTypeAccessorialCharge = TemplateType("AccessorialCharge")
)

func (tt TemplateType) String() string {
	return string(tt)
}

func TemplateTypeFromString(s string) (TemplateType, error) {
	switch s {
	case "FreightCharge":
		return TemplateTypeFreightCharge, nil
	case "AccessorialCharge":
		return TemplateTypeAccessorialCharge, nil
	default:
		return "", errors.New("invalid template type")
	}
}

func (v Status) IsValid() bool {
	switch v {
	case StatusActive,
		StatusInactive,
		StatusDraft,
		StatusInReview:
		return true
	default:
		return false
	}
}

func (v TemplateType) IsValid() bool {
	switch v {
	case TemplateTypeFreightCharge,
		TemplateTypeAccessorialCharge:
		return true
	default:
		return false
	}
}
