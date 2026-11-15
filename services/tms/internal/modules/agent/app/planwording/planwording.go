// Package planwording is how an agent or the assistant is told that the
// organization's plan, not a person's access, kept something from it: the
// one place the words are built, so a tool refused at dispatch, a tool the
// turn was never offered and a figure left off a board say it the same way.
package planwording

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

// Reason is what a refused call records as its cause, for its trace and the
// analytics that sort refusals.
func Reason(denial *permissionports.PlanDenial) string {
	return fmt.Sprintf("%s is not on the organization's plan", denial.Feature)
}

// Restriction is the plan's own sentence: "This feature is part of the
// Accounting & Cash add-on, which is not on your plan".
func Restriction(denial *permissionports.PlanDenial) string {
	return denial.Restriction().Error()
}

// Advice tells the model what to do instead of trying another way. An agent
// working alone reports the missing add-on; one working for a person tells
// them, and does the rest of what they asked.
func Advice(unattended bool) string {
	if unattended {
		return "Do not try it another way. Finish what the run can do without it and say in " +
			"your report that this step needs an add-on the organization's plan does not include."
	}

	return "Do not try it another way or propose it instead. Tell the person plainly that " +
		"their organization's plan does not include it and that an administrator can add the " +
		"add-on. Do everything else they asked that does not need it."
}

// Refusal is the whole answer for something the plan withholds: what it is,
// the plan's sentence and the advice. subject names it ("Tool \"x\"", "The
// marginPercent metric").
func Refusal(subject string, denial *permissionports.PlanDenial, unattended bool) string {
	return fmt.Sprintf(
		"%s is not available: %s. %s",
		subject,
		Restriction(denial),
		Advice(unattended),
	)
}
