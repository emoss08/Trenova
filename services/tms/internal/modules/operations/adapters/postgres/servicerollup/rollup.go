package servicerollup

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

const (
	DailyName         = "stop_service_daily"
	CustomerDailyName = "stop_service_customer_daily"

	Customer  = "customer_id"
	Delivered = "row_count"
	OnTime    = "on_time_stops"
	Late      = "late_stops"
)

var Daily = rollup.MustDefine(DailyName, shipmentdb.StopsSchema(),
	grain(),
	nil,
	measures(),
	options()...,
)

var CustomerDaily = rollup.MustDefine(CustomerDailyName, shipmentdb.StopsSchema(),
	grain(),
	rollup.Dims(shipmentdb.Stops.CustomerID),
	measures(),
	options()...,
)

var (
	DailyTable         = rollupread.Of(Daily)
	CustomerDailyTable = rollupread.Of(CustomerDaily)
)

func All() []*rollup.Rollup {
	return []*rollup.Rollup{Daily, CustomerDaily}
}

func DeliveredStop() keel.Pred {
	stp := shipmentdb.Stops

	return keel.And(
		stp.Status.Eq(shipmentvocab.StopStatusCompleted),
		stp.Type.In(shipmentvocab.DeliveryStopTypes()...),
		stp.ActualArrival.IsNotNull(),
		stp.ActualArrival.Gt(0),
		stp.ScheduledWindowStart.Gt(0),
	)
}

func OnTimeStop() keel.Pred {
	stp := shipmentdb.Stops

	return keel.Int64Of(stp.ActualArrival).
		LteExpr(stp.ScheduledWindowEnd.Coalesce(stp.ScheduledWindowStart))
}

func OrganizationZone() keel.StringExpr {
	return rollupread.OrganizationZone(shipmentdb.Stops.OrganizationID)
}

func grain() rollup.Grain {
	return rollup.Day(shipmentdb.Stops.ActualArrival).Epoch()
}

func measures() []rollup.Measure {
	return rollup.Measures(
		rollup.Count(),
		rollup.CountWhere(OnTime, OnTimeStop()),
		rollup.CountWhere(Late, keel.Not(OnTimeStop())),
	)
}

func options() []rollup.Option {
	return []rollup.Option{
		rollup.TenantZone(OrganizationZone()),
		rollup.Where(DeliveredStop()),
		rollup.WithoutSourceIndex(),
	}
}
