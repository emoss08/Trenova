package errortypes

import "fmt"

type Severity string

const (
	SeverityWarn          = Severity("Warn")
	SeverityRequireReview = Severity("RequireReview")
)

func (s Severity) String() string { return string(s) }

func (s Severity) IsValid() bool {
	switch s {
	case SeverityWarn, SeverityRequireReview:
		return true
	default:
		return false
	}
}

type Advisory struct {
	Field    string    `json:"field"`
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Severity Severity  `json:"severity"`
	RuleKey  string    `json:"ruleKey,omitempty"`
}

func (a *Advisory) Error() string {
	if a.Field == "" {
		return a.Message
	}
	return fmt.Sprintf("%s: %s", a.Field, a.Message)
}

func NewAdvisory(field string, code ErrorCode, message string, severity Severity) *Advisory {
	return &Advisory{
		Field:    field,
		Code:     code,
		Message:  message,
		Severity: severity,
	}
}

func (a *Advisory) WithRuleKey(ruleKey string) *Advisory {
	a.RuleKey = ruleKey
	return a
}

func (m *MultiError) AddAdvisory(advisory *Advisory) {
	if advisory == nil {
		return
	}

	root := m.root()

	advisoryCopy := &Advisory{
		Field:    advisory.Field,
		Code:     advisory.Code,
		Message:  advisory.Message,
		Severity: advisory.Severity,
		RuleKey:  advisory.RuleKey,
	}

	if prefix := m.getFullPrefix(); prefix != "" {
		if advisoryCopy.Field != "" {
			advisoryCopy.Field = prefix + "." + advisoryCopy.Field
		} else {
			advisoryCopy.Field = prefix
		}
	}

	root.Advisories = append(root.Advisories, advisoryCopy)
}

func (m *MultiError) Advise(
	field string,
	code ErrorCode,
	message string,
	severity Severity,
	ruleKey string,
) {
	m.AddAdvisory(&Advisory{
		Field:    field,
		Code:     code,
		Message:  message,
		Severity: severity,
		RuleKey:  ruleKey,
	})
}

func (m *MultiError) HasAdvisories() bool {
	if m == nil {
		return false
	}
	return len(m.root().Advisories) > 0
}

func (m *MultiError) RequiresReview() bool {
	if m == nil {
		return false
	}

	for _, advisory := range m.root().Advisories {
		if advisory.Severity == SeverityRequireReview {
			return true
		}
	}

	return false
}

func (m *MultiError) AllAdvisories() []*Advisory {
	if m == nil {
		return nil
	}
	return m.root().Advisories
}
