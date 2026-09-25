package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingmappingservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	_ serviceports.ToolPreviewer = (*checkAccountingConnectionTool)(nil)
	_ serviceports.ToolPreviewer = (*setAccountingMappingTool)(nil)
	_ serviceports.ToolPreviewer = (*clearAccountingMappingTool)(nil)
	_ serviceports.ToolPreviewer = (*createAccountingReferenceRecordTool)(nil)
	_ serviceports.ToolPreviewer = (*refreshAccountingReferenceDataTool)(nil)
)

var accountingCheckVolatileFields = []string{"lastCheckedAt", "lastSuccessAt"}

var accountingCheckFields = []string{
	"status",
	"lastCheckedAt",
	"lastSuccessAt",
	"consecutiveFailures",
	"lastErrorCategory",
	"lastErrorMessage",
}

var mappingFields = []string{"state", "externalName", "source", "reason", "confirmedAt"}

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
		Version: previewVersion(conn.Version),
	}
}

func mappingRecord(row *accountingsync.AccountingMapping) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		ID:       row.ID,
		Label:    row.TargetLabel,
		Version:  previewVersion(row.Version),
	}
}

func mappingOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(mappingFields...),
		toolpreview.Labels(mappingLabels),
		toolpreview.Volatile("confirmedAt"),
	}
}

func (t *checkAccountingConnectionTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	conn, err := t.connection(ctx, &params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(
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

func (t *setAccountingMappingTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	req, row, ref, err := t.request(ctx, &params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(
				toolpreview.Build("Would set an accounting mapping."),
				err,
			), nil
		}

		return nil, err
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

	return toolpreview.Build(fmt.Sprintf(
		"Would send %s as %s in the accounting system and confirm it.",
		row.TargetLabel,
		ref.Label(),
	), change), nil
}

func (t *clearAccountingMappingTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(
				toolpreview.Build("Would clear an accounting mapping."),
				err,
			), nil
		}

		return nil, err
	}

	plan, err := planUpdate(
		mappingRecord(row),
		row,
		func(cleared *accountingsync.AccountingMapping) error { return cleared.Clear() },
		mappingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return plan.preview(fmt.Sprintf(
		"Would unmatch %s from %s, and never propose %s for it again.",
		row.TargetLabel,
		mappedLabel(row),
		mappedLabel(row),
	)), nil
}

type accountingRecordDraft struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func (t *createAccountingReferenceRecordTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(
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

	name := t.requestedName(params, row)
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

func (t *createAccountingReferenceRecordTool) requestedName(
	params serviceports.ToolExecuteParams,
	row *accountingsync.AccountingMapping,
) string {
	if name := strings.TrimSpace(optionalString(params.Params, "name")); name != "" {
		return name
	}

	return row.TargetLabel
}

func (t *refreshAccountingReferenceDataTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	_, summary, err := t.setup(ctx, &params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(
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
