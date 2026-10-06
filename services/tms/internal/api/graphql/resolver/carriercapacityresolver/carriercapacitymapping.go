package carriercapacityresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func optionalIDPtr(field string, value *string) (*pulid.ID, error) {
	id, err := base.OptionalScopedID(field, value)
	if err != nil || id.IsNil() {
		return nil, err
	}
	return &id, nil
}

func postingFromInput(
	input *gqlmodel.CarrierCapacityPostingInput,
	tenantInfo pagination.TenantInfo,
) (*carriercapacity.Posting, error) {
	multiErr := errortypes.NewMultiError()

	carrierID, err := base.RequiredID("carrierId", input.CarrierID)
	if err != nil {
		return nil, err
	}

	posting := &carriercapacity.Posting{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		CarrierID:         carrierID,
		OriginRadiusMiles: input.OriginRadiusMiles,
		AvailableFrom:     int64(input.AvailableFrom),
		AvailableTo:       int64(input.AvailableTo),
		TruckCount:        1,
		RateMethod:        carriercapacity.RateMethodPerMile,
		Source:            carriercapacity.SourceManual,
		Notes:             base.StringValue(input.Notes),
	}
	if input.TruckCount != nil {
		posting.TruckCount = *input.TruckCount
	}
	if input.RateMethod != nil {
		posting.RateMethod = *input.RateMethod
	}
	if input.Source != nil {
		posting.Source = *input.Source
	}

	refs := []struct {
		field string
		raw   *string
		dest  **pulid.ID
	}{
		{"originLocationId", input.OriginLocationID, &posting.OriginLocationID},
		{"originStateId", input.OriginStateID, &posting.OriginStateID},
		{"destinationStateId", input.DestinationStateID, &posting.DestinationStateID},
		{"equipmentTypeId", input.EquipmentTypeID, &posting.EquipmentTypeID},
	}
	for _, ref := range refs {
		id, idErr := optionalIDPtr(ref.field, ref.raw)
		if idErr != nil {
			multiErr.Add(ref.field, errortypes.ErrInvalid, "Value must be a valid id")
			continue
		}
		*ref.dest = id
	}

	rate, rateErr := base.ParseNullDecimalField("rate", input.Rate)
	if rateErr != nil {
		multiErr.Add("rate", errortypes.ErrInvalid, "Rate must be a number")
	}
	posting.Rate = rate

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return posting, nil
}

func postingConnectionToModel(
	result *pagination.CursorListResult[*carriercapacity.Posting],
) (*gqlmodel.CarrierCapacityPostingConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *carriercapacity.Posting, cursor string) *gqlmodel.CarrierCapacityPostingEdge {
			return &gqlmodel.CarrierCapacityPostingEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.CarrierCapacityPostingEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CarrierCapacityPostingConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
