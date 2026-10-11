package accountingtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/timeutils"
)

var accountingCheckVolatileFields = []string{"lastCheckedAt", "lastSuccessAt"}

var accountingCheckFields = []string{
	common.FieldStatus,
	"lastCheckedAt",
	"lastSuccessAt",
	"consecutiveFailures",
	"lastErrorCategory",
	"lastErrorMessage",
}

var mappingFields = []string{common.FieldState, "externalName", "source", common.FieldReason, "confirmedAt"}

var mappingLabels = map[string]string{"externalName": "Mapped to"}

func accountingConnectionRecord(conn *accountingsync.AccountingConnection) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		ID:       conn.ID,
		Label: fmt.Sprintf(
			"%s for %s",
			accountingsync.ProviderName(conn.IntegrationType),
			conn.ExternalCompanyName,
		),
		Version: common.PreviewVersion(conn.Version),
	}
}

func mappingRecord(row *accountingsync.AccountingMapping) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		ID:       row.ID,
		Label:    row.TargetLabel,
		Version:  common.PreviewVersion(row.Version),
	}
}

func mappingOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(mappingFields...),
		toolpreview.Labels(mappingLabels),
		toolpreview.Volatile("confirmedAt"),
	}
}

func (t *CheckAccountingConnectionTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	conn, err := t.connection(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(
				toolpreview.Build("Would check the accounting connection now."),
				err,
			), nil
		}

		return nil, err
	}

	now := timeutils.NowUnix()
	change, err := toolpreview.Update(
		accountingConnectionRecord(conn),
		conn,
		func(checked *accountingsync.AccountingConnection) error {
			checked.RecordSuccess(now)

			return nil
		},
		toolpreview.Only(accountingCheckFields...),
		toolpreview.Volatile(accountingCheckVolatileFields...),
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would check %s for %s now and record whether it answers. It is %s at the moment; "+
			"shown is what is recorded when it answers, and a failure is recorded instead "+
			"when it does not.",
		accountingsync.ProviderName(conn.IntegrationType),
		conn.ExternalCompanyName,
		conn.Status,
	), change)
	preview.Partial = true

	return preview, nil
}

func (t *SetAccountingMappingTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, row, ref, err := t.request(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(
				toolpreview.Build("Would set an accounting mapping."),
				err,
			), nil
		}

		return nil, err
	}

	summary := fmt.Sprintf(
		"Would send %s as %s in the accounting system and confirm it.",
		row.TargetLabel,
		ref.Label(),
	)
	if row.ExternalID != ref.ExternalID {
		if _, err = t.mappings.GuardHistory(
			ctx,
			req.TenantInfo,
			row,
			req.AcknowledgeHistory,
		); err != nil {
			if common.IsRefusal(err) {
				return common.WarnWouldFail(toolpreview.Build(summary), err), nil
			}

			return nil, err
		}
	}

	choice := accountingmappingservice.MappingChoice(req, ref, timeutils.NowUnix())
	change, err := toolpreview.Update(
		mappingRecord(row),
		row,
		func(mapped *accountingsync.AccountingMapping) error {
			mapped.Confirm(choice)

			return nil
		},
		mappingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}

func (t *ClearAccountingMappingTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(
				toolpreview.Build("Would clear an accounting mapping."),
				err,
			), nil
		}

		return nil, err
	}

	summary := fmt.Sprintf(
		"Would unmatch %s from %s, and never propose %s for it again.",
		row.TargetLabel,
		mappedLabel(row),
		mappedLabel(row),
	)
	if _, err = t.mappings.GuardHistory(ctx, common.TenantFrom(params), row, false); err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	plan, err := common.PlanUpdate(
		mappingRecord(row),
		row,
		func(cleared *accountingsync.AccountingMapping) error { return cleared.Clear() },
		mappingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(summary), nil
}

type accountingRecordDraft struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func (t *CreateAccountingReferenceRecordTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(
				toolpreview.Build("Would create a record in the accounting system."),
				err,
			), nil
		}

		return nil, err
	}

	system, err := accountingSystemFrom(params.Params)
	if err != nil {
		return nil, err
	}

	name := t.requestedName(&params, row)
	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		Label:    name,
	}, &accountingRecordDraft{Kind: string(row.ProviderKind), Name: name})
	if err != nil {
		return nil, err
	}

	choice := accountingmappingservice.CreatedReferenceChoice(
		&accountingmappingservice.CreatedReference{
			ExternalName:    name,
			RequestedSource: accountingsync.MappingSourceAgent,
			IntegrationType: system,
			ActorID:         params.Actor.UserID,
			At:              timeutils.NowUnix(),
		},
	)
	mapped, err := toolpreview.Update(
		mappingRecord(row),
		row,
		func(confirmed *accountingsync.AccountingMapping) error {
			confirmed.Confirm(choice)

			return nil
		},
		mappingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would create the %s %q in %s and map %s to it. The accounting system may adjust "+
			"the name to its own rules; it cannot be deleted from Trenova afterwards.",
		strings.ToLower(string(row.ProviderKind)),
		name,
		accountingsync.ProviderName(system),
		row.TargetLabel,
	), created, mapped)
	preview.Partial = true

	return preview, nil
}

func (t *CreateAccountingReferenceRecordTool) requestedName(
	params *agentports.ToolExecuteParams,
	row *accountingsync.AccountingMapping,
) string {
	if name := strings.TrimSpace(common.OptionalString(params.Params, "name")); name != "" {
		return name
	}

	return row.TargetLabel
}

func (t *RefreshAccountingReferenceDataTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	_, summary, err := t.setup(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(
				toolpreview.Build("Would refresh the accounting system's records."),
				err,
			), nil
		}

		return nil, err
	}

	record := accountingConnectionRecord(summary.Connection)
	preview := toolpreview.Build(fmt.Sprintf(
		"Would read %s's accounts, items, customers, vendors, terms and payment methods "+
			"for %s again, in the background, and refresh the open proposals. Confirmed "+
			"mappings and the books are left alone.",
		summary.ProviderName,
		summary.Connection.ExternalCompanyName,
	), &agent.RecordChange{
		Resource:  record.Resource,
		EntityID:  record.ID,
		Label:     record.Label,
		Operation: agent.PreviewOperationRun,
		Version:   record.Version,
	})
	preview.Partial = true

	return preview, nil
}
