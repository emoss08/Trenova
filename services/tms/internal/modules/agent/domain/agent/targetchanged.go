package agent

import (
	"errors"
	"strings"
)

var (
	ErrTargetChanged     = errors.New("the record changed since this was proposed")
	ErrPlanTargetChanged = errors.New(
		"nothing ran, because a record the plan changes moved on after it was proposed",
	)
)

func TargetChanged(executionError string) bool {
	return strings.HasPrefix(strings.TrimSpace(executionError), ErrTargetChanged.Error())
}

func PlanTargetChanged(plan *AgentPlan) bool {
	return plan != nil && plan.Status == PlanStatusFailed &&
		strings.HasPrefix(strings.TrimSpace(plan.FailureError), ErrPlanTargetChanged.Error())
}
