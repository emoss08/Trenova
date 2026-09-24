package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// reportRecordEntity is the record-link registry's name for a saved report
// definition, which opens on the report explorer.
const reportRecordEntity = "report"

var (
	_ serviceports.ToolResultReporter = (*createReportTool)(nil)
	_ serviceports.ToolResultReporter = (*forkReportTool)(nil)
)

func classifyReportVisibility(params serviceports.ToolExecuteParams) serviceports.CallPolicy {
	shared := serviceports.CallPolicy{
		Egress:  agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	}

	actor := params.Actor
	if actor == nil || actor.PrincipalType != serviceports.PrincipalTypeUser ||
		actor.UserID.IsNil() {
		return shared
	}

	meta, err := readReportMetadata(params.Params)
	if err != nil {
		return shared
	}
	if meta.given["visibility"] && meta.Visibility != report.VisibilityPrivate {
		return shared
	}

	return serviceports.CallPolicy{Egress: agent.EgressPersonal}
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

	if created == nil {
		return &agent.ToolExecutionResult{Action: "created", Kind: "report"}, nil
	}

	return reportResult(created.ID.String(), created.Name), nil
}

// reportResult names a report definition a tool created, by the id the
// report tools take and as the record it opens as.
func reportResult(id, name string) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: "created",
		Kind:   "report",
		Name:   name,
		IDs:    map[string]string{"definitionId": id},
		Record: &agent.RecordRef{EntityType: reportRecordEntity, ID: id},
	}
}
