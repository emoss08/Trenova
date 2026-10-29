package alerttools

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablechangealertports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamAlertID = "tableChangeAlertId"

const alertSupplier = "The alert, from report_list_table_change_alerts. Never guess one."

var alertStatuses = agenttoolschema.Source(
	"tableChangeAlert.status",
	tablechangealert.SubscriptionStatusValues(),
)

type AlertKeeper interface {
	GetSubscriptionByID(
		ctx context.Context,
		req tablechangealertports.GetTCASubscriptionByIDRequest,
	) (*tablechangealert.TCASubscription, error)
	PlanUpdateSubscription(
		ctx context.Context,
		entity *tablechangealert.TCASubscription,
	) (*kernelports.RecordChange[tablechangealert.TCASubscription], error)
	UpdateSubscription(
		ctx context.Context,
		entity *tablechangealert.TCASubscription,
	) (*tablechangealert.TCASubscription, error)
	PlanSetSubscriptionStatus(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
		status tablechangealert.SubscriptionStatus,
	) (*kernelports.RecordChange[tablechangealert.TCASubscription], error)
	PauseSubscription(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*tablechangealert.TCASubscription, error)
	ResumeSubscription(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*tablechangealert.TCASubscription, error)
	PlanDeleteSubscription(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*tablechangealert.TCASubscription, error)
	DeleteSubscription(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo) error
}

func alertRecord(subscription *tablechangealert.TCASubscription) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceTableChangeAlert,
		ID:       subscription.ID,
		Label:    subscription.Name,
		Version:  changepreview.PinnedVersion(subscription.Version),
	}
}

func targetAlert(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, ParamAlertID, permission.ResourceTableChangeAlert)
}

func alertIDRequest(params *agenttoolports.ToolExecuteParams) (pulid.ID, error) {
	return toolparams.RequirePulid(params.Params, ParamAlertID)
}

type alertEdit struct {
	id     pulid.ID
	values map[string]any
}

func alertEditFrom(params *agenttoolports.ToolExecuteParams) (*alertEdit, error) {
	id, err := toolparams.RequirePulid(params.Params, ParamAlertID)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, len(params.Params))
	for key, value := range params.Params {
		if key != ParamAlertID {
			values[key] = value
		}
	}
	if len(values) == 0 {
		return nil, errors.New("name at least one part of the alert to change")
	}

	return &alertEdit{id: id, values: values}, nil
}

func (e *alertEdit) apply(subscription *tablechangealert.TCASubscription) error {
	multiErr := errortypes.NewMultiError()
	for _, key := range []string{masterdatakit.MdName, alertTableName, alertCustomMessage, "conditionMatch"} {
		if _, given := e.values[key]; !given {
			continue
		}
		text := strings.TrimSpace(toolparams.OptionalString(e.values, key))
		switch key {
		case masterdatakit.MdName:
			subscription.Name = text
		case alertTableName:
			subscription.TableName = text
		case alertCustomMessage:
			subscription.CustomMessage = text
		default:
			if !slices.Contains(alertConditionMatches.Values, text) {
				multiErr.Add(key, errortypes.ErrInvalid, "Use all or any")
				continue
			}
			subscription.ConditionMatch = text
		}
	}
	if _, given := e.values["eventTypes"]; given {
		events := toolparams.StringSliceParam(e.values, "eventTypes")
		for idx, event := range events {
			events[idx] = strings.ToUpper(event)
		}
		subscription.EventTypes = events
	}
	if _, given := e.values["watchedColumns"]; given {
		subscription.WatchedColumns = toolparams.StringSliceParam(e.values, "watchedColumns")
	}
	if _, given := e.values["conditions"]; given {
		subscription.Conditions = alertConditions(e.values)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (e *alertEdit) Build(
	ctx context.Context,
	alerts AlertKeeper,
	params *agenttoolports.ToolExecuteParams,
) (*tablechangealert.TCASubscription, error) {
	stored, err := alerts.GetSubscriptionByID(
		ctx,
		tablechangealertports.GetTCASubscriptionByIDRequest{
			SubscriptionID: e.id,
			TenantInfo:     toolparams.TenantFrom(*params),
		},
	)
	if err != nil {
		return nil, err
	}
	changed := *stored
	changed.User = nil
	changed.Organization = nil
	changed.BusinessUnit = nil
	if err = e.apply(&changed); err != nil {
		return nil, err
	}

	return &changed, nil
}

func NewUpdateTableChangeAlertTool(alerts AlertKeeper) agenttoolports.AgentTool {
	properties := alertProperties(toolparams.KeepWhenLeftOut)
	properties[ParamAlertID] = agenttoolschema.RecordIDText(
		permission.ResourceTableChangeAlert, alertSupplier,
	)

	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "report_update_table_change_alert",
		Description: "Change one of the person's change alerts: what it watches, which " +
			"changes, the conditions or its message. Parts left out keep their value.",
		Resource:    permission.ResourceTableChangeAlert,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale: "Changes an alert whose notices go to the person who owns it; nothing " +
			"else changes.",
		Properties: properties,
		Required:   []string{ParamAlertID},
		SearchTerms: []string{
			alertKind,
			"edit notification",
			"watch different column",
		},
		Target: targetAlert,
	}, toolspec.ReceivablePlan[*alertEdit, *kernelports.RecordChange[tablechangealert.TCASubscription]]{
		Request: alertEditFrom,
		Plan: func(
			ctx context.Context,
			edit *alertEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*kernelports.RecordChange[tablechangealert.TCASubscription], error) {
			entity, err := edit.Build(ctx, alerts, params)
			if err != nil {
				return nil, err
			}

			return alerts.PlanUpdateSubscription(ctx, entity)
		},
		Refused: func(*alertEdit) string { return "Would change a change alert." },
		Render: func(
			_ *alertEdit,
			change *kernelports.RecordChange[tablechangealert.TCASubscription],
		) (*agentpreview.ToolPreview, error) {
			built, err := toolpreview.Changed(alertRecord(change.Before),
				alertViewOf(change.Before), alertViewOf(change.After),
				toolpreview.Labels(alertViewLabels))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would change the alert %q.",
				change.Before.Name), built), nil
		},
		Run: func(
			ctx context.Context,
			edit *alertEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			entity, err := edit.Build(ctx, alerts, params)
			if err != nil {
				return nil, err
			}
			updated, err := alerts.UpdateSubscription(ctx, entity)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: masterdatakit.ActionUpdated,
				Kind:   alertKind,
				Name:   updated.Name,
				IDs:    map[string]string{ParamAlertID: updated.ID.String()},
			}, nil
		},
	})
}

type alertStatusChange struct {
	id     pulid.ID
	status tablechangealert.SubscriptionStatus
}

func NewSetTableChangeAlertStatusTool(alerts AlertKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "report_set_table_change_alert_status",
		Description: "Pause one of the person's change alerts so it stops notifying them, or " +
			"resume a paused one.",
		Resource:    permission.ResourceTableChangeAlert,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale: "Pauses or resumes an alert whose notices go to the person who owns it; " +
			"the opposite call undoes it.",
		Properties: map[string]any{
			ParamAlertID: agenttoolschema.RecordIDText(
				permission.ResourceTableChangeAlert, alertSupplier,
			),
			changepreview.FieldStatus: agenttoolschema.Enum(
				"Paused stops its notices; Active resumes them.",
				alertStatuses,
			),
		},
		Required:    []string{ParamAlertID, changepreview.FieldStatus},
		SearchTerms: []string{"pause alert", "resume alert", "mute notification"},
		Target:      targetAlert,
	}, toolspec.ReceivablePlan[*alertStatusChange, *kernelports.RecordChange[tablechangealert.TCASubscription]]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*alertStatusChange, error) {
			id, err := alertIDRequest(params)
			if err != nil {
				return nil, err
			}
			status, err := toolparams.RequireEnum(
				params.Params,
				changepreview.FieldStatus,
				alertStatuses.Values,
			)
			if err != nil {
				return nil, err
			}

			return &alertStatusChange{id: id, status: status}, nil
		},
		Plan: func(
			ctx context.Context,
			change *alertStatusChange,
			params *agenttoolports.ToolExecuteParams,
		) (*kernelports.RecordChange[tablechangealert.TCASubscription], error) {
			return alerts.PlanSetSubscriptionStatus(ctx, change.id, toolparams.TenantFrom(*params),
				change.status)
		},
		Refused: func(change *alertStatusChange) string {
			return fmt.Sprintf("Would set a change alert %s.", change.status)
		},
		Render: func(
			change *alertStatusChange,
			planned *kernelports.RecordChange[tablechangealert.TCASubscription],
		) (*agentpreview.ToolPreview, error) {
			built, err := toolpreview.Changed(alertRecord(planned.Before),
				alertViewOf(planned.Before), alertViewOf(planned.After),
				toolpreview.Labels(alertViewLabels))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would set the alert %q %s.",
				planned.Before.Name, change.status), built), nil
		},
		Run: func(
			ctx context.Context,
			change *alertStatusChange,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			set := alerts.ResumeSubscription
			if change.status == tablechangealert.SubscriptionStatusPaused {
				set = alerts.PauseSubscription
			}
			updated, err := set(ctx, change.id, toolparams.TenantFrom(*params))
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: masterdatakit.ActionUpdated,
				Kind:   alertKind,
				Name:   updated.Name,
				IDs:    map[string]string{ParamAlertID: updated.ID.String()},
			}, nil
		},
	})
}

func NewDeleteTableChangeAlertTool(alerts AlertKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name:        "report_delete_table_change_alert",
		Description: "Delete one of the person's change alerts they no longer want.",
		Resource:    permission.ResourceTableChangeAlert,
		Operation:   permission.OpDelete,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Rationale: "Removes an alert whose notices went to the person who owns it; " +
			"report_create_table_change_alert sets it up again.",
		Properties: map[string]any{ParamAlertID: agenttoolschema.RecordIDText(
			permission.ResourceTableChangeAlert, alertSupplier,
		)},
		Required:    []string{ParamAlertID},
		SearchTerms: []string{"delete alert", "remove notification", "stop alerting"},
		Target:      targetAlert,
	}, toolspec.ReceivablePlan[pulid.ID, *tablechangealert.TCASubscription]{
		Request: alertIDRequest,
		Plan: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*tablechangealert.TCASubscription, error) {
			return alerts.PlanDeleteSubscription(ctx, id, toolparams.TenantFrom(*params))
		},
		Refused: func(pulid.ID) string { return "Would delete a change alert." },
		Render: func(
			_ pulid.ID,
			subscription *tablechangealert.TCASubscription,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Delete(alertRecord(subscription),
				alertViewOf(subscription), toolpreview.Labels(alertViewLabels))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would delete the alert %q.",
				subscription.Name), change), nil
		},
		Run: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if err := alerts.DeleteSubscription(ctx, id, toolparams.TenantFrom(*params)); err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: changepreview.ActionDeleted,
				Kind:   alertKind,
				IDs:    map[string]string{ParamAlertID: id.String()},
			}, nil
		},
	})
}
