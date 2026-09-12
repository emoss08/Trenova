package helpers

import (
	"fmt"

	"github.com/emoss08/trenova/shared/i18n"
)

const ProblemJSONContentType = "application/problem+json"

type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`

	Errors     []ValidationError `json:"errors,omitempty"`
	TraceID    string            `json:"traceId,omitempty"`
	UsageStats any               `json:"usageStats,omitempty"`
	Params     map[string]string `json:"params,omitempty"`
}

type ValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Code     string `json:"code,omitempty"`
	Location string `json:"location,omitempty"`
	Args     []any  `json:"-"`
	Details  string `json:"-"`
}

type Message struct {
	Text    string
	Args    []any
	Details string
}

func (m Message) Localize(locale i18n.Locale) string {
	text := i18n.Translate(locale, m.Text, m.Args...)
	if m.Details == "" {
		return text
	}
	return fmt.Sprintf("%s: %s", text, m.Details)
}

func LocalizeErrors(locale i18n.Locale, errs []ValidationError) []ValidationError {
	if len(errs) == 0 {
		return errs
	}

	localized := make([]ValidationError, len(errs))
	for i, e := range errs {
		localized[i] = e
		localized[i].Message = Message{Text: e.Message, Args: e.Args, Details: e.Details}.
			Localize(locale)
		localized[i].Args = nil
	}

	return localized
}

type ProblemBuilder struct {
	baseURI     string
	problemType ProblemType
	detail      Message
	instance    string
	traceID     string
	errors      []ValidationError
	usageStats  any
	params      map[string]string
	locale      i18n.Locale
}

func NewProblemBuilder(baseURI string) *ProblemBuilder {
	return &ProblemBuilder{
		baseURI:     baseURI,
		problemType: ProblemTypeInternal,
	}
}

func (b *ProblemBuilder) WithType(t ProblemType) *ProblemBuilder {
	b.problemType = t
	return b
}

func (b *ProblemBuilder) WithDetail(d Message) *ProblemBuilder {
	b.detail = d
	return b
}

func (b *ProblemBuilder) WithInstance(path, requestID string) *ProblemBuilder {
	if requestID != "" {
		b.instance = fmt.Sprintf("%s#%s", path, requestID)
	} else {
		b.instance = path
	}
	return b
}

func (b *ProblemBuilder) WithTraceID(id string) *ProblemBuilder {
	b.traceID = id
	return b
}

func (b *ProblemBuilder) WithErrors(errors []ValidationError) *ProblemBuilder {
	b.errors = errors
	return b
}

func (b *ProblemBuilder) WithUsageStats(stats any) *ProblemBuilder {
	b.usageStats = stats
	return b
}

func (b *ProblemBuilder) WithParams(params map[string]string) *ProblemBuilder {
	b.params = params
	return b
}

func (b *ProblemBuilder) WithLocale(locale i18n.Locale) *ProblemBuilder {
	b.locale = locale
	return b
}

func (b *ProblemBuilder) translatedErrors() []ValidationError {
	return LocalizeErrors(b.locale, b.errors)
}

func (b *ProblemBuilder) Build() *ProblemDetail {
	info := b.problemType.Info()

	typeURI := string(ProblemTypeBlank)
	if b.problemType != "" && b.problemType != ProblemTypeBlank {
		typeURI = b.baseURI + string(b.problemType)
	}

	return &ProblemDetail{
		Type:       typeURI,
		Title:      i18n.Translate(b.locale, info.Title),
		Status:     info.StatusCode,
		Detail:     b.detail.Localize(b.locale),
		Instance:   b.instance,
		TraceID:    b.traceID,
		Errors:     b.translatedErrors(),
		UsageStats: b.usageStats,
		Params:     b.params,
	}
}
