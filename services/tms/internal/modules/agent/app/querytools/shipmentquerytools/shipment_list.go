package shipmentquerytools

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/typeutils"
)

var (
	shipmentStatuses = []string{
		"New", "PartiallyAssigned", "Assigned", "InTransit", "Delayed",
		"PartiallyCompleted", "ReadyToInvoice", "Completed", "Invoiced", "Canceled",
	}
	billingTransferStates = []string{
		"ReadyForReview", "InReview", "OnHold", "Exception",
		"SentBackToOps", "Approved", "Posted", "Canceled",
	}
	freightTerms = []string{"Prepaid", "Collect", "ThirdParty"}
)

type Row struct {
	ID                 string                 `json:"id"`
	ProNumber          string                 `json:"proNumber"`
	BOL                string                 `json:"bol,omitempty"`
	Status             string                 `json:"status"`
	Customer           string                 `json:"customer,omitempty"`
	Pickup             string                 `json:"pickup,omitempty"`
	Delivery           string                 `json:"delivery,omitempty"`
	Drivers            string                 `json:"drivers,omitempty"`
	Carrying           string                 `json:"carrying,omitempty"`
	TotalCharge        string                 `json:"totalCharge,omitempty"`
	ActualShipDate     querytool.OptionalDate `json:"actualShipDate"`
	ActualDeliveryDate querytool.OptionalDate `json:"actualDeliveryDate"`
	BillingStatus      string                 `json:"billingStatus,omitempty"`
	ActiveHolds        []shipmentRowHold      `json:"activeHolds,omitempty"`
}

func ListSpec(repo shipmentports.ShipmentRepository) listtool.Spec {
	return listtool.Spec{
		Name:         querytool.ShipmentListName,
		Row:          Row{},
		EntityPlural: "shipments",
		Summary: "Shipments by status, billing state, dates or charges, such as picking up " +
			"today.",
		Resource: permission.ResourceShipment,
		Config:   listquery.Describe((*shipment.Shipment)(nil), shipmentdb.ShipmentsSchema()),
		Fields: []filtercatalog.Field{
			{
				Name:   "status",
				Kind:   filtercatalog.KindEnum,
				Values: shipmentStatuses,
				Note: "rolling is InTransit and Delayed, one in filter; no Delivered: a " +
					"delivered load is Completed, ReadyToInvoice, then Invoiced",
			},
			{
				Name:   "billingTransferStatus",
				Kind:   filtercatalog.KindEnum,
				Values: billingTransferStates,
				Note:   "isnull means never transferred",
			},
			{Name: "freightTerms", Kind: filtercatalog.KindEnum, Values: freightTerms},
			{
				Name: "customerId",
				Kind: filtercatalog.KindText,
				Note: "from customer_list, never guessed",
			},
			{Name: "proNumber", Kind: filtercatalog.KindText, Sortable: true},
			{Name: "bol", Kind: filtercatalog.KindText},
			{Name: "actualShipDate", Kind: filtercatalog.KindDate, Sortable: true},
			{Name: "actualDeliveryDate", Kind: filtercatalog.KindDate, Sortable: true},
			{Name: "billedAt", Kind: filtercatalog.KindDate, Sortable: true},
			{Name: "canceledAt", Kind: filtercatalog.KindDate},
			{Name: "totalChargeAmount", Kind: filtercatalog.KindNumber, Sortable: true},
			{Name: "weight", Kind: filtercatalog.KindNumber},
			{Name: "createdAt", Kind: filtercatalog.KindDate, Sortable: true},
		},
		PagesByCursor: true,
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, cursor pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &shipmentports.ListShipmentsRequest{
				Filter: opts,
				Cursor: cursor,
				ShipmentOptions: shipmentports.ShipmentOptions{
					IncludeCustomer:    true,
					IncludeRoute:       true,
					IncludeCommodities: true,
				},
			})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.CursorPage(result, func(item *shipment.Shipment) any {
				return ToRow(item)
			})
		},
		Count: func(ctx context.Context, opts *pagination.QueryOptions) (int, bool, error) {
			return countShipments(ctx, repo, opts, shipmentports.ShipmentOptions{})
		},
	}
}

var errNoTotal = errors.New("the shipment count was not returned")

func countShipments(
	ctx context.Context,
	repo shipmentports.ShipmentRepository,
	opts *pagination.QueryOptions,
	options shipmentports.ShipmentOptions,
) (int, bool, error) {
	probe := *opts
	probe.Pagination = pagination.Info{Limit: 1}
	probe.Sort = nil

	result, err := repo.List(ctx, &shipmentports.ListShipmentsRequest{
		Filter:          &probe,
		Cursor:          pagination.CursorInfo{Limit: 1, IncludeTotalCount: true},
		ShipmentOptions: options,
	})
	if err != nil {
		return 0, false, err
	}
	if result.TotalCount == nil {
		return 0, false, errNoTotal
	}

	return *result.TotalCount, result.TotalCountExact, nil
}

// ToRow is the one shipment projection, for the same reasons.
func ToRow(item *shipment.Shipment) Row {
	row := Row{
		ID:            item.ID.String(),
		ProNumber:     item.ProNumber,
		BOL:           item.BOL,
		Status:        string(item.Status),
		BillingStatus: string(item.BillingTransferStatus),
	}
	if item.TotalChargeAmount.Valid {
		row.TotalCharge = item.TotalChargeAmount.Decimal.String()
	}
	row.ActualShipDate = querytool.ExpectedDate(
		typeutils.ValueOrZero(item.ActualShipDate),
		"not shipped yet",
	)
	row.ActualDeliveryDate = querytool.ExpectedDate(
		typeutils.ValueOrZero(item.ActualDeliveryDate), "not delivered yet")
	if item.Customer != nil {
		row.Customer = item.Customer.Name
	}
	route := routeOf(item)
	row.Pickup = route.pickup
	row.Delivery = route.delivery
	row.Drivers = route.coverage
	row.Carrying = carriedCommodities(item.Commodities)

	return row
}

func carriedCommodities(commodities []*shipment.ShipmentCommodity) string {
	names := make([]string, 0, len(commodities))
	for _, line := range commodities {
		if line == nil || line.Commodity == nil {
			continue
		}
		if name := strings.TrimSpace(line.Commodity.Name); name != "" &&
			!slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	return strings.Join(names, ", ")
}
