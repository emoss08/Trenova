package agentscoring

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const numericEpsilon = 1e-9

func Match(rule agentquality.Tolerance, expected, actual any, present bool) bool {
	switch rule.Kind {
	case agentquality.ToleranceIgnore:
		return true
	case agentquality.TolerancePresent:
		return present && !isEmpty(actual)
	}
	if !present {
		return false
	}

	switch rule.Kind {
	case agentquality.ToleranceCI:
		return strings.EqualFold(
			strings.TrimSpace(fmt.Sprint(expected)),
			strings.TrimSpace(fmt.Sprint(actual)),
		)
	case agentquality.ToleranceNumeric:
		return matchNumeric(rule, expected, actual)
	case agentquality.ToleranceDateWindow:
		return matchDateWindow(rule, expected, actual)
	case agentquality.ToleranceOneOf:
		for _, accepted := range rule.Values {
			if canonical(accepted) == canonical(actual) {
				return true
			}
		}

		return false
	case agentquality.ToleranceSetEq:
		return matchSet(expected, actual)
	default:
		return canonical(expected) == canonical(actual)
	}
}

func matchNumeric(rule agentquality.Tolerance, expected, actual any) bool {
	want, ok := floatutils.ParseNumber(expected)
	if !ok {
		return false
	}
	got, ok := floatutils.ParseNumber(actual)
	if !ok {
		return false
	}

	diff := math.Abs(want - got)
	if rule.Abs == nil && rule.Rel == nil {
		return diff <= numericEpsilon
	}
	if rule.Abs != nil && diff <= *rule.Abs+numericEpsilon {
		return true
	}
	if rule.Rel != nil {
		scale := math.Max(math.Abs(want), math.Abs(got))
		return diff <= *rule.Rel*scale+numericEpsilon
	}

	return false
}

func matchDateWindow(rule agentquality.Tolerance, expected, actual any) bool {
	want, ok := timeutils.ParseInstant(expected)
	if !ok {
		return false
	}
	got, ok := timeutils.ParseInstant(actual)
	if !ok {
		return false
	}

	diff := want - got
	if diff < 0 {
		diff = -diff
	}

	return diff <= rule.WindowSeconds
}

func matchSet(expected, actual any) bool {
	want, ok := expected.([]any)
	if !ok {
		return false
	}
	got := asList(actual)
	if got == nil {
		return false
	}

	return sameSet(want, got)
}

func asList(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}

		return out
	default:
		return nil
	}
}

func sameSet(left, right []any) bool {
	leftSet := make(map[string]struct{}, len(left))
	for _, item := range left {
		leftSet[canonical(item)] = struct{}{}
	}
	rightSet := make(map[string]struct{}, len(right))
	for _, item := range right {
		key := canonical(item)
		if _, known := leftSet[key]; !known {
			return false
		}
		rightSet[key] = struct{}{}
	}

	return len(leftSet) == len(rightSet)
}

func canonical(value any) string {
	return fmt.Sprint(agent.NormalizeValue(value))
}

func isEmpty(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

func argumentsMatch(
	params map[string]any,
	rules map[string]agentquality.Tolerance,
	actual map[string]any,
) (bool, []string) {
	keys := make(map[string]struct{}, len(params)+len(rules))
	for key := range params {
		keys[key] = struct{}{}
	}
	for key := range rules {
		keys[key] = struct{}{}
	}

	failed := make([]string, 0, len(keys))
	for key := range keys {
		value, present := actual[key]
		if !Match(agentquality.RuleFor(rules, key), params[key], value, present) {
			failed = append(failed, key)
		}
	}

	slices.Sort(failed)

	return len(failed) == 0, failed
}
