package toolartifacts

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/typeutils"
)

// dispatchPlanArtifact is a dispatch_plan read as the plan a dispatcher acts
// on: each load beside the driver the optimizer put on it, what that costs
// and how sure it is, the next best drivers, the loads nobody can take and
// why, and the assignments ready to make.
//
// It used to be nothing at all. Asked who should take twenty loads, the Desk
// ranked one move at a time and wrote the answer out in prose, and the person
// asked for "a side by side" three times before giving up. The card reads
// each move's coverage live, so a plan already acted on says which loads now
// have a driver.
func dispatchPlanArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	lines, _ := result["lines"].([]any)
	uncovered, _ := result["uncovered"].([]any)
	if len(lines) == 0 && len(uncovered) == 0 {
		return nil
	}

	notPlanned, _ := result["notPlanned"].([]any)
	assignments, _ := result["assignments"].([]any)
	payload := map[string]any{
		payloadDisplay:              assistantartifact.DisplayVersion,
		artifactpayload.PayloadTool: "dispatch_plan",
		"planningMode":              typeutils.StringOfTrimmed(result["planningMode"]),
		"cost":                      typeutils.StringOfTrimmed(result["cost"]),
		"lines":                     lines,
		"uncovered":                 uncovered,
		"assignments":               assignments,
		"notPlannedCount":           float64(len(notPlanned)),
	}
	fitRows(payload, "lines")
	fitRows(payload, "uncovered")

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindDispatchPlan,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(dispatchPlanTitle(len(lines) + len(uncovered))),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

func dispatchPlanTitle(loads int) string {
	if loads == 1 {
		return "Driver for 1 load"
	}

	return fmt.Sprintf("Drivers for %d loads", loads)
}
