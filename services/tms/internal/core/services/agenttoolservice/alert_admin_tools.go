package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramAlertID  = "tableChangeAlertId"
	alertSupplier = "The alert, from list_table_change_alerts. Never guess one."
)

var alertStatuses = agenttoolschema.Source(
	"tableChangeAlert.status",
	tablechangealert.SubscriptionStatusValues(),
)

type alertKeeper interface {
	GetSubscriptionByID(
		ctx context.Context,
		req repositories.GetTCASubscriptionByIDRequest,
	) (*tablechangealert.TCASubscription, error)
	PlanUpdateSubscription(
		ctx context.Context,
		entity *tablechangealert.TCASubscription,
	) (*serviceports.RecordChange[tablechangealert.TCASubscription], error)
	UpdateSubscription(
		ctx context.Context,
		entity *tablechangealert.TCASubscription,
	) (*tablechangealert.TCASubscription, error)
	PlanSetSubscriptionStatus(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
		status tablechangealert.SubscriptionStatus,
	) (*serviceports.RecordChange[tablechangealert.TCASubscription], error)
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
		Version:  pinnedVersion(subscription.Version),
	}
}

func targetAlert(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramAlertID, permission.ResourceTableChangeAlert)
}

func alertIDRequest(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
	return requirePulid(params.Params, paramAlertID)
}

type alertEdit struct {
	id     pulid.ID
	values map[string]any
}

func alertEditFrom(params *serviceports.ToolExecuteParams) (*alertEdit, error) {
	id, err := requirePulid(params.Params, paramAlertID)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, len(params.Params))
	for key, value := range params.Params {
		if key != paramAlertID {
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
	for _, key := range []string{mdName, alertTableName, alertCustomMessage, "conditionMatch"} {
		if _, given := e.values[key]; !given {
			continue
		}
		text := strings.TrimSpace(optionalString(e.values, key))
		switch key {
		case mdName:
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
		events := stringSliceParam(e.values, "eventTypes")
		for idx, event := range events {
			events[idx] = strings.ToUpper(event)
		}
		subscription.EventTypes = events
	}
	if _, given := e.values["watchedColumns"]; given {
		subscription.WatchedColumns = stringSliceParam(e.values, "watchedColumns")
	}
	if _, given := e.values["conditions"]; given {
		subscription.Conditions = alertConditions(e.values)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (e *alertEdit) build(
	ctx context.Context,
	alerts alertKeeper,
	params *serviceports.ToolExecuteParams,
) (*tablechangealert.TCASubscription, error) {
	stored, err := alerts.GetSubscriptionByID(ctx, repositories.GetTCASubscriptionByIDRequest{
		SubscriptionID: e.id,
		TenantInfo:     tenantFrom(*params),
	})
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

func newUpdateTableChangeAlertTool(alerts alertKeeper) serviceports.AgentTool {
	properties := alertProperties(keepWhenLeftOut)
	properties[paramAlertID] = agenttoolschema.RecordIDText(
		permission.ResourceTableChangeAlert, alertSupplier,
	)

	return newReceivableTool(&receivableSpec{
		name: "update_table_change_alert",
		description: "Change one of the person's change alerts: what it watches, which " +
			"changes, the conditions or its message. Parts left out keep their value.",
		resource:    permission.ResourceTableChangeAlert,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Changes an alert whose notices go to the person who owns it; nothing " +
			"else changes.",
		properties:  properties,
		required:    []string{paramAlertID},
		searchTerms: []string{alertKind, "edit notification", "watch different column"},
		target:      targetAlert,
	}, receivablePlan[*alertEdit, *serviceports.RecordChange[tablechangealert.TCASubscription]]{
		request: alertEditFrom,
		plan: func(
			ctx context.Context,
			edit *alertEdit,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.RecordChange[tablechangealert.TCASubscription], error) {
			entity, err := edit.build(ctx, alerts, params)
			if err != nil {
				return nil, err
			}

			return alerts.PlanUpdateSubscription(ctx, entity)
		},
		refused: func(*alertEdit) string { return "Would change a change alert." },
		render: func(
			_ *alertEdit,
			change *serviceports.RecordChange[tablechangealert.TCASubscription],
		) (*agent.ToolPreview, error) {
			built, err := toolpreview.Changed(alertRecord(change.Before),
				alertViewOf(change.Before), alertViewOf(change.After),
				toolpreview.Labels(alertViewLabels))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would change the alert %q.",
				change.Before.Name), built), nil
		},
		run: func(
			ctx context.Context,
			edit *alertEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := edit.build(ctx, alerts, params)
			if err != nil {
				return nil, err
			}
			updated, err := alerts.UpdateSubscription(ctx, entity)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{Action: actionUpdated, Kind: alertKind,
				Name: updated.Name, IDs: map[string]string{paramAlertID: updated.ID.String()}}, nil
		},
	})
}

type alertStatusChange struct {
	id     pulid.ID
	status tablechangealert.SubscriptionStatus
}

func newSetTableChangeAlertStatusTool(alerts alertKeeper) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "set_table_change_alert_status",
		description: "Pause one of the person's change alerts so it stops notifying them, or " +
			"resume a paused one.",
		resource:    permission.ResourceTableChangeAlert,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Pauses or resumes an alert whose notices go to the person who owns it; " +
			"the opposite call undoes it.",
		properties: map[string]any{
			paramAlertID: agenttoolschema.RecordIDText(
				permission.ResourceTableChangeAlert, alertSupplier,
			),
			fieldStatus: agenttoolschema.Enum("Paused stops its notices; Active resumes them.",
				alertStatuses),
		},
		required:    []string{paramAlertID, fieldStatus},
		searchTerms: []string{"pause alert", "resume alert", "mute notification"},
		target:      targetAlert,
	}, receivablePlan[*alertStatusChange, *serviceports.RecordChange[tablechangealert.TCASubscription]]{
		request: func(params *serviceports.ToolExecuteParams) (*alertStatusChange, error) {
			id, err := alertIDRequest(params)
			if err != nil {
				return nil, err
			}
			status, err := requireEnum(params.Params, fieldStatus, alertStatuses.Values)
			if err != nil {
				return nil, err
			}

			return &alertStatusChange{id: id, status: status}, nil
		},
		plan: func(
			ctx context.Context,
			change *alertStatusChange,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.RecordChange[tablechangealert.TCASubscription], error) {
			return alerts.PlanSetSubscriptionStatus(ctx, change.id, tenantFrom(*params),
				change.status)
		},
		refused: func(change *alertStatusChange) string {
			return fmt.Sprintf("Would set a change alert %s.", change.status)
		},
		render: func(
			change *alertStatusChange,
			planned *serviceports.RecordChange[tablechangealert.TCASubscription],
		) (*agent.ToolPreview, error) {
			built, err := toolpreview.Changed(alertRecord(planned.Before),
				alertViewOf(planned.Before), alertViewOf(planned.After),
				toolpreview.Labels(alertViewLabels))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would set the alert %q %s.",
				planned.Before.Name, change.status), built), nil
		},
		run: func(
			ctx context.Context,
			change *alertStatusChange,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			set := alerts.ResumeSubscription
			if change.status == tablechangealert.SubscriptionStatusPaused {
				set = alerts.PauseSubscription
			}
			updated, err := set(ctx, change.id, tenantFrom(*params))
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{Action: actionUpdated, Kind: alertKind,
				Name: updated.Name, IDs: map[string]string{paramAlertID: updated.ID.String()}}, nil
		},
	})
}

func newDeleteTableChangeAlertTool(alerts alertKeeper) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name:        "delete_table_change_alert",
		description: "Delete one of the person's change alerts they no longer want.",
		resource:    permission.ResourceTableChangeAlert,
		operation:   permission.OpDelete,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		rationale: "Removes an alert whose notices went to the person who owns it; " +
			"create_table_change_alert sets it up again.",
		properties: map[string]any{paramAlertID: agenttoolschema.RecordIDText(
			permission.ResourceTableChangeAlert, alertSupplier,
		)},
		required:    []string{paramAlertID},
		searchTerms: []string{"delete alert", "remove notification", "stop alerting"},
		target:      targetAlert,
	}, receivablePlan[pulid.ID, *tablechangealert.TCASubscription]{
		request: alertIDRequest,
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*tablechangealert.TCASubscription, error) {
			return alerts.PlanDeleteSubscription(ctx, id, tenantFrom(*params))
		},
		refused: func(pulid.ID) string { return "Would delete a change alert." },
		render: func(
			_ pulid.ID,
			subscription *tablechangealert.TCASubscription,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(alertRecord(subscription),
				alertViewOf(subscription), toolpreview.Labels(alertViewLabels))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would delete the alert %q.",
				subscription.Name), change), nil
		},
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if err := alerts.DeleteSubscription(ctx, id, tenantFrom(*params)); err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{Action: actionDeleted, Kind: alertKind,
				IDs: map[string]string{paramAlertID: id.String()}}, nil
		},
	})
}
