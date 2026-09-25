package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingAlerts struct {
	allowed map[string]bool
	saved   *tablechangealert.TCASubscription
	guard   writeGuard
}

func (f *savingAlerts) CheckSubscription(
	_ context.Context,
	entity *tablechangealert.TCASubscription,
) error {
	if !f.allowed[entity.TableName] {
		return errortypes.NewValidationError(
			"tableName", errortypes.ErrInvalid, "Table is not eligible for change alerts",
		)
	}

	return nil
}

func (f *savingAlerts) CreateSubscription(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	if err := f.CheckSubscription(ctx, entity); err != nil {
		return nil, err
	}
	entity.ID = pulid.MustNew("tcas_")
	f.saved = entity

	return entity, nil
}

func alertParams() map[string]any {
	return map[string]any{
		"name":       "Big loads",
		"tableName":  "shipments",
		"eventTypes": []any{"INSERT", "UPDATE"},
		"conditions": []any{map[string]any{
			"field":    "weight",
			"operator": "gt",
			"value":    float64(40000),
		}},
		"customMessage": "Check the permit.",
	}
}

func TestCreateTableChangeAlert_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	alerts := &savingAlerts{allowed: map[string]bool{"shipments": true}}
	tool := &createTableChangeAlertTool{alerts: alerts}
	params := executeParams(alertParams())

	preview := previewWithoutWrites(t, &alerts.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Warnings)
	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceTableChangeAlert, change.Resource)
	assert.Equal(t, "Watches", fieldByPath(t, change, "table").Label)
	assert.Contains(t, fieldByPath(t, change, "conditions").After, "weight gt 40000")
	assert.Contains(t, preview.Summary, "added or changed")

	require.NoError(t, tool.Execute(t.Context(), params))
	requireCreateParity(t, change, alertViewOf(alerts.saved), toolpreview.Labels(alertViewLabels))
}

func TestCreateTableChangeAlert_PreviewWarnsOnATableThatCannotBeWatched(t *testing.T) {
	t.Parallel()

	alerts := &savingAlerts{}
	tool := &createTableChangeAlertTool{alerts: alerts}

	preview := previewWithoutWrites(t, &alerts.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(alertParams()))
	})

	require.Len(t, preview.Changes, 1)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
