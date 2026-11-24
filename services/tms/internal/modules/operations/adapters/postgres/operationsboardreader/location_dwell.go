package operationsboardreader

import (
	"context"
	"math"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionboard"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardwording"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Dwell struct {
	intelligence detentionports.DetentionIntelligenceRepository
}

func NewDwell(p DetentionParams) *Dwell { return &Dwell{intelligence: p.Intelligence} }

func (*Dwell) Kind() recordkind.Kind { return recordkind.Kind(permission.ResourceLocation) }

func (*Dwell) Measure() string { return detentionboard.MeasureDwell }

func (r *Dwell) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}
	ask := &detentionAsk{
		tenant:    req.Tenant,
		period:    boardcatalog.PeriodThisQuarter,
		now:       req.Now,
		location:  req.Location,
		locations: req.IDs,
	}
	window, ok := ask.window()
	if !ok {
		return map[pulid.ID]boardreading.Reading{}, nil
	}
	facilities, err := r.intelligence.FacilityFigures(ctx, window.req)
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardreading.Reading, len(req.IDs))
	for _, id := range req.IDs {
		out[id] = quietFacility(ctx)
	}
	for idx := range facilities {
		facility := &facilities[idx]
		facility.WithholdFields(req.Visible)
		if _, asked := out[facility.LocationID]; asked {
			out[facility.LocationID] = dwell(ctx, facility)
		}
	}

	return out, nil
}

func quietFacility(ctx context.Context) boardreading.Reading {
	value := boardcatalog.Duration(0)

	return boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: slot, Value: &value},
		Fact:     &boardreading.Words{Text: i18n.T(ctx, "No stops this quarter")},
		Spread:   &boardreading.SpreadFacts{},
	}
}

func dwell(
	ctx context.Context,
	facility *detentionports.FacilityIntelligence,
) boardreading.Reading {
	value := boardcatalog.Duration(facility.PastFreeMinutes * stayMinute)

	return boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: slot, Value: &value},
		Fact: &boardreading.Words{Text: i18n.T(
			ctx,
			"{0, plural, one {# stop} other {# stops}}, median {1}, one in ten over {2}",
			facility.StopCount,
			boardwording.FormatDuration(minutesInSeconds(facility.MedianDwellMinutes)),
			boardwording.FormatDuration(minutesInSeconds(facility.P90DwellMinutes)),
		)},
		Spread: &boardreading.SpreadFacts{
			Allowance: decimal.NewFromFloat(facility.MedianFreeMinutes),
			Median:    decimal.NewFromFloat(facility.MedianDwellMinutes),
			Tail:      decimal.NewFromFloat(facility.P90DwellMinutes),
		},
	}
}

func minutesInSeconds(minutes float64) int64 {
	return int64(math.Round(minutes * stayMinute))
}
