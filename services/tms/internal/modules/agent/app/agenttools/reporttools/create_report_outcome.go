package reporttools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

// reportRecordEntity is the record-link registry's name for a saved report
// definition, which opens on the report explorer.
const reportRecordEntity = "report"

func classifyReportVisibility(
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agenttoolports.CallPolicy {
	shared := agenttoolports.CallPolicy{
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

	return agenttoolports.CallPolicy{Egress: agent.EgressPersonal}
}

// ExecuteWithResult saves the report and names it by the id report_get,
// report_run and report_update take, so the conversation that asked for it
// can use it next.
func (t *CreateReportTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	save, err := t.prepare(&params)
	if err != nil {
		return nil, err
	}

	created, err := t.reports.CreateDefinition(ctx, save)
	if err != nil {
		return nil, err
	}

	if created == nil {
		return &agentrecord.ToolExecutionResult{
			Action: changepreview.ResultCreated,
			Kind:   reportRecordEntity,
		}, nil
	}

	return reportResult(created.ID.String(), created.Name), nil
}

// reportResult names a report definition a tool created, by the id the
// report tools take and as the record it opens as.
func reportResult(id, name string) *agentrecord.ToolExecutionResult {
	return &agentrecord.ToolExecutionResult{
		Action: changepreview.ResultCreated,
		Kind:   reportRecordEntity,
		Name:   name,
		IDs:    map[string]string{"definitionId": id},
		Record: &agentrecord.RecordRef{EntityType: reportRecordEntity, ID: id},
	}
}
