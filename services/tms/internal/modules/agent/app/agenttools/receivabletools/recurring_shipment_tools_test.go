package receivabletools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/recurringshipmentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSeries struct {
	guard     *agenttoolstest.WriteGuard
	current   *recurringshipment.RecurringShipment
	generated bool

	created   *recurringshipment.RecurringShipment
	updated   *recurringshipment.RecurringShipment
	status    *recurringshipmentports.UpdateRecurringShipmentStatusRequest
	generate  *recurringshipmentports.GenerateRecurringShipmentRequest
	createdBy pulid.ID
}

func newFakeSeries() *fakeSeries {
	return &fakeSeries{
		guard: &agenttoolstest.WriteGuard{},
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
	*recurringshipmentports.GetRecurringShipmentByIDRequest,
) (*recurringshipment.RecurringShipment, error) {
	copied := *f.current

	return &copied, nil
}

func (f *fakeSeries) Create(
	_ context.Context,
	entity *recurringshipment.RecurringShipment,
	userID pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	if err := f.guard.Write(); err != nil {
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
	if err := f.guard.Write(); err != nil {
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
	req *recurringshipmentports.UpdateRecurringShipmentStatusRequest,
	_ pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.status = req

	return f.current, nil
}

func (f *fakeSeries) PreviewUpdateStatus(
	_ context.Context,
	req *recurringshipmentports.UpdateRecurringShipmentStatusRequest,
) (*recurringshipmentservice.SeriesChange, error) {
	after := *f.current
	after.Status = req.Status

	return &recurringshipmentservice.SeriesChange{Before: f.current, After: &after}, nil
}

func (f *fakeSeries) Generate(
	_ context.Context,
	req *recurringshipmentports.GenerateRecurringShipmentRequest,
) (*recurringshipmentports.GenerateRecurringShipmentResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.generate = req

	return &recurringshipmentports.GenerateRecurringShipmentResult{
		Series:   f.current,
		Shipment: &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "PRO-55"},
	}, nil
}

func (f *fakeSeries) PreviewGenerate(
	_ context.Context,
	_ *recurringshipmentports.GenerateRecurringShipmentRequest,
) (*recurringshipmentports.RecurringShipmentGenerationPlan, error) {
	return &recurringshipmentports.RecurringShipmentGenerationPlan{
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
	tool := NewCreateRecurringShipmentTool(series)
	source := pulid.MustNew("shp_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramSourceShipmentID:      source.String(),
		paramSeriesName:            "Acme Dallas run",
		ParamCronExpression:        "0 8 * * 1-5",
		ParamTimezone:              "America/Chicago",
		common.FieldRecurringStart: "2026-10-05",
		common.FieldRecurringEnd:   "2026-12-31",
		paramBlackoutDates:         []any{"2026-11-26"},
		paramExceptionPolicy:       "NextBusinessDay",
		paramAutoGenerate:          true,
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, series.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, "0 8 * * 1-5", agenttoolstest.FieldByPath(t, change, ParamCronExpression).After)

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

	tool := NewCreateRecurringShipmentTool(newFakeSeries())
	valid := func(extra map[string]any) map[string]any {
		params := map[string]any{
			paramSourceShipmentID: pulid.MustNew("shp_").String(),
			paramSeriesName:       "Run",
			ParamCronExpression:   "0 8 * * *",
			ParamTimezone:         "UTC",
		}
		for key, value := range extra {
			params[key] = value
		}

		return params
	}
	for name, raw := range map[string]map[string]any{
		"no schedule":        {paramSourceShipmentID: pulid.MustNew("shp_").String(), paramSeriesName: "Run", ParamTimezone: "UTC"},
		"a bad timezone":     valid(map[string]any{ParamTimezone: "Mars/Olympus", common.FieldRecurringStart: "2026-10-05"}),
		"a unix start":       valid(map[string]any{common.FieldRecurringStart: float64(1_900_000_000)}),
		"a bad blackout":     valid(map[string]any{paramBlackoutDates: []any{"Nov 26"}}),
		"an unknown policy":  valid(map[string]any{paramExceptionPolicy: "Whenever"}),
		"too much lead time": valid(map[string]any{paramLeadTimeDays: float64(90)}),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
				agenttoolstest.ExecuteParams(raw)))
		})
	}
}

func TestUpdateRecurringShipment_ChangesOnlyWhatIsSent(t *testing.T) {
	t.Parallel()

	series := newFakeSeries()
	tool := NewUpdateRecurringShipmentTool(series)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramRecurringShipmentID: series.current.ID.String(),
		ParamCronExpression:      "0 6 * * 1-5",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, series.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "0 6 * * 1-5",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), ParamCronExpression).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, series.updated)
	assert.Equal(t, "Acme weekday run", series.updated.Name)
	assert.Equal(t, int64(9), series.updated.Version)
	target, ok := tool.(agentports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceRecurringShipment, target.Resource)

	require.ErrorIs(t, tool.(agentports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{paramRecurringShipmentID: series.current.ID.String()})),
		common.ErrNothingToChange)
}

func TestSetRecurringShipmentStatus_PausesAtTheCurrentVersion(t *testing.T) {
	t.Parallel()

	series := newFakeSeries()
	tool := NewSetRecurringShipmentStatusTool(series)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramRecurringShipmentID: series.current.ID.String(),
		common.FieldStatus:       "Paused",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, series.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Paused", agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), common.FieldStatus).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, series.status)
	assert.Equal(t, recurringshipment.StatusPaused, series.status.Status)
	assert.Equal(t, int64(9), series.status.Version)

	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			paramRecurringShipmentID: series.current.ID.String(),
			common.FieldStatus:       "Stopped",
		})))
}

func TestGenerateRecurringShipment_ShowsTheCopyAndSkipsAFilledSlot(t *testing.T) {
	t.Parallel()

	series := newFakeSeries()
	tool := NewGenerateRecurringShipmentTool(series)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramRecurringShipmentID: series.current.ID.String(),
		paramOccurrenceAt:        "2030-03-17T17:46:40Z",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, series.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	created := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "BOL-2030", agenttoolstest.FieldByPath(t, created, common.FieldBol).After)

	result, err := tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, series.generate)
	require.NotNil(t, series.generate.OccurrenceAt)
	assert.Equal(t, int64(1_900_000_000), *series.generate.OccurrenceAt)
	assert.Equal(t, recurringshipment.RunTriggerManual, series.generate.Trigger)
	assert.Equal(t, params.Actor.UserID, series.generate.RequestedBy)
	assert.Equal(t, "PRO-55", result.Name)
	assert.Equal(t, permission.OpDuplicate, tool.Policy().Operation)

	series.generated = true
	filled, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, filled.Changes)
	assert.Contains(t, filled.Summary, "already has its shipment")
}
