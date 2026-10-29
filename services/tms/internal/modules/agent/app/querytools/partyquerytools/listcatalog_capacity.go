package partyquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity/carriercapacitydb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carriercapacityports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
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

func NewListCarrierCapacityPostingsTool(
	repo carriercapacityports.CarrierCapacityPostingRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         "carrier_list_capacity_postings",
		Row:          capacityPostingRow{},
		EntityPlural: "carrier capacity postings",
		Summary: "List trucks carriers have said they have available: carrier, origin, " +
			"destination, equipment, the window and the rate asked. Filter availableTo on or " +
			"after today for offers still open.",
		SearchTerms: []string{"carrier capacity", "available trucks", "truck postings"},
		Resource:    permission.ResourceCarrierCapacityPosting,
		Config: listquery.Describe(
			(*carriercapacity.Posting)(nil),
			carriercapacitydb.PostingsSchema(),
		),
		Fields: []filtercatalog.Field{
			{Name: "carrierId", Kind: filtercatalog.KindText},
			{Name: "originLocationId", Kind: filtercatalog.KindText},
			{Name: "originStateId", Kind: filtercatalog.KindText},
			{Name: "destinationStateId", Kind: filtercatalog.KindText},
			{Name: "equipmentTypeId", Kind: filtercatalog.KindText},
			{Name: "availableFrom", Kind: filtercatalog.KindDate, Sortable: true},
			{Name: "availableTo", Kind: filtercatalog.KindDate, Sortable: true},
			{Name: "truckCount", Kind: filtercatalog.KindNumber},
			{
				Name:   "rateMethod",
				Kind:   filtercatalog.KindEnum,
				Values: sliceutils.Strings(carriercapacity.RateMethods()),
			},
			{
				Name:   "source",
				Kind:   filtercatalog.KindEnum,
				Values: sliceutils.Strings(carriercapacity.Sources()),
			},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &carriercapacityports.ListCarrierCapacityPostingsRequest{
				Filter: opts,
			})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *carriercapacity.Posting) any {
				return toCapacityPostingRow(item)
			}), nil
		},
	})
}
