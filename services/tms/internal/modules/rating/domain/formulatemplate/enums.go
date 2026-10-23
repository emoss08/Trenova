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

var allowedTransitions = map[Status][]Status{
	StatusDraft:    {StatusInReview},
	StatusInReview: {StatusActive, StatusInactive, StatusDraft},
	StatusActive:   {StatusInactive, StatusDraft},
	StatusInactive: {StatusInReview},
}

func CanTransition(from, to Status) bool {
	return from == to || slices.Contains(allowedTransitions[from], to)
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
