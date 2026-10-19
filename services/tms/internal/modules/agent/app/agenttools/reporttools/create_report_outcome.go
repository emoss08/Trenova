package reporttools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

// reportRecordEntity is the record-link registry's name for a saved report
// definition, which opens on the report explorer.
const reportRecordEntity = "report"

func classifyReportVisibility(params agentports.ToolExecuteParams) agentports.CallPolicy {
	shared := agentports.CallPolicy{
		Egress:  agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	}

	actor := params.Actor
	if actor == nil || actor.PrincipalType != permissionports.PrincipalTypeUser ||
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

	return agentports.CallPolicy{Egress: agent.EgressPersonal}
}

// ExecuteWithResult saves the report and names it by the id report_get,
// report_run and report_update take, so the conversation that asked for it
// can use it next.
func (t *CreateReportTool) ExecuteWithResult(
	ctx context.Context,
	params agentports.ToolExecuteParams,
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
		return &agent.ToolExecutionResult{
			Action: common.ResultCreated,
			Kind:   reportRecordEntity,
		}, nil
	}

	return reportResult(created.ID.String(), created.Name), nil
}

// reportResult names a report definition a tool created, by the id the
// report tools take and as the record it opens as.
func reportResult(id, name string) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: common.ResultCreated,
		Kind:   reportRecordEntity,
		Name:   name,
		IDs:    map[string]string{"definitionId": id},
		Record: &agent.RecordRef{EntityType: reportRecordEntity, ID: id},
	}
}
