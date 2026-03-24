package formulatemplate

import "errors"

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
	StatusDraft    = Status("Draft")
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
	default:
		return "", errors.New("invalid status")
	}
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
