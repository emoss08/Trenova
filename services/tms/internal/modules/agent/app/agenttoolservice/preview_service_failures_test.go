package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/servicefailuretools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type detectingFailures struct {
	servicefailuretools.ServiceFailureDecider

	shipment *shipment.Shipment
	late     []*shipment.Stop
	open     map[pulid.ID]*servicefailure.ServiceFailure
	refuse   error
	saved    []*servicefailure.ServiceFailure
	delayed  bool
	guard    agenttoolstest.WriteGuard
}

func (f *detectingFailures) plan() (*servicefailureports.ServiceFailureDetectionPlan, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}

	plan := &servicefailureports.ServiceFailureDetectionPlan{
		Shipment: f.shipment,
		Detected: make([]servicefailureports.DetectedServiceFailure, 0, len(f.late)),
		SkippedStops: []servicefailureports.ServiceFailureSkippedStop{
			{Reason: "not late after grace"},
		},
	}
	for _, stop := range f.late {
		failure := &servicefailure.ServiceFailure{
			ShipmentID:         f.shipment.ID,
			ShipmentMoveID:     stop.ShipmentMoveID,
			StopID:             stop.ID,
			Type:               servicefailure.TypeForStop(stop),
			Source:             servicefailure.SourceDetected,
			Status:             servicefailure.StatusOpen,
			StopType:           stop.Type,
			ScheduledCutoff:    stop.EffectiveScheduledCutoff(),
			ActualArrival:      *stop.ActualArrival,
			GracePeriodMinutes: 15,
			LateMinutes:        45,
			Notes:              "Detected late arrival",
		}
		existing := f.open[stop.ID]
		if existing != nil {
			updated := *existing
			updated.ScheduledCutoff = failure.ScheduledCutoff
			updated.ActualArrival = failure.ActualArrival
			updated.LateMinutes = failure.LateMinutes
			failure = &updated
		}
		plan.Detected = append(plan.Detected, servicefailureports.DetectedServiceFailure{
			Failure:  failure,
			Existing: existing,
		})
		if existing == nil {
			plan.MarksDelayed = f.shipment.Status != shipmentvocab.StatusDelayed
		}
	}

	return plan, nil
}

func (f *detectingFailures) PreviewEvaluateShipment(
	context.Context,
	*servicefailureports.EvaluateShipmentServiceFailuresRequest,
) (*servicefailureports.ServiceFailureDetectionPlan, error) {
	return f.plan()
}

func (f *detectingFailures) EvaluateShipment(
	context.Context,
	*servicefailureports.EvaluateShipmentServiceFailuresRequest,
	*permissionports.RequestActor,
) (*servicefailureports.ServiceFailureEvaluationResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	plan, err := f.plan()
	if err != nil {
		return nil, err
	}

	for _, detected := range plan.Detected {
		saved := detected.Failure
		if detected.Existing == nil {
			saved.ID = pulid.MustNew("sf_")
			saved.Number = "SF-12"
		}
		f.saved = append(f.saved, saved)
	}
	f.delayed = plan.MarksDelayed

	return &servicefailureports.ServiceFailureEvaluationResult{}, nil
}

func lateShipment() (*shipment.Shipment, *shipment.Stop, *shipment.Stop) {
	moveID := pulid.MustNew("smv_")
	arrival := int64(1_700_003_600)
	pickup := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		ShipmentMoveID:       moveID,
		Type:                 shipmentvocab.StopTypePickup,
		Sequence:             0,
		ScheduledWindowStart: 1_700_000_000,
		ActualArrival:        &arrival,
		Location:             &location.Location{Name: "Dock 4"},
	}
	delivery := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		ShipmentMoveID:       moveID,
		Type:                 shipmentvocab.StopTypeDelivery,
		Sequence:             1,
		ScheduledWindowStart: 1_700_000_000,
		ActualArrival:        &arrival,
		Location:             &location.Location{Name: "Warehouse 12"},
	}

	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "S-40211",
		Status:    shipmentvocab.StatusInTransit,
		Version:   4,
		Moves: []*shipment.ShipmentMove{{
			ID:    moveID,
			Stops: []*shipment.Stop{pickup, delivery},
		}},
	}, pickup, delivery
}

func evaluateParams(entity *shipment.Shipment) agenttoolports.ToolExecuteParams {
	return agenttoolstest.ExecuteParams(map[string]any{"shipmentId": entity.ID.String()})
}

func TestEvaluateServiceFailures_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	entity, pickup, delivery := lateShipment()
	existing := &servicefailure.ServiceFailure{
		ID:              pulid.MustNew("sf_"),
		Number:          "SF-3",
		StopID:          pickup.ID,
		Type:            servicefailure.TypeLatePickup,
		Status:          servicefailure.StatusOpen,
		StopType:        shipmentvocab.StopTypePickup,
		ScheduledCutoff: 1_699_990_000,
		ActualArrival:   1_700_000_900,
		LateMinutes:     10,
		Version:         2,
	}
	before := *existing
	failures := &detectingFailures{
		shipment: entity,
		late:     []*shipment.Stop{pickup, delivery},
		open:     map[pulid.ID]*servicefailure.ServiceFailure{pickup.ID: existing},
	}
	tool := servicefailuretools.NewEvaluateServiceFailuresTool(failures).(*servicefailuretools.EvaluateServiceFailuresTool)
	params := evaluateParams(entity)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&failures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "S-40211")
	assert.Contains(t, preview.Summary, "opening 1 service failure")
	assert.Contains(t, preview.Summary, "refreshing 1 open failure")
	assert.Contains(t, preview.Summary, "mark the shipment delayed")
	require.Len(t, preview.Changes, 3)

	refreshed := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationUpdate, refreshed.Operation)
	assert.Equal(t, existing.ID, refreshed.EntityID)
	assert.Equal(t, "SF-3", refreshed.Label)
	arrived := agenttoolstest.FieldByPath(t, refreshed, "actualArrival")
	assert.Equal(t, "Arrived", arrived.Label)

	opened := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agentpreview.PreviewOperationCreate, opened.Operation)
	assert.Equal(t, permission.ResourceServiceFailure, opened.Resource)
	assert.Equal(t, "Late delivery, Delivery at Warehouse 12", opened.Label)
	assert.Equal(t, "LateDelivery", agenttoolstest.FieldByPath(t, opened, "type").After)

	delayed := agenttoolstest.PreviewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceShipment, delayed.Resource)
	assert.Equal(t, entity.ID, delayed.EntityID)
	assert.Equal(
		t,
		string(shipmentvocab.StatusDelayed),
		agenttoolstest.FieldByPath(t, delayed, "status").After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, failures.saved, 2)
	agenttoolstest.RequireUpdateParity(
		t,
		refreshed,
		&before,
		failures.saved[0],
		servicefailuretools.DetectedFailureOptions()...)
	agenttoolstest.RequireCreateParity(
		t,
		opened,
		failures.saved[1],
		servicefailuretools.DetectedFailureOptions()...)
	assert.True(t, failures.delayed)
}

func TestEvaluateServiceFailures_PreviewSaysWhenNothingIsLate(t *testing.T) {
	t.Parallel()

	entity, _, _ := lateShipment()
	failures := &detectingFailures{shipment: entity}
	tool := servicefailuretools.NewEvaluateServiceFailuresTool(failures).(*servicefailuretools.EvaluateServiceFailuresTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&failures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), evaluateParams(entity))
		},
	)

	assert.Empty(t, preview.Changes)
	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "find no late stop")
}

func TestEvaluateServiceFailures_PreviewWarnsWhenTheShipmentIsRefused(t *testing.T) {
	t.Parallel()

	entity, _, _ := lateShipment()
	failures := &detectingFailures{
		shipment: entity,
		refuse:   errortypes.NewBusinessError("Service failure detection is off"),
	}
	tool := servicefailuretools.NewEvaluateServiceFailuresTool(failures).(*servicefailuretools.EvaluateServiceFailuresTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&failures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), evaluateParams(entity))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}
