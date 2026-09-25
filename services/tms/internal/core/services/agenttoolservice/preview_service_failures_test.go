package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type detectingFailures struct {
	serviceFailureDecider

	shipment *shipment.Shipment
	late     []*shipment.Stop
	open     map[pulid.ID]*servicefailure.ServiceFailure
	refuse   error
	saved    []*servicefailure.ServiceFailure
	delayed  bool
	guard    writeGuard
}

func (f *detectingFailures) plan() (*serviceports.ServiceFailureDetectionPlan, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}

	plan := &serviceports.ServiceFailureDetectionPlan{
		Shipment:     f.shipment,
		Detected:     make([]serviceports.DetectedServiceFailure, 0, len(f.late)),
		SkippedStops: []serviceports.ServiceFailureSkippedStop{{Reason: "not late after grace"}},
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
		plan.Detected = append(plan.Detected, serviceports.DetectedServiceFailure{
			Failure:  failure,
			Existing: existing,
		})
		if existing == nil {
			plan.MarksDelayed = f.shipment.Status != shipment.StatusDelayed
		}
	}

	return plan, nil
}

func (f *detectingFailures) PreviewEvaluateShipment(
	context.Context,
	*serviceports.EvaluateShipmentServiceFailuresRequest,
) (*serviceports.ServiceFailureDetectionPlan, error) {
	return f.plan()
}

func (f *detectingFailures) EvaluateShipment(
	context.Context,
	*serviceports.EvaluateShipmentServiceFailuresRequest,
	*serviceports.RequestActor,
) (*serviceports.ServiceFailureEvaluationResult, error) {
	if err := f.guard.write(); err != nil {
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

	return &serviceports.ServiceFailureEvaluationResult{}, nil
}

func lateShipment() (*shipment.Shipment, *shipment.Stop, *shipment.Stop) {
	moveID := pulid.MustNew("smv_")
	arrival := int64(1_700_003_600)
	pickup := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		ShipmentMoveID:       moveID,
		Type:                 shipment.StopTypePickup,
		Sequence:             0,
		ScheduledWindowStart: 1_700_000_000,
		ActualArrival:        &arrival,
		Location:             &location.Location{Name: "Dock 4"},
	}
	delivery := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		ShipmentMoveID:       moveID,
		Type:                 shipment.StopTypeDelivery,
		Sequence:             1,
		ScheduledWindowStart: 1_700_000_000,
		ActualArrival:        &arrival,
		Location:             &location.Location{Name: "Warehouse 12"},
	}

	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "S-40211",
		Status:    shipment.StatusInTransit,
		Version:   4,
		Moves: []*shipment.ShipmentMove{{
			ID:    moveID,
			Stops: []*shipment.Stop{pickup, delivery},
		}},
	}, pickup, delivery
}

func evaluateParams(entity *shipment.Shipment) serviceports.ToolExecuteParams {
	return executeParams(map[string]any{"shipmentId": entity.ID.String()})
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
		StopType:        shipment.StopTypePickup,
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
	tool := newEvaluateServiceFailuresTool(failures).(*evaluateServiceFailuresTool)
	params := evaluateParams(entity)

	preview := previewWithoutWrites(t, &failures.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "S-40211")
	assert.Contains(t, preview.Summary, "opening 1 service failure")
	assert.Contains(t, preview.Summary, "refreshing 1 open failure")
	assert.Contains(t, preview.Summary, "mark the shipment delayed")
	require.Len(t, preview.Changes, 3)

	refreshed := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationUpdate, refreshed.Operation)
	assert.Equal(t, existing.ID, refreshed.EntityID)
	assert.Equal(t, "SF-3", refreshed.Label)
	arrived := fieldByPath(t, refreshed, "actualArrival")
	assert.Equal(t, "Arrived", arrived.Label)

	opened := previewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationCreate, opened.Operation)
	assert.Equal(t, permission.ResourceServiceFailure, opened.Resource)
	assert.Equal(t, "Late delivery, Delivery at Warehouse 12", opened.Label)
	assert.Equal(t, "LateDelivery", fieldByPath(t, opened, "type").After)

	delayed := previewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceShipment, delayed.Resource)
	assert.Equal(t, entity.ID, delayed.EntityID)
	assert.Equal(t, string(shipment.StatusDelayed), fieldByPath(t, delayed, "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, failures.saved, 2)
	requireUpdateParity(t, refreshed, &before, failures.saved[0], detectedFailureOptions()...)
	requireCreateParity(t, opened, failures.saved[1], detectedFailureOptions()...)
	assert.True(t, failures.delayed)
}

func TestEvaluateServiceFailures_PreviewSaysWhenNothingIsLate(t *testing.T) {
	t.Parallel()

	entity, _, _ := lateShipment()
	failures := &detectingFailures{shipment: entity}
	tool := newEvaluateServiceFailuresTool(failures).(*evaluateServiceFailuresTool)

	preview := previewWithoutWrites(t, &failures.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), evaluateParams(entity))
	})

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
	tool := newEvaluateServiceFailuresTool(failures).(*evaluateServiceFailuresTool)

	preview := previewWithoutWrites(t, &failures.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), evaluateParams(entity))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
