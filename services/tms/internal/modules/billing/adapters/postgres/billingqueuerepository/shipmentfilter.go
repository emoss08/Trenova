package billingqueuerepository

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listcatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	shipmentRelation    = "shipment"
	shipmentFilterAlias = "bqi_filter_shp"
)

func shipmentFilters(filter *pagination.QueryOptions) (*pagination.QueryOptions, keel.Pred) {
	if filter == nil {
		return nil, nil
	}

	bq := billingqueuedb.BillingQueueItems
	rel, ok := listcatalog.For[billingqueue.BillingQueueItem](bq.Query().TableRef().Table).
		Relation(shipmentRelation)
	if !ok {
		return filter, nil
	}

	own := make([]domaintypes.FieldFilter, 0, len(filter.FieldFilters))
	var onShipment []domaintypes.FieldFilter
	for _, f := range filter.FieldFilters {
		head, field, nested := strings.Cut(f.Field, ".")
		if _, known := rel.Fields[field]; !nested || head != shipmentRelation || !known {
			own = append(own, f)
			continue
		}
		f.Field = field
		onShipment = append(onShipment, f)
	}
	if len(onShipment) == 0 {
		return filter, nil
	}

	narrowed := *filter
	narrowed.FieldFilters = own

	return &narrowed, shipmentMatches(onShipment)
}

func shipmentMatches(filters []domaintypes.FieldFilter) keel.Pred {
	bq := billingqueuedb.BillingQueueItems
	sh := shipmentdb.Shipments.As(shipmentFilterAlias)

	match := listquery.Filter(
		sh.Pick(keel.True()).Where(
			sh.ID.EqCol(bq.ShipmentID),
			sh.OrganizationID.EqCol(bq.OrganizationID),
			sh.BusinessUnitID.EqCol(bq.BusinessUnitID),
		),
		&pagination.QueryOptions{FieldFilters: filters},
	)

	return keel.BoolOf(keel.Scalar(match)).IsTrue()
}
