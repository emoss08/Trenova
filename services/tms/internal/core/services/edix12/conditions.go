package edix12

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edistarlark"
	"github.com/emoss08/trenova/shared/maputils"
	"go.starlark.net/starlark"
)

const (
	starlarkConditionPrefix = "starlark:"
	conditionSuggestedFix   = "Check the condition syntax, field path, comparison value, or Starlark include function."
)

var validConditionRoots = map[string]struct{}{
	"shipment": {},
	"repeat":   {},
	"partner":  {},
	"mapping":  {},
	"runtime":  {},
}

type conditionEvalParams struct {
	Context   context.Context
	Condition string
	Env       map[string]any
	Segment   *edi.EDITemplateSegment
	Element   *edi.TemplateElement
}

func evaluateCondition(params conditionEvalParams) (bool, *Diagnostic) {
	condition := strings.TrimSpace(params.Condition)
	if condition == "" {
		return true, nil
	}

	include, err := evaluateConditionValue(params, condition)
	if err != nil {
		diagnostic := conditionErrorDiagnostic(params, conditionDiagnosticPath(condition), err)
		return false, &diagnostic
	}
	return include, nil
}

func evaluateConditionValue(params conditionEvalParams, condition string) (bool, error) {
	if strings.HasPrefix(condition, starlarkConditionPrefix) {
		return evaluateStarlarkCondition(params, condition)
	}
	if strings.Contains(condition, "&&") || strings.Contains(condition, "||") {
		return false, fmt.Errorf("boolean operators are not supported in EDI template conditions")
	}

	operator, left, right, hasComparison, err := splitConditionComparison(condition)
	if err != nil {
		return false, err
	}
	if hasComparison {
		if err = validateConditionPath(left); err != nil {
			return false, err
		}
		target, err := parseConditionStringLiteral(right)
		if err != nil {
			return false, err
		}

		value := valueToString(maputils.Path(params.Env, left))
		if operator == "==" {
			return value == target, nil
		}
		return value != target, nil
	}

	if strings.ContainsAny(condition, "=<>") {
		return false, fmt.Errorf("unsupported condition operator")
	}

	negated := strings.HasPrefix(condition, "!")
	path := condition
	if negated {
		path = strings.TrimSpace(strings.TrimPrefix(condition, "!"))
	}
	if err = validateConditionPath(path); err != nil {
		return false, err
	}

	truthy := isTruthyTransformValue(maputils.Path(params.Env, path))
	if negated {
		return !truthy, nil
	}
	return truthy, nil
}

func splitConditionComparison(condition string) (string, string, string, bool, error) {
	eqIndex := strings.Index(condition, "==")
	neIndex := strings.Index(condition, "!=")
	if eqIndex >= 0 && neIndex >= 0 {
		return "", "", "", false, fmt.Errorf("condition must contain only one comparison operator")
	}
	if eqIndex >= 0 {
		return splitConditionOperator(condition, "==", eqIndex)
	}
	if neIndex >= 0 {
		return splitConditionOperator(condition, "!=", neIndex)
	}
	return "", "", "", false, nil
}

func splitConditionOperator(
	condition string,
	operator string,
	index int,
) (string, string, string, bool, error) {
	left := strings.TrimSpace(condition[:index])
	right := strings.TrimSpace(condition[index+len(operator):])
	if left == "" || right == "" {
		return "", "", "", false, fmt.Errorf("condition comparison is incomplete")
	}
	return operator, left, right, true, nil
}

func validateConditionPath(path string) error {
	if path == "" {
		return fmt.Errorf("condition path is required")
	}

	parts := strings.Split(path, ".")
	root := parts[0]
	if _, ok := validConditionRoots[root]; !ok {
		return fmt.Errorf("invalid condition root %q", root)
	}

	for _, part := range parts {
		if !isConditionPathPart(part) {
			return fmt.Errorf("malformed condition path %q", path)
		}
	}
	return nil
}

func isConditionPathPart(part string) bool {
	if part == "" {
		return false
	}
	for _, char := range part {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '_' {
			continue
		}
		return false
	}
	return true
}

func parseConditionStringLiteral(value string) (string, error) {
	if len(value) < 2 {
		return "", fmt.Errorf("condition comparison value must be a quoted string")
	}

	quote := value[0]
	if quote != '"' && quote != '\'' {
		return "", fmt.Errorf("condition comparison value must be a quoted string")
	}
	if value[len(value)-1] != quote {
		return "", fmt.Errorf("condition comparison value must be a quoted string")
	}

	var builder strings.Builder
	escaped := false
	for i := 1; i < len(value)-1; i++ {
		char := value[i]
		if escaped {
			switch char {
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			default:
				builder.WriteByte(char)
			}
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == quote {
			return "", fmt.Errorf("condition comparison value must be a single quoted string literal")
		}
		builder.WriteByte(char)
	}
	if escaped {
		return "", fmt.Errorf("condition comparison value has an incomplete escape sequence")
	}
	return builder.String(), nil
}

func evaluateStarlarkCondition(params conditionEvalParams, condition string) (bool, error) {
	starlarkCtx, err := edistarlark.BuildContext(
		envMap(params.Env, "shipment"),
		envMap(params.Env, "partner"),
		envMap(params.Env, "runtime"),
		envMap(params.Env, "mapping"),
	)
	if err != nil {
		return false, err
	}

	repeatValue := params.Env["repeat"]
	if repeatValue != nil {
		starlarkCtx["repeat"] = repeatValue
		starlarkCtx["item"] = repeatValue
	}

	result := edistarlark.Evaluate(params.Context, edistarlark.EvalRequest{
		Script:       strings.TrimSpace(strings.TrimPrefix(condition, starlarkConditionPrefix)),
		FunctionName: "include",
		Context:      starlarkCtx,
		Item:         repeatValue,
		SegmentID:    params.Segment.SegmentID,
		Path:         starlarkConditionPath(),
	})
	if len(result.Diagnostics) > 0 {
		return false, errors.New(joinStarlarkConditionMessages(result.Diagnostics))
	}
	return starlarkConditionTruthy(result.Raw)
}

func starlarkConditionTruthy(value starlark.Value) (bool, error) {
	switch typed := value.(type) {
	case nil:
		return false, nil
	case starlark.NoneType:
		return false, nil
	case starlark.Bool:
		return bool(typed), nil
	case starlark.String:
		return strings.TrimSpace(string(typed)) != "", nil
	case starlark.Int:
		return true, nil
	case starlark.Float:
		return true, nil
	default:
		return false, fmt.Errorf("Starlark include returned unsupported %s result", value.Type())
	}
}

func joinStarlarkConditionMessages(diagnostics []edistarlark.Diagnostic) string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	return strings.Join(messages, "; ")
}

func conditionDiagnosticPath(condition string) string {
	if strings.HasPrefix(condition, starlarkConditionPrefix) {
		return starlarkConditionPath()
	}
	return condition
}

func starlarkConditionPath() string {
	return "condition:starlark:include"
}

func conditionErrorDiagnostic(
	params conditionEvalParams,
	path string,
	err error,
) Diagnostic {
	position := 0
	if params.Element != nil {
		position = params.Element.Position
	}

	return Diagnostic{
		Severity:        edi.ValidationSeverityError,
		Code:            "condition_error",
		SegmentID:       params.Segment.SegmentID,
		ElementPosition: position,
		Path:            path,
		Message:         err.Error(),
		SuggestedFix:    conditionSuggestedFix,
	}
}
