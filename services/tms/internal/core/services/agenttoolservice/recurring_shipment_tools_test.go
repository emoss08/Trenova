package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/recurringshipmentservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSeries struct {
	guard     *writeGuard
	current   *recurringshipment.RecurringShipment
	generated bool

	created   *recurringshipment.RecurringShipment
	updated   *recurringshipment.RecurringShipment
	status    *repositories.UpdateRecurringShipmentStatusRequest
	generate  *repositories.GenerateRecurringShipmentRequest
	createdBy pulid.ID
}

func newFakeSeries() *fakeSeries {
	return &fakeSeries{
		guard: &writeGuard{},
		current: &recurringshipment.RecurringShipment{
			ID:               pulid.MustNew("rsh_"),
			SourceShipmentID: pulid.MustNew("shp_"),
			Name:             "Acme weekday run",
			Status:           recurringshipment.StatusActive,
			CronExpression:   "0 8 * * 1-5",
			Timezone:         "America/Chicago",
			ExceptionPolicy:  recurringshipment.ExceptionPolicySkip,
			LeadTimeDays:     1,
			Version:          9,
		},
	}
}

func (f *fakeSeries) Get(
	context.Context,
	*repositories.GetRecurringShipmentByIDRequest,
) (*recurringshipment.RecurringShipment, error) {
	copied := *f.current

	return &copied, nil
}

func (f *fakeSeries) Create(
	_ context.Context,
	entity *recurringshipment.RecurringShipment,
	userID pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = entity
	f.createdBy = userID

	return entity, nil
}

func (f *fakeSeries) PreviewCreate(
	_ context.Context,
	entity *recurringshipment.RecurringShipment,
	_ pulid.ID,
) (*recurringshipmentservice.SeriesChange, error) {
	return &recurringshipmentservice.SeriesChange{After: entity}, nil
}

func (f *fakeSeries) Update(
	_ context.Context,
	entity *recurringshipment.RecurringShipment,
	_ pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = entity

	return entity, nil
}

func (f *fakeSeries) PreviewUpdate(
	_ context.Context,
	entity *recurringshipment.RecurringShipment,
) (*recurringshipmentservice.SeriesChange, error) {
	return &recurringshipmentservice.SeriesChange{Before: f.current, After: entity}, nil
}

func (f *fakeSeries) UpdateStatus(
	_ context.Context,
	req *repositories.UpdateRecurringShipmentStatusRequest,
	_ pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.status = req

	return f.current, nil
}

func (f *fakeSeries) PreviewUpdateStatus(
	_ context.Context,
	req *repositories.UpdateRecurringShipmentStatusRequest,
) (*recurringshipmentservice.SeriesChange, error) {
	after := *f.current
	after.Status = req.Status

	return &recurringshipmentservice.SeriesChange{Before: f.current, After: &after}, nil
}

func (f *fakeSeries) Generate(
	_ context.Context,
	req *repositories.GenerateRecurringShipmentRequest,
) (*repositories.GenerateRecurringShipmentResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.generate = req

	return &repositories.GenerateRecurringShipmentResult{
		Series:   f.current,
		Shipment: &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "PRO-55"},
	}, nil
}

func (f *fakeSeries) PreviewGenerate(
	_ context.Context,
	_ *repositories.GenerateRecurringShipmentRequest,
) (*repositories.RecurringShipmentGenerationPlan, error) {
	return &repositories.RecurringShipmentGenerationPlan{
		Series:           f.current,
		Occurrence:       &recurringshipment.Occurrence{At: 1_900_000_000},
		AlreadyGenerated: f.generated,
		Shipment: &shipment.Shipment{
			CustomerID: pulid.MustNew("cus_"),
			BOL:        "BOL-2030",
			Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
				{ScheduledWindowStart: 1_900_000_000},
			}}},
		},
	}, nil
}

func TestCreateRecurringShipment_ReadsDaysInTheSeriesTimezone(t *testing.T) {
	t.Parallel()

	series := newFakeSeries()
	tool := newCreateRecurringShipmentTool(series)
	source := pulid.MustNew("shp_")
	params := executeParams(map[string]any{
		paramSourceShipmentID: source.String(),
		paramSeriesName:       "Acme Dallas run",
		paramCronExpression:   "0 8 * * 1-5",
		paramTimezone:         "America/Chicago",
		fieldRecurringStart:   "2026-10-05",
		fieldRecurringEnd:     "2026-12-31",
		paramBlackoutDates:    []any{"2026-11-26"},
		paramExceptionPolicy:  "NextBusinessDay",
		paramAutoGenerate:     true,
	})

	preview := previewWithoutWrites(t, series.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, "0 8 * * 1-5", fieldByPath(t, change, paramCronExpression).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, series.created)
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 5, 0, 0, 0, 0, chicago).Unix(), series.created.StartDate)
	require.NotNil(t, series.created.EndDate)
	assert.Equal(t, time.Date(2027, 1, 1, 0, 0, 0, 0, chicago).Unix()-1, *series.created.EndDate)
	assert.Equal(t, []string{"2026-11-26"}, series.created.BlackoutDates)
	assert.Equal(t, recurringshipment.ExceptionPolicyNextBusinessDay,
		series.created.ExceptionPolicy)
	assert.True(t, series.created.AutoGenerate)
	assert.Equal(t, source, series.created.SourceShipmentID)
	assert.Equal(t, params.Actor.UserID, series.createdBy)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceRecurringShipment, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
}

func TestCreateRecurringShipment_RefusesBadArguments(t *testing.T) {
	t.Parallel()

	tool := newCreateRecurringShipmentTool(newFakeSeries())
	valid := func(extra map[string]any) map[string]any {
		params := map[string]any{
			paramSourceShipmentID: pulid.MustNew("shp_").String(),
			paramSeriesName:       "Run",
			paramCronExpression:   "0 8 * * *",
			paramTimezone:         "UTC",
		}
		for key, value := range extra {
			params[key] = value
		}

		return params
	}
	for name, raw := range map[string]map[string]any{
		"no schedule":        {paramSourceShipmentID: pulid.MustNew("shp_").String(), paramSeriesName: "Run", paramTimezone: "UTC"},
		"a bad timezone":     valid(map[string]any{paramTimezone: "Mars/Olympus", fieldRecurringStart: "2026-10-05"}),
		"a unix start":       valid(map[string]any{fieldRecurringStart: float64(1_900_000_000)}),
		"a bad blackout":     valid(map[string]any{paramBlackoutDates: []any{"Nov 26"}}),
		"an unknown policy":  valid(map[string]any{paramExceptionPolicy: "Whenever"}),
		"too much lead time": valid(map[string]any{paramLeadTimeDays: float64(90)}),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
				executeParams(raw)))
		})
	}
}

func TestUpdateRecurringShipment_ChangesOnlyWhatIsSent(t *testing.T) {
	t.Parallel()

	series := newFakeSeries()
	tool := newUpdateRecurringShipmentTool(series)
	params := executeParams(map[string]any{
		paramRecurringShipmentID: series.current.ID.String(),
		paramCronExpression:      "0 6 * * 1-5",
	})

	preview := previewWithoutWrites(t, series.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "0 6 * * 1-5",
		fieldByPath(t, previewChange(t, preview, 0), paramCronExpression).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, series.updated)
	assert.Equal(t, "Acme weekday run", series.updated.Name)
	assert.Equal(t, int64(9), series.updated.Version)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceRecurringShipment, target.Resource)

	require.ErrorIs(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramRecurringShipmentID: series.current.ID.String()})),
		errNothingToChange)
}

func TestSetRecurringShipmentStatus_PausesAtTheCurrentVersion(t *testing.T) {
	t.Parallel()

	series := newFakeSeries()
	tool := newSetRecurringShipmentStatusTool(series)
	params := executeParams(map[string]any{
		paramRecurringShipmentID: series.current.ID.String(),
		fieldStatus:              "Paused",
	})

	preview := previewWithoutWrites(t, series.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Paused", fieldByPath(t, previewChange(t, preview, 0), fieldStatus).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, series.status)
	assert.Equal(t, recurringshipment.StatusPaused, series.status.Status)
	assert.Equal(t, int64(9), series.status.Version)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramRecurringShipmentID: series.current.ID.String(),
			fieldStatus:              "Stopped",
		})))
}

func TestGenerateRecurringShipment_ShowsTheCopyAndSkipsAFilledSlot(t *testing.T) {
	t.Parallel()

	series := newFakeSeries()
	tool := newGenerateRecurringShipmentTool(series)
	params := executeParams(map[string]any{
		paramRecurringShipmentID: series.current.ID.String(),
		paramOccurrenceAt:        "2030-03-17T17:46:40Z",
	})

	preview := previewWithoutWrites(t, series.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	created := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "BOL-2030", fieldByPath(t, created, fieldBol).After)

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, series.generate)
	require.NotNil(t, series.generate.OccurrenceAt)
	assert.Equal(t, int64(1_900_000_000), *series.generate.OccurrenceAt)
	assert.Equal(t, recurringshipment.RunTriggerManual, series.generate.Trigger)
	assert.Equal(t, params.Actor.UserID, series.generate.RequestedBy)
	assert.Equal(t, "PRO-55", result.Name)
	assert.Equal(t, permission.OpDuplicate, tool.Policy().Operation)

	series.generated = true
	filled, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, filled.Changes)
	assert.Contains(t, filled.Summary, "already has its shipment")
}
