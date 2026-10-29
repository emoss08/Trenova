package alerttools_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablechangealertports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAlerts struct {
	guard      *agenttoolstest.WriteGuard
	stored     *tablechangealert.TCASubscription
	getRequest tablechangealertports.GetTCASubscriptionByIDRequest
	updated    *tablechangealert.TCASubscription
	paused     bool
	resumed    bool
	deleted    bool
}

func (f *fakeAlerts) GetSubscriptionByID(
	_ context.Context,
	req tablechangealertports.GetTCASubscriptionByIDRequest,
) (*tablechangealert.TCASubscription, error) {
	f.getRequest = req
	copied := *f.stored

	return &copied, nil
}

func (f *fakeAlerts) PlanUpdateSubscription(
	_ context.Context,
	entity *tablechangealert.TCASubscription,
) (*kernelports.RecordChange[tablechangealert.TCASubscription], error) {
	if entity.TableName == "secrets" {
		return nil, errortypes.NewValidationError("tableName", errortypes.ErrInvalid,
			"Table is not eligible for change alerts")
	}

	return &kernelports.RecordChange[tablechangealert.TCASubscription]{
		Before: f.stored,
		After:  entity,
	}, nil
}

func (f *fakeAlerts) UpdateSubscription(
	_ context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = entity

	return entity, nil
}

func (f *fakeAlerts) PlanSetSubscriptionStatus(
	_ context.Context,
	_ pulid.ID,
	_ pagination.TenantInfo,
	status tablechangealert.SubscriptionStatus,
) (*kernelports.RecordChange[tablechangealert.TCASubscription], error) {
	changed := *f.stored
	changed.Status = status

	return &kernelports.RecordChange[tablechangealert.TCASubscription]{
		Before: f.stored,
		After:  &changed,
	}, nil
}

func (f *fakeAlerts) PauseSubscription(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	f.paused = true

	return f.stored, f.guard.Write()
}

func (f *fakeAlerts) ResumeSubscription(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	f.resumed = true

	return f.stored, f.guard.Write()
}

func (f *fakeAlerts) PlanDeleteSubscription(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	return f.stored, nil
}

func (f *fakeAlerts) DeleteSubscription(context.Context, pulid.ID, pagination.TenantInfo) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.deleted = true

	return nil
}

func TestTableChangeAlertTools_ActOnlyOnTheCallersOwnAlert(t *testing.T) {
	t.Parallel()

	alerts := &fakeAlerts{
		guard: &agenttoolstest.WriteGuard{},
		stored: &tablechangealert.TCASubscription{
			ID:             pulid.MustNew("tcas_"),
			Name:           "Delayed loads",
			TableName:      "shipments",
			EventTypes:     []string{"UPDATE"},
			ConditionMatch: "all",
			Status:         tablechangealert.SubscriptionStatusActive,
			Version:        3,
		},
	}
	update := alerttools.NewUpdateTableChangeAlertTool(alerts)
	params := agenttoolstest.ExecuteParams(map[string]any{
		alerttools.ParamAlertID: alerts.stored.ID.String(),
		"customMessage":         "A load slipped",
		"eventTypes":            []any{"update", "insert"},
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		alerts.guard,
		func() (*agentpreview.ToolPreview, error) {
			return update.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would change the alert")
	require.NoError(t, update.Execute(t.Context(), params))
	assert.Equal(t, params.Actor.UserID, alerts.getRequest.TenantInfo.UserID)
	assert.Equal(t, "A load slipped", alerts.updated.CustomMessage)
	assert.Equal(t, []string{"UPDATE", "INSERT"}, alerts.updated.EventTypes)
	assert.Equal(t, "Delayed loads", alerts.updated.Name)

	require.Error(
		t,
		update.(agenttoolports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{
				alerttools.ParamAlertID: alerts.stored.ID.String(),
				"tableName":             "secrets",
			},
		)),
	)

	status := alerttools.NewSetTableChangeAlertStatusTool(alerts)
	require.NoError(t, status.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		alerttools.ParamAlertID:   alerts.stored.ID.String(),
		changepreview.FieldStatus: "Paused",
	})))
	assert.True(t, alerts.paused)
	require.NoError(t, status.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		alerttools.ParamAlertID:   alerts.stored.ID.String(),
		changepreview.FieldStatus: "Active",
	})))
	assert.True(t, alerts.resumed)

	remove := alerttools.NewDeleteTableChangeAlertTool(alerts)
	deletePreview := agenttoolstest.PreviewWithoutWrites(
		t,
		alerts.guard,
		func() (*agentpreview.ToolPreview, error) {
			return remove.(agenttoolports.ToolPreviewer).Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(
					map[string]any{alerttools.ParamAlertID: alerts.stored.ID.String()}),
			)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, deletePreview, 0).Operation,
	)
	require.NoError(t, remove.Execute(t.Context(), agenttoolstest.ExecuteParams(
		map[string]any{alerttools.ParamAlertID: alerts.stored.ID.String()})))
	assert.True(t, alerts.deleted)
}
