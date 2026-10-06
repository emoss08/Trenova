package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/sliceutils"
)

type capacityPostingRow struct {
	ID               string `json:"id"`
	CarrierID        string `json:"carrierId"`
	Carrier          string `json:"carrier,omitempty"`
	OriginLocation   string `json:"originLocation,omitempty"`
	OriginState      string `json:"originState,omitempty"`
	RadiusMiles      *int   `json:"originRadiusMiles,omitempty"`
	DestinationState string `json:"destinationState,omitempty"`
	EquipmentType    string `json:"equipmentType,omitempty"`
	AvailableFrom    int64  `json:"availableFrom"`
	AvailableTo      int64  `json:"availableTo"`
	TruckCount       int    `json:"truckCount"`
	RateMethod       string `json:"rateMethod"`
	Rate             string `json:"rate,omitempty"`
	Source           string `json:"source"`
	Version          int64  `json:"version"`
}

func toCapacityPostingRow(item *carriercapacity.Posting) capacityPostingRow {
	row := capacityPostingRow{
		ID:            item.ID.String(),
		CarrierID:     item.CarrierID.String(),
		RadiusMiles:   item.OriginRadiusMiles,
		AvailableFrom: item.AvailableFrom,
		AvailableTo:   item.AvailableTo,
		TruckCount:    item.TruckCount,
		RateMethod:    string(item.RateMethod),
		Source:        string(item.Source),
		Version:       item.Version,
	}
	if item.Carrier != nil {
		row.Carrier = item.Carrier.Name
	}
	if item.OriginLocation != nil {
		row.OriginLocation = item.OriginLocation.Name + ", " + item.OriginLocation.City
	}
	if item.OriginState != nil {
		row.OriginState = item.OriginState.Abbreviation
	}
	if item.DestinationState != nil {
		row.DestinationState = item.DestinationState.Abbreviation
	}
	if item.EquipmentType != nil {
		row.EquipmentType = item.EquipmentType.Code
	}
	if item.Rate.Valid {
		row.Rate = item.Rate.Decimal.String()
	}

	return row
}

func newListCarrierCapacityPostingsTool(
	repo repositories.CarrierCapacityPostingRepository,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_carrier_capacity_postings",
		entityPlural: "carrier capacity postings",
		summary: "List trucks carriers have said they have available: carrier, origin, " +
			"destination, equipment, the window and the rate asked. Filter availableTo on or " +
			"after today for offers still open.",
		searchTerms: []string{"carrier capacity", "available trucks", "truck postings"},
		resource:    permission.ResourceCarrierCapacityPosting,
		config:      querybuilder.GetFieldConfiguration((*carriercapacity.Posting)(nil)),
		fields: []listField{
			{Name: "carrierId", Kind: filterText},
			{Name: "originLocationId", Kind: filterText},
			{Name: "originStateId", Kind: filterText},
			{Name: "destinationStateId", Kind: filterText},
			{Name: "equipmentTypeId", Kind: filterText},
			{Name: "availableFrom", Kind: filterDate, Sortable: true},
			{Name: "availableTo", Kind: filterDate, Sortable: true},
			{Name: "truckCount", Kind: filterNumber},
			{
				Name:   "rateMethod",
				Kind:   filterEnum,
				Values: sliceutils.Strings(carriercapacity.RateMethods()),
			},
			{
				Name:   "source",
				Kind:   filterEnum,
				Values: sliceutils.Strings(carriercapacity.Sources()),
			},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListCarrierCapacityPostingsRequest{
				Filter: opts,
			})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *carriercapacity.Posting) any {
				return toCapacityPostingRow(item)
			}), nil
		},
	})
}
