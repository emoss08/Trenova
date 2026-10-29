package accountingtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/timeutils"
)

var accountingCheckVolatileFields = []string{"lastCheckedAt", "lastSuccessAt"}

var accountingCheckFields = []string{
	changepreview.FieldStatus,
	"lastCheckedAt",
	"lastSuccessAt",
	"consecutiveFailures",
	"lastErrorCategory",
	"lastErrorMessage",
}

var mappingFields = []string{
	changepreview.FieldState,
	"externalName",
	"source",
	changepreview.FieldReason,
	"confirmedAt",
}

var mappingLabels = map[string]string{"externalName": "Mapped to"}

func accountingConnectionRecord(
	conn *accountingconnection.AccountingConnection,
) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		ID:       conn.ID,
		Label: fmt.Sprintf(
			"%s for %s",
			accountingprovider.ProviderName(conn.IntegrationType),
			conn.ExternalCompanyName,
		),
		Version: changepreview.PreviewVersion(conn.Version),
	}
}

func MappingRecord(row *accountingmapping.AccountingMapping) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		ID:       row.ID,
		Label:    row.TargetLabel,
		Version:  changepreview.PreviewVersion(row.Version),
	}
}

func MappingOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(mappingFields...),
		toolpreview.Labels(mappingLabels),
		toolpreview.Volatile("confirmedAt"),
	}
}

func (t *CheckAccountingConnectionTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	conn, err := t.connection(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
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
		func(checked *accountingconnection.AccountingConnection) error {
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
		accountingprovider.ProviderName(conn.IntegrationType),
		conn.ExternalCompanyName,
		conn.Status,
	), change)
	preview.Partial = true

	return preview, nil
}

func (t *SetAccountingMappingTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	req, row, ref, err := t.request(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
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
		if _, err = t.history.GuardHistory(
			ctx,
			req.TenantInfo,
			row,
			req.AcknowledgeHistory,
		); err != nil {
			if changepreview.IsRefusal(err) {
				return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
			}

			return nil, err
		}
	}

	choice := accountingmappingkit.MappingChoice(req, ref, timeutils.NowUnix())
	change, err := toolpreview.Update(
		MappingRecord(row),
		row,
		func(mapped *accountingmapping.AccountingMapping) error {
			mapped.Confirm(choice)

			return nil
		},
		MappingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}

func (t *ClearAccountingMappingTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would clear an accounting mapping."),
				err,
			), nil
		}

		return nil, err
	}

	summary := fmt.Sprintf(
		"Would unmatch %s from %s, and never propose %s for it again.",
		row.TargetLabel,
		MappedLabel(row),
		MappedLabel(row),
	)
	if _, err = t.history.GuardHistory(
		ctx,
		toolparams.TenantFrom(&params),
		row,
		false,
	); err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	plan, err := changepreview.PlanUpdate(
		MappingRecord(row),
		row,
		func(cleared *accountingmapping.AccountingMapping) error { return cleared.Clear() },
		MappingOptions()...,
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
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would create a record in the accounting system."),
				err,
			), nil
		}

		return nil, err
	}

	system, err := SystemFrom(params.Params)
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

	choice := accountingmappingkit.CreatedReferenceChoice(
		&accountingmappingkit.CreatedReference{
			ExternalName:    name,
			RequestedSource: accountingmapping.MappingSourceAgent,
			IntegrationType: system,
			ActorID:         params.Actor.UserID,
			At:              timeutils.NowUnix(),
		},
	)
	mapped, err := toolpreview.Update(
		MappingRecord(row),
		row,
		func(confirmed *accountingmapping.AccountingMapping) error {
			confirmed.Confirm(choice)

			return nil
		},
		MappingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would create the %s %q in %s and map %s to it. The accounting system may adjust "+
			"the name to its own rules; it cannot be deleted from Trenova afterwards.",
		strings.ToLower(string(row.ProviderKind)),
		name,
		accountingprovider.ProviderName(system),
		row.TargetLabel,
	), created, mapped)
	preview.Partial = true

	return preview, nil
}

func (t *CreateAccountingReferenceRecordTool) requestedName(
	params *agenttoolports.ToolExecuteParams,
	row *accountingmapping.AccountingMapping,
) string {
	if name := strings.TrimSpace(toolparams.OptionalString(params.Params, "name")); name != "" {
		return name
	}

	return row.TargetLabel
}

func (t *RefreshAccountingReferenceDataTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	_, summary, err := t.setup(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
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
	), &agentpreview.RecordChange{
		Resource:  record.Resource,
		EntityID:  record.ID,
		Label:     record.Label,
		Operation: agentpreview.PreviewOperationRun,
		Version:   record.Version,
	})
	preview.Partial = true

	return preview, nil
}
