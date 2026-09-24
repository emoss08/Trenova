package agentquality

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	maxExpectedTools     = 50
	maxMentionRules      = 50
	maxMentionRuleLength = 200
)

var runtimeTools = []string{"find_tools", "ask_user", "publish_artifact", "delegate_task"}

func IsRuntimeTool(name string) bool {
	return slices.Contains(runtimeTools, name)
}

type ToolMatchMode string

const (
	ToolMatchAnyOrder = ToolMatchMode("AnyOrder")
	ToolMatchOrdered  = ToolMatchMode("Ordered")
	ToolMatchSubset   = ToolMatchMode("Subset")
)

func AllToolMatchModes() []ToolMatchMode {
	return []ToolMatchMode{ToolMatchAnyOrder, ToolMatchOrdered, ToolMatchSubset}
}

func (m ToolMatchMode) IsValid() bool {
	return slices.Contains(AllToolMatchModes(), m)
}

type ToleranceKind string

const (
	ToleranceExact      = ToleranceKind("exact")
	ToleranceCI         = ToleranceKind("ci")
	ToleranceNumeric    = ToleranceKind("numeric")
	ToleranceDateWindow = ToleranceKind("dateWindow")
	ToleranceOneOf      = ToleranceKind("oneOf")
	TolerancePresent    = ToleranceKind("present")
	ToleranceIgnore     = ToleranceKind("ignore")
	ToleranceSetEq      = ToleranceKind("setEq")
)

func AllToleranceKinds() []ToleranceKind {
	return []ToleranceKind{
		ToleranceExact,
		ToleranceCI,
		ToleranceNumeric,
		ToleranceDateWindow,
		ToleranceOneOf,
		TolerancePresent,
		ToleranceIgnore,
		ToleranceSetEq,
	}
}

func (k ToleranceKind) IsValid() bool {
	return slices.Contains(AllToleranceKinds(), k)
}

func (k ToleranceKind) NeedsExpectedValue() bool {
	switch k {
	case ToleranceExact, ToleranceCI, ToleranceNumeric, ToleranceDateWindow, ToleranceSetEq:
		return true
	default:
		return false
	}
}

type Tolerance struct {
	Kind          ToleranceKind `json:"kind"`
	Abs           *float64      `json:"abs,omitempty"`
	Rel           *float64      `json:"rel,omitempty"`
	WindowSeconds int64         `json:"windowSeconds,omitempty"`
	Values        []any         `json:"values,omitempty"`
}

func (t Tolerance) validate(arg string, expected map[string]any, multiErr *errortypes.MultiError) {
	field := "rules." + arg
	if !t.Kind.IsValid() {
		multiErr.Add(field+".kind", errortypes.ErrInvalid, "Tolerance rule is not recognised")
		return
	}

	value, hasValue := expected[arg]
	if t.Kind.NeedsExpectedValue() && (!hasValue || value == nil) {
		multiErr.Add(
			field,
			errortypes.ErrRequired,
			"This rule compares against an expected value; give the argument a value",
		)
	}

	switch t.Kind {
	case ToleranceNumeric:
		if t.Abs != nil && *t.Abs < 0 {
			multiErr.Add(
				field+".abs",
				errortypes.ErrInvalid,
				"Absolute tolerance cannot be negative",
			)
		}
		if t.Rel != nil && (*t.Rel < 0 || *t.Rel > 1) {
			multiErr.Add(
				field+".rel",
				errortypes.ErrInvalid,
				"Relative tolerance is a share between 0 and 1",
			)
		}
	case ToleranceDateWindow:
		if t.WindowSeconds <= 0 {
			multiErr.Add(
				field+".windowSeconds",
				errortypes.ErrInvalid,
				"A date window must be longer than zero seconds",
			)
		}
	case ToleranceOneOf:
		if len(t.Values) == 0 {
			multiErr.Add(
				field+".values",
				errortypes.ErrRequired,
				"List at least one accepted value",
			)
		}
	case ToleranceSetEq:
		if hasValue && value != nil {
			if _, isList := value.([]any); !isList {
				multiErr.Add(
					field,
					errortypes.ErrInvalid,
					"A set comparison needs the expected argument to be a list",
				)
			}
		}
	}
}

type ExpectedTool struct {
	Name  string               `json:"name"`
	Args  map[string]any       `json:"args,omitempty"`
	Rules map[string]Tolerance `json:"rules,omitempty"`
}

func (t ExpectedTool) validate(multiErr *errortypes.MultiError) {
	if strings.TrimSpace(t.Name) == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Name the tool the agent should call")
	}
	validateRules(t.Rules, t.Args, multiErr)
}

func (t ExpectedTool) Checked() []string {
	keys := make([]string, 0, len(t.Args)+len(t.Rules))
	for key := range t.Args {
		keys = append(keys, key)
	}
	for key := range t.Rules {
		if _, listed := t.Args[key]; !listed {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	return slices.DeleteFunc(keys, func(key string) bool {
		return t.RuleFor(key).Kind == ToleranceIgnore
	})
}

func (t ExpectedTool) RuleFor(arg string) Tolerance {
	return RuleFor(t.Rules, arg)
}

type ExpectedProposal struct {
	ToolName         string               `json:"toolName"`
	Params           map[string]any       `json:"params,omitempty"`
	Rejected         bool                 `json:"rejected"`
	Rules            map[string]Tolerance `json:"rules,omitempty"`
	SourceProposalID string               `json:"sourceProposalId,omitempty"`
}

func (p ExpectedProposal) validate(multiErr *errortypes.MultiError) {
	if strings.TrimSpace(p.ToolName) == "" {
		multiErr.Add("toolName", errortypes.ErrRequired, "Name the tool the proposal calls")
	}
	if !p.Rejected && len(p.Params) == 0 {
		multiErr.Add(
			"params",
			errortypes.ErrRequired,
			"An approved proposal needs the parameters a person approved",
		)
	}
	validateRules(p.Rules, p.Params, multiErr)
}

func (p ExpectedProposal) RuleFor(arg string) Tolerance {
	return RuleFor(p.Rules, arg)
}

type Expected struct {
	ToolMode       ToolMatchMode      `json:"toolMode"`
	Tools          []ExpectedTool     `json:"tools"`
	ForbiddenTools []string           `json:"forbiddenTools"`
	Proposals      []ExpectedProposal `json:"proposals"`
	ExpectRefusal  bool               `json:"expectRefusal"`
	MustMention    []string           `json:"mustMention"`
	MustNotMention []string           `json:"mustNotMention"`
}

func (e *Expected) Normalize() {
	if e.ToolMode == "" {
		e.ToolMode = ToolMatchAnyOrder
	}
	if e.Tools == nil {
		e.Tools = []ExpectedTool{}
	}
	if e.Proposals == nil {
		e.Proposals = []ExpectedProposal{}
	}
	e.ForbiddenTools = trimmedDistinct(e.ForbiddenTools)
	e.MustMention = trimmedDistinct(e.MustMention)
	e.MustNotMention = trimmedDistinct(e.MustNotMention)
	for i := range e.Tools {
		e.Tools[i].Name = strings.TrimSpace(e.Tools[i].Name)
	}
	for i := range e.Proposals {
		e.Proposals[i].ToolName = strings.TrimSpace(e.Proposals[i].ToolName)
	}
}

func (e *Expected) Validate(multiErr *errortypes.MultiError) {
	if !e.ToolMode.IsValid() {
		multiErr.Add("toolMode", errortypes.ErrInvalid, "Tool matching mode is not recognised")
	}
	if len(e.Tools) > maxExpectedTools {
		multiErr.Add("tools", errortypes.ErrInvalid, "A case expects at most 50 tool calls")
	}

	for i, tool := range e.Tools {
		tool.validate(multiErr.WithIndex("tools", i))
		if slices.Contains(e.ForbiddenTools, tool.Name) {
			multiErr.WithIndex("tools", i).Add(
				"name",
				errortypes.ErrInvalid,
				"A tool cannot be both expected and forbidden",
			)
		}
	}
	for i, proposal := range e.Proposals {
		proposal.validate(multiErr.WithIndex("proposals", i))
	}

	if e.ExpectRefusal && len(e.Tools) > 0 {
		multiErr.Add(
			"expectRefusal",
			errortypes.ErrInvalid,
			"A case that expects a refusal cannot also expect tool calls",
		)
	}
	if e.ExpectRefusal && e.approvedProposals() > 0 {
		multiErr.Add(
			"expectRefusal",
			errortypes.ErrInvalid,
			"A case that expects a refusal cannot also expect an approved proposal",
		)
	}

	validateMentions("mustMention", e.MustMention, multiErr)
	validateMentions("mustNotMention", e.MustNotMention, multiErr)
	for _, required := range e.MustMention {
		if slices.ContainsFunc(e.MustNotMention, func(forbidden string) bool {
			return strings.EqualFold(forbidden, required)
		}) {
			multiErr.Add(
				"mustNotMention",
				errortypes.ErrInvalid,
				"A phrase cannot be both required and forbidden",
			)
			break
		}
	}
}

func (e *Expected) IsEmpty() bool {
	return len(e.Tools) == 0 && len(e.ForbiddenTools) == 0 && len(e.Proposals) == 0 &&
		!e.ExpectRefusal && len(e.MustMention) == 0 && len(e.MustNotMention) == 0
}

func (e *Expected) approvedProposals() int {
	count := 0
	for _, proposal := range e.Proposals {
		if !proposal.Rejected {
			count++
		}
	}

	return count
}

func validateRules(
	rules map[string]Tolerance,
	expected map[string]any,
	multiErr *errortypes.MultiError,
) {
	for arg, rule := range rules {
		if strings.TrimSpace(arg) == "" {
			multiErr.Add("rules", errortypes.ErrInvalid, "A rule must name the argument it checks")
			continue
		}
		rule.validate(arg, expected, multiErr)
	}
}

func validateMentions(field string, phrases []string, multiErr *errortypes.MultiError) {
	if len(phrases) > maxMentionRules {
		multiErr.Add(field, errortypes.ErrInvalid, "At most 50 phrases")
	}
	for i, phrase := range phrases {
		if len(phrase) > maxMentionRuleLength {
			multiErr.WithIndex(field, i).Add(
				"phrase",
				errortypes.ErrInvalid,
				"A phrase is at most 200 characters",
			)
		}
	}
}

func RuleFor(rules map[string]Tolerance, arg string) Tolerance {
	if rule, ok := rules[arg]; ok {
		return rule
	}

	return Tolerance{Kind: ToleranceExact}
}

func trimmedDistinct(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || slices.Contains(out, trimmed) {
			continue
		}
		out = append(out, trimmed)
	}

	return out
}
