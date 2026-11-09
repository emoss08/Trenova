package dataentrycontrol

type CaseFormat string

const (
	CaseFormatAsEntered = CaseFormat("AsEntered")
	CaseFormatUpper     = CaseFormat("Upper")
	CaseFormatLower     = CaseFormat("Lower")
	CaseFormatTitleCase = CaseFormat("TitleCase")
)

func (c CaseFormat) String() string {
	return string(c)
}

func (c CaseFormat) IsValid() bool {
	switch c {
	case CaseFormatAsEntered, CaseFormatUpper, CaseFormatLower, CaseFormatTitleCase:
		return true
	default:
		return false
	}
}
