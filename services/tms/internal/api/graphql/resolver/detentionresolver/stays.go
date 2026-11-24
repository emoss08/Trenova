package detentionresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func detentionPeriod(period gqlmodel.DetentionPeriod) (boardcatalog.Period, error) {
	switch period {
	case gqlmodel.DetentionPeriodToday:
		return boardcatalog.PeriodToday, nil
	case gqlmodel.DetentionPeriodYesterday:
		return boardcatalog.PeriodYesterday, nil
	case gqlmodel.DetentionPeriodThisWeek:
		return boardcatalog.PeriodThisWeek, nil
	case gqlmodel.DetentionPeriodLastWeek:
		return boardcatalog.PeriodLastWeek, nil
	case gqlmodel.DetentionPeriodThisMonth:
		return boardcatalog.PeriodThisMonth, nil
	case gqlmodel.DetentionPeriodLastMonth:
		return boardcatalog.PeriodLastMonth, nil
	case gqlmodel.DetentionPeriodThisQuarter:
		return boardcatalog.PeriodThisQuarter, nil
	case gqlmodel.DetentionPeriodLastQuarter:
		return boardcatalog.PeriodLastQuarter, nil
	}

	return "", errortypes.NewValidationError("period", errortypes.ErrInvalid, "Unknown period")
}

func detentionStayStage(stage *gqlmodel.DetentionStayStage) detentionports.StayStage {
	if stage == nil {
		return detentionports.StayStageAll
	}

	return detentionports.StayStage(*stage)
}

func detentionStayToModel(stay *detentionreview.Stay) *gqlmodel.DetentionStay {
	model := &gqlmodel.DetentionStay{
		Occurrence:      detentionOccurrenceToModel(stay.Occurrence),
		MinutesOnSite:   int(stay.MinutesOnSite),
		MinutesPastFree: int(stay.MinutesPastFree),
		BillableAmount:  stay.BillableAmount.String(),
	}
	if stay.HoldBack != detention.HoldBackNone {
		holdBack := gqlmodel.DetentionHoldBack(stay.HoldBack)
		model.HoldBack = &holdBack
	}

	return model
}

func detentionStaysToModel(list *detentionreview.StayList) *gqlmodel.DetentionStayList {
	items := make([]*gqlmodel.DetentionStay, 0, len(list.Items))
	for _, stay := range list.Items {
		items = append(items, detentionStayToModel(stay))
	}

	return &gqlmodel.DetentionStayList{Items: items, TotalCount: list.Total}
}

func detentionFacilityWaitsToModel(
	facilities []detentionports.FacilityIntelligence,
) []*gqlmodel.DetentionFacilityWait {
	out := make([]*gqlmodel.DetentionFacilityWait, 0, len(facilities))
	for idx := range facilities {
		facility := &facilities[idx]
		out = append(out, &gqlmodel.DetentionFacilityWait{
			ID:                 facility.LocationID.String(),
			Name:               facility.LocationName,
			City:               facility.LocationCity,
			State:              facility.LocationState,
			StopCount:          facility.StopCount,
			DetainedCount:      facility.DetainedCount,
			PastFreeMinutes:    int(facility.PastFreeMinutes),
			MedianFreeMinutes:  facility.MedianFreeMinutes,
			MedianDwellMinutes: facility.MedianDwellMinutes,
			P90DwellMinutes:    facility.P90DwellMinutes,
			BilledAmount:       facility.BilledAmount.String(),
		})
	}

	return out
}
