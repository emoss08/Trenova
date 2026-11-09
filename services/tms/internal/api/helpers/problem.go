package helpers

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fieldpath"
	"github.com/emoss08/trenova/shared/i18n"
)

const ProblemJSONContentType = "application/problem+json"

type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Code     string `json:"code,omitempty"`

	Errors     []ValidationError        `json:"errors,omitempty"`
	Advisories []Advisory               `json:"advisories,omitempty"`
	TraceID    string                   `json:"traceId,omitempty"`
	UsageStats any                      `json:"usageStats,omitempty"`
	Conflict   *errortypes.EditConflict `json:"conflict,omitempty"`
	Params     map[string]string        `json:"params,omitempty"`
}

type ValidationError struct {
	Path     fieldpath.Path          `json:"path"`
	Message  string                  `json:"message"`
	Code     string                  `json:"code,omitempty"`
	Location string                  `json:"location,omitempty"`
	Range    *errortypes.SourceRange `json:"range,omitempty"`
	Args     []any                   `json:"-"`
	Details  string                  `json:"-"`
}

type Advisory struct {
	Path     fieldpath.Path      `json:"path"`
	Code     string              `json:"code"`
	Message  string              `json:"message"`
	Severity errortypes.Severity `json:"severity"`
	RuleKey  string              `json:"ruleKey,omitempty"`
	Args     []any               `json:"-"`
}

type Message struct {
	Text    string
	Args    []any
	Details string
	Parts   []Message
}

func (m Message) Localize(locale i18n.Locale) string {
	if len(m.Parts) > 0 {
		parts := make([]string, 0, len(m.Parts))
		for _, part := range m.Parts {
			parts = append(parts, part.Localize(locale))
		}
		return strings.Join(parts, " ")
	}

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

func LocalizeAdvisories(locale i18n.Locale, advisories []Advisory) []Advisory {
	if len(advisories) == 0 {
		return advisories
	}

	localized := make([]Advisory, len(advisories))
	for i, advisory := range advisories {
		localized[i] = advisory
		localized[i].Message = i18n.Translate(locale, advisory.Message, advisory.Args...)
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
	code        errortypes.ErrorCode
	errors      []ValidationError
	advisories  []Advisory
	usageStats  any
	conflict    *errortypes.EditConflict
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

func (b *ProblemBuilder) WithCode(code errortypes.ErrorCode) *ProblemBuilder {
	b.code = code
	return b
}

func (b *ProblemBuilder) WithErrors(errors []ValidationError) *ProblemBuilder {
	b.errors = errors
	return b
}

func (b *ProblemBuilder) WithAdvisories(advisories []Advisory) *ProblemBuilder {
	b.advisories = advisories
	return b
}

func (b *ProblemBuilder) WithUsageStats(stats any) *ProblemBuilder {
	b.usageStats = stats
	return b
}

func (b *ProblemBuilder) WithConflict(conflict *errortypes.EditConflict) *ProblemBuilder {
	b.conflict = conflict
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
		Code:       string(b.code),
		TraceID:    b.traceID,
		Errors:     b.translatedErrors(),
		Advisories: LocalizeAdvisories(b.locale, b.advisories),
		UsageStats: b.usageStats,
		Conflict:   b.conflict,
		Params:     b.params,
	}
}
