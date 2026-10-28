package shipmentboardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
)

func watchlistToModel(w *shipmentboardports.ShipmentWatchlist) *gqlmodel.ShipmentWatchlist {
	return &gqlmodel.ShipmentWatchlist{
		Deliveries: deliveryWatchToModel(&w.Deliveries),
		Uncovered:  uncoveredWatchToModel(&w.Uncovered),
		Detention:  detentionWatchToModel(&w.Detention),
		Billing:    billingWatchToModel(&w.Billing),
	}
}

func deliveryWatchToModel(
	w *shipmentboardports.ShipmentDeliveryWatch,
) *gqlmodel.ShipmentDeliveryWatch {
	buckets := make([]*gqlmodel.DeliveryHourBucket, 0, len(w.Buckets))
	for _, bucket := range w.Buckets {
		buckets = append(buckets, &gqlmodel.DeliveryHourBucket{
			Hour:      bucket.Hour,
			Delivered: bucket.Delivered,
			Scheduled: bucket.Scheduled,
			Late:      bucket.Late,
		})
	}

	late := make([]*gqlmodel.LateDelivery, 0, len(w.WorstLate))
	for _, item := range w.WorstLate {
		late = append(late, &gqlmodel.LateDelivery{
			ShipmentID:   item.ShipmentID.String(),
			ProNumber:    gqlscalar.EmptyToNil(item.ProNumber),
			DeltaMinutes: item.DeltaMinutes,
			City:         item.City,
			CustomerName: item.CustomerName,
		})
	}

	return &gqlmodel.ShipmentDeliveryWatch{
		OnTime:    w.OnTime,
		Total:     w.Total,
		LateCount: w.LateCount,
		Buckets:   buckets,
		WorstLate: late,
	}
}

func uncoveredWatchToModel(
	w *shipmentboardports.ShipmentUncoveredWatch,
) *gqlmodel.ShipmentUncoveredWatch {
	windows := make([]*gqlmodel.UncoveredWindowSummary, 0, len(w.Windows))
	for _, window := range w.Windows {
		windows = append(windows, &gqlmodel.UncoveredWindowSummary{
			Window:       window.Window,
			StartMinutes: window.StartMinutes,
			EndMinutes:   window.EndMinutes,
			Count:        window.Count,
			Revenue:      gqldecimal.DecimalString(window.Revenue),
		})
	}

	out := &gqlmodel.ShipmentUncoveredWatch{
		Count:   w.Count,
		Revenue: gqldecimal.DecimalString(w.Revenue),
		Windows: windows,
	}
	if w.Next != nil {
		out.Next = &gqlmodel.NextUncoveredPickup{
			ShipmentID:      w.Next.ShipmentID.String(),
			PickupAt:        int(w.Next.PickupAt),
			OriginCity:      w.Next.OriginCity,
			DestinationCity: w.Next.DestinationCity,
		}
	}

	return out
}

func detentionWatchToModel(
	w *shipmentboardports.ShipmentDetentionWatch,
) *gqlmodel.ShipmentDetentionWatch {
	top := make([]*gqlmodel.DetentionAccrual, 0, len(w.Top))
	for _, item := range w.Top {
		top = append(top, &gqlmodel.DetentionAccrual{
			ShipmentID:    item.ShipmentID.String(),
			StopID:        item.StopID.String(),
			OccurrenceID:  gqlid.IDPtrFromPtr(item.OccurrenceID),
			FacilityName:  item.FacilityName,
			CoverageName:  gqlscalar.EmptyToNil(item.CoverageName),
			BillableSince: int(item.BillableSince),
			RatePerHour:   gqldecimal.DecimalString(item.RatePerHour),
			Amount:        gqldecimal.DecimalString(item.Amount),
		})
	}

	return &gqlmodel.ShipmentDetentionWatch{
		StopCount:   w.StopCount,
		Amount:      gqldecimal.DecimalString(w.Amount),
		RatePerHour: gqldecimal.DecimalString(w.RatePerHour),
		SnapshotAt:  int(w.SnapshotAt),
		Top:         top,
	}
}

func billingWatchToModel(
	w *shipmentboardports.ShipmentBillingWatch,
) *gqlmodel.ShipmentBillingWatch {
	customers := make([]*gqlmodel.ReadyToBillCustomer, 0, len(w.Customers))
	for _, customer := range w.Customers {
		customers = append(customers, &gqlmodel.ReadyToBillCustomer{
			CustomerID: customer.CustomerID.String(),
			Name:       customer.Name,
			Count:      customer.Count,
			Total:      gqldecimal.DecimalString(customer.Total),
		})
	}

	return &gqlmodel.ShipmentBillingWatch{
		Count:         w.Count,
		Total:         gqldecimal.DecimalString(w.Total),
		Customers:     customers,
		MoreCustomers: w.MoreCustomers,
	}
}
