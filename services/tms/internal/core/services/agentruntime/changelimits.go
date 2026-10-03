package agentruntime

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// businessHoursNote tells the model why a change it expected to run is
// waiting for a person instead.
const businessHoursNote = " It was not run automatically because it is outside this agent's " +
	"business hours; a person decides it instead. Tell them it is waiting for them."

// changeSize is how many records one call would change: the most ids any of
// its record parameters carries. A record parameter is one its schema marks as
// a set of records, or an array of ids by name. A call naming one record by a
// single id is one record and never near any limit, so it counts as zero.
func changeSize(schema, arguments map[string]any) int {
	properties, _ := schema[toolschema.KeyProperties].(map[string]any)
	size := 0
	for name, raw := range properties {
		property, ok := raw.(map[string]any)
		if !ok || !recordList(name, property) {
			continue
		}
		size = max(size, distinctIDs(arguments[name]))
	}

	return size
}

func recordList(name string, property map[string]any) bool {
	if toolschema.SubsetResource(property) != "" {
		return true
	}
	if kind, _ := property[toolschema.KeyType].(string); kind != toolschema.TypeArray {
		return false
	}
	lower := strings.ToLower(name)

	return strings.HasSuffix(lower, "ids") || strings.HasSuffix(lower, "_ids")
}

func distinctIDs(value any) int {
	seen := make(map[string]struct{})
	switch typed := value.(type) {
	case []string:
		for _, id := range typed {
			seen[strings.TrimSpace(id)] = struct{}{}
		}
	case []any:
		for _, item := range typed {
			if id, ok := item.(string); ok {
				seen[strings.TrimSpace(id)] = struct{}{}
			}
		}
	}
	delete(seen, "")

	return len(seen)
}

// oversizedChange refuses a write that would touch more records than the
// agent's largest single change. It is refused rather than cut down, so
// nothing is silently left out, and the model is told to split it: each part
// is then proposed, and approved, on its own.
func oversizedChange(
	req *serviceports.RunRequest,
	toolName string,
	schema map[string]any,
	arguments map[string]any,
) (toolOutcome, bool) {
	limit := req.Definition.ChangeLimit()
	size := changeSize(schema, arguments)
	if size <= limit {
		return toolOutcome{}, false
	}

	reason := fmt.Sprintf("it would change %d records; the most this agent may change at once is %d",
		size, limit)

	return refusedOutcome(aitrace.OutcomeOverBudget, reason,
		"Tool %q was not proposed or run: it would change %d records, and this agent may "+
			"change at most %d in one change. Split it into calls of at most %d records each; "+
			"each part is approved separately.",
		toolName, size, limit, limit), true
}

// heldForBusinessHours lowers a write that would run on its own to one a
// person approves, when the agent may only change things in business hours
// and it is outside them. A write already waiting for a person is left as it
// is: a person deciding outside the window is their call to make.
func (s *Service) heldForBusinessHours(
	ctx context.Context,
	req *serviceports.RunRequest,
	tier agent.AutonomyTier,
) (agent.AutonomyTier, bool) {
	if tier != agent.TierAutoExecute || !req.Definition.BusinessHoursOnly || s.budgets == nil {
		return tier, false
	}
	if s.budgets.WithinBusinessHours(ctx, req.Definition) {
		return tier, false
	}

	return agent.TierActWithApproval, true
}
