package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

var (
	_ serviceports.ToolTierLimiter    = (*createReportTool)(nil)
	_ serviceports.ToolResultReporter = (*createReportTool)(nil)
	_ serviceports.ToolResultReporter = (*forkReportTool)(nil)
	_ serviceports.ToolPrivateWrite   = (*createReportTool)(nil)
)

// TierLimit lets a report saved for the person asking run without a
// decision, and holds one the whole organization would see for approval.
//
// A private report changes nothing anyone else sees: it is a saved query on
// the person's own list, which they could have built by hand, and asking them
// to approve it only put a card between them and what they asked for. A
// shared one appears on every colleague's Reports page, so it stays a
// decision. A call not driven by a person, or one whose visibility cannot be
// read, stays a decision too: "private" means private to someone.
func (t *createReportTool) TierLimit(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) agent.AutonomyTier {
	actor := params.Actor
	if actor == nil || actor.PrincipalType != serviceports.PrincipalTypeUser ||
		actor.UserID.IsNil() {
		return agent.TierActWithApproval
	}

	meta, err := readReportMetadata(params.Params)
	if err != nil {
		return agent.TierActWithApproval
	}
	if meta.given["visibility"] && meta.Visibility != report.VisibilityPrivate {
		return agent.TierActWithApproval
	}

	return agent.TierAutoExecute
}

// PrivateToCaller reports a report saved only to the person's own list, which
// is theirs to save whatever the agent's ceiling.
func (t *createReportTool) PrivateToCaller(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) bool {
	return t.TierLimit(ctx, params) == agent.TierAutoExecute
}

// ExecuteWithResult saves the report and names it by the id describe_report,
// run_report and update_report take, so the conversation that asked for it
// can use it next.
func (t *createReportTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	save, err := t.prepare(params)
	if err != nil {
		return nil, err
	}

	created, err := t.reports.CreateDefinition(ctx, save)
	if err != nil {
		return nil, err
	}

	result := &agent.ToolExecutionResult{Action: "created", Kind: "report"}
	if created != nil {
		result.Name = created.Name
		result.IDs = map[string]string{"definitionId": created.ID.String()}
	}

	return result, nil
}
