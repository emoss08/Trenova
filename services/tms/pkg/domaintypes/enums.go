package domaintypes

import "fmt"

var ErrInvalidStatus = fmt.Errorf("invalid status")

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
)

func StatusFromString(s string) (Status, error) {
	switch s {
	case "Active":
		return StatusActive, nil
	case "Inactive":
		return StatusInactive, nil
	default:
		return "", ErrInvalidStatus
	}
}

type TimeFormat string

const (
	TimeFormat12Hour TimeFormat = "12-hour"
	TimeFormat24Hour TimeFormat = "24-hour"
)
