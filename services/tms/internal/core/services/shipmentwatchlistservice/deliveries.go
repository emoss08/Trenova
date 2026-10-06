package shipmentwatchlistservice

import (
	"cmp"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	FirstBucketHour  = 6
	LastBucketHour   = 23
	worstLateLimit   = 3
	secondsPerMinute = 60
)

type deliveryState int

const (
	deliveryScheduled deliveryState = iota
	deliveryDelivered
	deliveryLate
)

func classifyDelivery(
	row *repositories.ShipmentDeliveryRow,
	now int64,
) (state deliveryState, lateMinutes int) {
	if row.ActualArrival != nil {
		if *row.ActualArrival > row.Cutoff {
			return deliveryLate, int((*row.ActualArrival - row.Cutoff) / secondsPerMinute)
		}
		return deliveryDelivered, 0
	}

	if now > row.Cutoff {
		return deliveryLate, int((now - row.Cutoff) / secondsPerMinute)
	}
	if row.StageRank == shipment.StageLate.Rank() {
		return deliveryLate, 0
	}

	return deliveryScheduled, 0
}

func BuildDeliveryWatch(
	rows []*repositories.ShipmentDeliveryRow,
	now int64,
	loc *time.Location,
) services.ShipmentDeliveryWatch {
	buckets := make([]services.DeliveryHourBucket, 0, LastBucketHour-FirstBucketHour+1)
	for hour := FirstBucketHour; hour <= LastBucketHour; hour++ {
		buckets = append(buckets, services.DeliveryHourBucket{Hour: hour})
	}

	watch := services.ShipmentDeliveryWatch{Total: len(rows), Buckets: buckets}
	late := make([]services.LateDelivery, 0, len(rows))

	for _, row := range rows {
		state, delta := classifyDelivery(row, now)
		hour := time.Unix(row.DeliveryAt, 0).In(loc).Hour()
		var bucket *services.DeliveryHourBucket
		if hour >= FirstBucketHour && hour <= LastBucketHour {
			bucket = &watch.Buckets[hour-FirstBucketHour]
		}

		switch state {
		case deliveryDelivered:
			watch.OnTime++
			if bucket != nil {
				bucket.Delivered++
			}
		case deliveryLate:
			watch.LateCount++
			if bucket != nil {
				bucket.Late++
			}
			late = append(late, services.LateDelivery{
				ShipmentID:   row.ShipmentID,
				ProNumber:    row.ProNumber,
				DeltaMinutes: delta,
				City:         row.City,
				CustomerName: row.CustomerName,
			})
		case deliveryScheduled:
			if bucket != nil {
				bucket.Scheduled++
			}
		}
	}

	slices.SortStableFunc(late, func(a, b services.LateDelivery) int {
		return cmp.Compare(b.DeltaMinutes, a.DeltaMinutes)
	})
	watch.WorstLate = late[:min(len(late), worstLateLimit)]

	return watch
}
