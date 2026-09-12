package errortypes

import (
	"fmt"

	"github.com/emoss08/trenova/shared/i18n"
)

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

type AdvisoryError struct {
	Field    string    `json:"field"`
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Args     []any     `json:"-"`
	Severity Severity  `json:"severity"`
	RuleKey  string    `json:"ruleKey,omitempty"`
}

func (a *AdvisoryError) Error() string {
	message := i18n.Format(a.Message, a.Args...)
	if a.Field == "" {
		return message
	}
	return fmt.Sprintf("%s: %s", a.Field, message)
}

func NewAdvisory(
	field string,
	code ErrorCode,
	message string,
	severity Severity,
	args ...any,
) *AdvisoryError {
	return &AdvisoryError{
		Field:    field,
		Code:     code,
		Message:  message,
		Args:     args,
		Severity: severity,
	}
}

func (a *AdvisoryError) WithRuleKey(ruleKey string) *AdvisoryError {
	a.RuleKey = ruleKey
	return a
}

func (m *MultiError) AddAdvisory(advisory *AdvisoryError) {
	if advisory == nil {
		return
	}

	root := m.root()

	advisoryCopy := &AdvisoryError{
		Field:    advisory.Field,
		Code:     advisory.Code,
		Message:  advisory.Message,
		Args:     advisory.Args,
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
	args ...any,
) {
	m.AddAdvisory(&AdvisoryError{
		Field:    field,
		Code:     code,
		Message:  message,
		Args:     args,
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

func (m *MultiError) AllAdvisories() []*AdvisoryError {
	if m == nil {
		return nil
	}
	return m.root().Advisories
}

func (a *AdvisoryError) LocalizedMessage() (message string, args []any) {
	return a.Message, a.Args
}
