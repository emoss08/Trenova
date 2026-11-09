package errortypes

import (
	"github.com/emoss08/trenova/pkg/fieldpath"
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
	Path     fieldpath.Path `json:"path"`
	Code     ErrorCode      `json:"code"`
	Message  string         `json:"message"`
	Args     []any          `json:"-"`
	Severity Severity       `json:"severity"`
	RuleKey  string         `json:"ruleKey,omitempty"`
}

func (a *AdvisoryError) Field() string {
	return a.Path.String()
}

func (a *AdvisoryError) Error() string {
	message := i18n.Format(a.Message, a.Args...)
	if a.Path.IsZero() {
		return message
	}
	return a.Field() + ": " + message
}

func NewAdvisory(
	field string,
	code ErrorCode,
	message string,
	severity Severity,
	args ...any,
) *AdvisoryError {
	return &AdvisoryError{
		Path:     fieldpath.Parse(field),
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

	scoped := *advisory
	scoped.Path = m.prefix.Join(advisory.Path)

	top := m.top()
	top.Advisories = append(top.Advisories, &scoped)
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
		Path:     fieldpath.Parse(field),
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
	return len(m.top().Advisories) > 0
}

func (m *MultiError) RequiresReview() bool {
	if m == nil {
		return false
	}

	for _, advisory := range m.top().Advisories {
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
	return m.top().Advisories
}

func (a *AdvisoryError) LocalizedMessage() (message string, args []any) {
	return a.Message, a.Args
}
