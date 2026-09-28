package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// validateExempt lists the action tools that do not check a call before it
// is filed, each with the reason: the service behind it has no plan or
// preview function to run, so there is nothing to ask short of the write.
// A tool whose service gains one comes off this list.
var validateExempt = map[string]string{
	"update_shipment": "shipmentservice has no update plan; the tool patches the loaded " +
		"record itself, and the service's validation runs only in Update",
	"dismiss_insight": "insightservice dismisses in the repository; the preview applies the " +
		"domain transition, which is not the code the write runs",
	"update_tractor_status": "tractorservice has no status plan; the preview reads the units " +
		"through the repository",
	"update_trailer_status": "trailerservice has no status plan; the preview reads the units " +
		"through the repository",
	"email_customer": "the send has no plan function; what refuses it is the mail provider " +
		"at send time",
	"request_missing_docs": "the send has no plan function; what refuses it is the mail " +
		"provider at send time",
	"fork_report": "reporting has no fork plan that can refuse; NewCannedFork always builds " +
		"a copy and the tool refuses an unknown report key while reading its arguments",
}

// Every action tool checks a call before it is filed, simulated or run, so
// a call the write would refuse is refused to the model while it can still
// fix it, rather than after a person has approved it.
func TestEveryActionToolValidatesBeforeFiling(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool, len(validateExempt))
	for _, tool := range buildRegistered(t).Actions {
		if tool.Policy().Kind != agent.ToolKindAction {
			continue
		}
		_, validates := tool.(serviceports.ToolValidator)
		reason, exempt := validateExempt[tool.Name()]
		seen[tool.Name()] = true

		switch {
		case validates && exempt:
			t.Errorf("%s validates now; remove it from validateExempt (%s)",
				tool.Name(), reason)
		case !validates && !exempt:
			t.Errorf("%s is an action tool without Validate; a call it would refuse is "+
				"filed for a person to approve first", tool.Name())
		}
	}

	for name := range validateExempt {
		if !seen[name] {
			t.Errorf("validateExempt names %s, which is not a registered action tool", name)
		}
	}
}
