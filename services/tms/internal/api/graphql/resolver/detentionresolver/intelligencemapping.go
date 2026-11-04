package detentionresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionintelligence"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func detentionIntelligenceToModel(
	result *detentionintelligence.Intelligence,
) *gqlmodel.DetentionIntelligence {
	facilities := make([]*gqlmodel.DetentionFacilityIntelligence, 0, len(result.Facilities))
	for idx := range result.Facilities {
		facilities = append(facilities, detentionFacilityToModel(&result.Facilities[idx]))
	}
	customers := make([]*gqlmodel.DetentionCustomerIntelligence, 0, len(result.Customers))
	for idx := range result.Customers {
		customers = append(customers, detentionCustomerToModel(&result.Customers[idx]))
	}
	waivers := make([]*gqlmodel.DetentionWaiverLeakageStat, 0, len(result.Waivers))
	for idx := range result.Waivers {
		waivers = append(waivers, detentionWaiverStatToModel(&result.Waivers[idx]))
	}
	arrivals := make([]*gqlmodel.DetentionArrivalCell, 0, len(result.Arrivals))
	for _, cell := range result.Arrivals {
		arrivals = append(arrivals, &gqlmodel.DetentionArrivalCell{
			Weekday:       cell.Weekday,
			Hour:          cell.Hour,
			StopCount:     cell.StopCount,
			DetainedCount: cell.DetainedCount,
		})
	}
	weeks := make([]*gqlmodel.DetentionWeek, 0, len(result.Weeks))
	for _, week := range result.Weeks {
		weeks = append(weeks, &gqlmodel.DetentionWeek{
			WeekStart:       int(week.WeekStart),
			StopCount:       week.StopCount,
			DetainedCount:   week.DetainedCount,
			PastFreeMinutes: int(week.PastFreeMinutes),
		})
	}
	changes := make([]*gqlmodel.DetentionPolicyChange, 0, len(result.PolicyChanges))
	for _, change := range result.PolicyChanges {
		changes = append(changes, &gqlmodel.DetentionPolicyChange{
			LocationID:   change.LocationID.String(),
			LocationName: change.LocationName,
			PolicyID:     change.PolicyID.String(),
			PolicyName:   change.PolicyName,
			ChangedAt:    int(change.ChangedAt),
		})
	}
	steps := make([]*gqlmodel.DetentionNextStep, 0, len(result.NextSteps))
	for idx := range result.NextSteps {
		steps = append(steps, detentionNextStepToModel(&result.NextSteps[idx]))
	}

	return &gqlmodel.DetentionIntelligence{
		Days:          result.Days,
		Through:       result.Through,
		From:          int(result.From),
		To:            int(result.To),
		PreviousFrom:  int(result.PreviousFrom),
		PreviousTo:    int(result.PreviousTo),
		Timezone:      result.Timezone,
		Totals:        detentionTotalsToModel(&result.Totals),
		Previous:      detentionTotalsToModel(&result.Previous),
		Facilities:    facilities,
		Customers:     customers,
		Waivers:       waivers,
		Recovery:      detentionRecoveryToModel(&result.Recovery),
		Arrivals:      arrivals,
		Splits:        detentionSplitsToModel(&result.Splits),
		Weeks:         weeks,
		PolicyChanges: changes,
		Summary:       detentionParagraphsToModel(result.Summary),
		NextSteps:     steps,
		Truncated:     result.Truncated,
	}
}

func detentionTotalsToModel(
	t *detentionports.IntelligenceTotals,
) *gqlmodel.DetentionIntelligenceTotals {
	return &gqlmodel.DetentionIntelligenceTotals{
		StopCount:         t.StopCount,
		DetainedCount:     t.DetainedCount,
		PastFreeMinutes:   int(t.PastFreeMinutes),
		DockMinutes:       int(t.DockMinutes),
		OpenMinutes:       int(t.OpenMinutes),
		EarlyMinutes:      int(t.EarlyMinutes),
		LateMinutes:       int(t.LateMinutes),
		BilledAmount:      t.BilledAmount.String(),
		DriverPayAmount:   gqldecimal.NullDecimalStringPtr(t.DriverPayAmount),
		NetMargin:         gqldecimal.NullDecimalStringPtr(t.NetMargin),
		WaivedAmount:      t.WaivedAmount.String(),
		DisputeCount:      t.DisputeCount,
		SuppressedCount:   t.SuppressedCount,
		NoticeDueCount:    t.NoticeDueCount,
		NoticeOnTimeCount: t.NoticeOnTimeCount,
	}
}

func detentionFacilityToModel(
	f *detentionintelligence.Facility,
) *gqlmodel.DetentionFacilityIntelligence {
	var previous *int
	if f.PreviousPastFreeMinutes != nil {
		minutes := int(*f.PreviousPastFreeMinutes)
		previous = &minutes
	}

	return &gqlmodel.DetentionFacilityIntelligence{
		LocationID:              f.LocationID.String(),
		LocationName:            f.LocationName,
		LocationCity:            f.LocationCity,
		LocationState:           f.LocationState,
		StopCount:               f.StopCount,
		DetainedCount:           f.DetainedCount,
		PastFreeMinutes:         int(f.PastFreeMinutes),
		PreviousPastFreeMinutes: previous,
		DockMinutes:             int(f.DockMinutes),
		OpenMinutes:             int(f.OpenMinutes),
		EarlyMinutes:            int(f.EarlyMinutes),
		LateMinutes:             int(f.LateMinutes),
		AvgDwellMinutes:         f.AvgDwellMinutes,
		MedianDwellMinutes:      f.MedianDwellMinutes,
		P90DwellMinutes:         f.P90DwellMinutes,
		MedianFreeMinutes:       f.MedianFreeMinutes,
		BilledAmount:            f.BilledAmount.String(),
		DriverPayAmount:         gqldecimal.NullDecimalStringPtr(f.DriverPayAmount),
		NetMargin:               gqldecimal.NullDecimalStringPtr(f.NetMargin),
		WaivedAmount:            f.WaivedAmount.String(),
		DisputeCount:            f.DisputeCount,
		SuppressedCount:         f.SuppressedCount,
		NoticeDueCount:          f.NoticeDueCount,
		NoticeOnTimeCount:       f.NoticeOnTimeCount,
	}
}

func detentionCustomerToModel(
	c *detentionports.CustomerIntelligence,
) *gqlmodel.DetentionCustomerIntelligence {
	return &gqlmodel.DetentionCustomerIntelligence{
		CustomerID:        c.CustomerID.String(),
		CustomerName:      c.CustomerName,
		StopCount:         c.StopCount,
		DetainedCount:     c.DetainedCount,
		PastFreeMinutes:   int(c.PastFreeMinutes),
		BilledAmount:      c.BilledAmount.String(),
		DriverPayAmount:   gqldecimal.NullDecimalStringPtr(c.DriverPayAmount),
		NetMargin:         gqldecimal.NullDecimalStringPtr(c.NetMargin),
		WaivedAmount:      c.WaivedAmount.String(),
		DisputeCount:      c.DisputeCount,
		NoticeDueCount:    c.NoticeDueCount,
		NoticeOnTimeCount: c.NoticeOnTimeCount,
	}
}

func detentionRecoveryToModel(r *detentionintelligence.Recovery) *gqlmodel.DetentionRecovery {
	steps := make([]*gqlmodel.DetentionRecoveryStep, 0, len(r.Steps))
	for _, step := range r.Steps {
		steps = append(steps, &gqlmodel.DetentionRecoveryStep{
			Bucket:          gqlmodel.DetentionRecoveryBucket(step.Bucket),
			OccurrenceCount: step.OccurrenceCount,
			Amount:          step.Amount.String(),
		})
	}

	return &gqlmodel.DetentionRecovery{
		Accrued:   r.Accrued.String(),
		Steps:     steps,
		Billed:    r.Billed.String(),
		Invoiced:  r.Invoiced.String(),
		Collected: r.Collected.String(),
	}
}

func detentionSplitToModel(f detentionports.SplitFigure) *gqlmodel.DetentionSplitFigure {
	return &gqlmodel.DetentionSplitFigure{
		StopCount:       f.StopCount,
		DetainedCount:   f.DetainedCount,
		PastFreeMinutes: int(f.PastFreeMinutes),
	}
}

func detentionSplitsToModel(s *detentionports.Splits) *gqlmodel.DetentionStopSplits {
	return &gqlmodel.DetentionStopSplits{
		Pickup:      detentionSplitToModel(s.Pickup),
		Delivery:    detentionSplitToModel(s.Delivery),
		Appointment: detentionSplitToModel(s.Appointment),
		Open:        detentionSplitToModel(s.Open),
		Live:        detentionSplitToModel(s.Live),
		Drop:        detentionSplitToModel(s.Drop),
	}
}

func detentionSegmentsToModel(
	segments []detentionintelligence.Segment,
) []*gqlmodel.DetentionNarrativeSegment {
	out := make([]*gqlmodel.DetentionNarrativeSegment, 0, len(segments))
	for _, segment := range segments {
		model := &gqlmodel.DetentionNarrativeSegment{
			Text:   segment.Text,
			Strong: segment.Strong,
		}
		if segment.Tone != "" {
			tone := segment.Tone
			model.Tone = &tone
		}
		if segment.Target != "" {
			target := gqlmodel.DetentionSegmentTarget(segment.Target)
			id := segment.TargetID
			model.Target = &target
			model.TargetID = &id
		}
		out = append(out, model)
	}

	return out
}

func optionalIDString(id pulid.ID) *string {
	if id.IsNil() {
		return nil
	}
	value := id.String()
	return &value
}

func detentionNextStepToModel(step *detentionintelligence.NextStep) *gqlmodel.DetentionNextStep {
	model := &gqlmodel.DetentionNextStep{
		Kind:          gqlmodel.DetentionNextStepKind(step.Kind),
		Title:         detentionSegmentsToModel(step.Title),
		Detail:        step.Detail,
		ImpactCaption: step.ImpactCaption,
		LocationID:    optionalIDString(step.LocationID),
		CustomerID:    optionalIDString(step.CustomerID),
	}
	if step.ImpactMinutes != nil {
		minutes := int(*step.ImpactMinutes)
		model.ImpactMinutes = &minutes
	}
	if step.ImpactAmount != nil {
		amount := step.ImpactAmount.String()
		model.ImpactAmount = &amount
	}

	return model
}

func detentionParagraphsToModel(
	paragraphs [][]detentionintelligence.Segment,
) []*gqlmodel.DetentionNarrativeParagraph {
	out := make([]*gqlmodel.DetentionNarrativeParagraph, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		out = append(out, &gqlmodel.DetentionNarrativeParagraph{
			Segments: detentionSegmentsToModel(paragraph),
		})
	}

	return out
}
