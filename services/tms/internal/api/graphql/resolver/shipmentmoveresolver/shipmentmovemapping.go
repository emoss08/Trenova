package shipmentmoveresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/shared/pulid"
)

type assignmentArgs struct {
	MoveID            pulid.ID
	PrimaryWorkerID   pulid.ID
	TractorID         pulid.ID
	TrailerID         *pulid.ID
	SecondaryWorkerID *pulid.ID
}

func assignmentArgsFromInput(
	moveID string,
	input *gqlmodel.ShipmentMoveAssignmentInput,
) (*assignmentArgs, error) {
	parsedMoveID, err := gqlid.ParseIDField("moveId", moveID)
	if err != nil {
		return nil, err
	}
	primaryWorkerID, err := gqlid.ParseIDField("primaryWorkerId", input.PrimaryWorkerID)
	if err != nil {
		return nil, err
	}
	tractorID, err := gqlid.ParseIDField("tractorId", input.TractorID)
	if err != nil {
		return nil, err
	}
	trailerID, err := optionalIDPtr("trailerId", input.TrailerID)
	if err != nil {
		return nil, err
	}
	secondaryWorkerID, err := optionalIDPtr("secondaryWorkerId", input.SecondaryWorkerID)
	if err != nil {
		return nil, err
	}

	return &assignmentArgs{
		MoveID:            parsedMoveID,
		PrimaryWorkerID:   primaryWorkerID,
		TractorID:         tractorID,
		TrailerID:         trailerID,
		SecondaryWorkerID: secondaryWorkerID,
	}, nil
}

func splitMoveRequestFromInput(
	moveID string,
	input *gqlmodel.SplitShipmentMoveInput,
) (*shipmentmoveports.SplitMoveRequest, error) {
	parsedMoveID, err := gqlid.ParseIDField("moveId", moveID)
	if err != nil {
		return nil, err
	}
	locationID, err := gqlid.ParseIDField("newDeliveryLocationId", input.NewDeliveryLocationID)
	if err != nil {
		return nil, err
	}

	return &shipmentmoveports.SplitMoveRequest{
		MoveID:                parsedMoveID,
		NewDeliveryLocationID: locationID,
		SplitPickupTimes:      splitStopTimes(input.SplitPickupTimes),
		NewDeliveryTimes:      splitStopTimes(input.NewDeliveryTimes),
		Pieces:                gqlscalar.Int64Ptr(input.Pieces),
		Weight:                gqlscalar.Int64Ptr(input.Weight),
	}, nil
}

func splitStopTimes(
	input *gqlmodel.ShipmentMoveSplitStopTimesInput,
) shipmentmoveports.SplitStopTimes {
	if input == nil {
		return shipmentmoveports.SplitStopTimes{}
	}
	return shipmentmoveports.SplitStopTimes{
		ScheduledWindowStart: int64(input.ScheduledWindowStart),
		ScheduledWindowEnd:   gqlscalar.Int64Ptr(input.ScheduledWindowEnd),
	}
}

func optionalIDPtr(field string, raw *string) (*pulid.ID, error) {
	id, err := gqlid.ParseOptionalIDField(field, raw)
	if err != nil || id.IsNil() {
		return nil, err
	}
	return &id, nil
}
