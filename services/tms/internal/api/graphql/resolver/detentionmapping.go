package resolver

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func detentionJSONMap(v any) map[string]any {
	if v == nil {
		return nil
	}

	data, err := sonic.Marshal(v)
	if err != nil {
		return nil
	}

	var result map[string]any
	if err = sonic.Unmarshal(data, &result); err != nil {
		return nil
	}

	return result
}

func detentionTierToModel(entity *detention.DetentionPolicyTier) *gqlmodel.DetentionPolicyTier {
	if entity == nil {
		return nil
	}

	model := &gqlmodel.DetentionPolicyTier{
		FromMinute: int(entity.FromMinute),
		Rate:       entity.Rate.String(),
		RateUnit:   entity.RateUnit,
		Label:      entity.Label,
		SortOrder:  int(entity.SortOrder),
	}

	if !entity.ID.IsNil() {
		id := entity.ID.String()
		model.ID = &id
	}

	if entity.ToMinute != nil {
		to := int(*entity.ToMinute)
		model.ToMinute = &to
	}

	return model
}

func detentionPolicyToModel(entity *detention.DetentionPolicy) *gqlmodel.DetentionPolicy {
	if entity == nil {
		return nil
	}

	model := &gqlmodel.DetentionPolicy{
		ID:                          entity.ID.String(),
		BusinessUnitID:              entity.BusinessUnitID.String(),
		OrganizationID:              entity.OrganizationID.String(),
		Name:                        entity.Name,
		Code:                        entity.Code,
		Description:                 entity.Description,
		Status:                      entity.Status,
		IsOrgDefault:                entity.IsOrgDefault,
		Priority:                    int(entity.Priority),
		SpecificityScore:            int(entity.SpecificityScore),
		ShipmentTypeIds:             pulidsToStrings(entity.ShipmentTypeIDs),
		ServiceTypeIds:              pulidsToStrings(entity.ServiceTypeIDs),
		CommodityIds:                pulidsToStrings(entity.CommodityIDs),
		AppointmentStopsOnly:        entity.AppointmentStopsOnly,
		EffectiveStartDate:          int64PtrToIntPtr(entity.EffectiveStartDate),
		EffectiveEndDate:            int64PtrToIntPtr(entity.EffectiveEndDate),
		ClockStartBasis:             entity.ClockStartBasis,
		LateArrivalRule:             entity.LateArrivalRule,
		LateArrivalGraceMinutes:     int(entity.LateArrivalGraceMinutes),
		BillingFreeMinutes:          int(entity.BillingFreeMinutes),
		PickupFreeMinutes:           int32PtrToIntPtr(entity.PickupFreeMinutes),
		DeliveryFreeMinutes:         int32PtrToIntPtr(entity.DeliveryFreeMinutes),
		PayFreeMinutes:              int32PtrToIntPtr(entity.PayFreeMinutes),
		MinimumBillableMinutes:      int(entity.MinimumBillableMinutes),
		BillingIncrementMinutes:     int(entity.BillingIncrementMinutes),
		RoundingMode:                entity.RoundingMode,
		RateSource:                  entity.RateSource,
		AccessorialChargeID:         entity.AccessorialChargeID.String(),
		MaxBillableMinutesPerStop:   int32PtrToIntPtr(entity.MaxBillableMinutesPerStop),
		MaxChargePerStop:            nullDecimalToStringPtr(entity.MaxChargePerStop),
		MaxChargePerDay:             nullDecimalToStringPtr(entity.MaxChargePerDay),
		MaxChargePerShipment:        nullDecimalToStringPtr(entity.MaxChargePerShipment),
		DayBoundaryMode:             entity.DayBoundaryMode,
		ConvertToLayoverAtMinutes:   int32PtrToIntPtr(entity.ConvertToLayoverAtMinutes),
		LayoverAccessorialChargeID:  idPtrFromPulidPtr(entity.LayoverAccessorialChargeID),
		NotificationRequirement:     entity.NotificationRequirement,
		NotificationLeadMinutes:     int(entity.NotificationLeadMinutes),
		NotificationDeadlineMinutes: int(entity.NotificationDeadlineMinutes),
		UnnotifiedBehavior:          entity.UnnotifiedBehavior,
		AutoSendNotice:              entity.AutoSendNotice,
		AttachNoticePDF:             entity.AttachNoticePDF,
		SendDepartureSummary:        entity.SendDepartureSummary,
		RequireApprovalOverAmount:   nullDecimalToStringPtr(entity.RequireApprovalOverAmount),
		AutoApproveUnderAmount:      nullDecimalToStringPtr(entity.AutoApproveUnderAmount),
		Currency:                    entity.Currency,
		Comments:                    entity.Comments,
		Version:                     int(entity.Version),
		CreatedAt:                   int(entity.CreatedAt),
		UpdatedAt:                   int(entity.UpdatedAt),
		CustomerID:                  idPtrFromPulidPtr(entity.CustomerID),
		LocationID:                  idPtrFromPulidPtr(entity.LocationID),
	}

	// Both lists are emitted as empty rather than nil: a policy with no rate
	// ladder is a policy with zero tiers, not one with an unknown number, and a
	// null on the wire makes every consumer guard for it.
	model.StopTypes = make([]gqlmodel.StopType, 0, len(entity.StopTypes))
	for _, stopType := range entity.StopTypes {
		model.StopTypes = append(model.StopTypes, gqlmodel.StopType(stopType))
	}

	model.Tiers = make([]*gqlmodel.DetentionPolicyTier, 0, len(entity.Tiers))
	for _, tier := range entity.Tiers {
		if tier == nil {
			continue
		}
		model.Tiers = append(model.Tiers, detentionTierToModel(tier))
	}

	return model
}

func detentionPolicyConnectionToModel(
	result *pagination.CursorListResult[*detention.DetentionPolicy],
) (*gqlmodel.DetentionPolicyConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *detention.DetentionPolicy, cursor string) *gqlmodel.DetentionPolicyEdge {
			return &gqlmodel.DetentionPolicyEdge{
				Node:   detentionPolicyToModel(node),
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.DetentionPolicyEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DetentionPolicyConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func detentionPolicyFromInput(
	input *gqlmodel.DetentionPolicyInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*detention.DetentionPolicy, error) {
	accessorialChargeID, err := pulid.MustParse(input.AccessorialChargeID)
	if err != nil {
		return nil, invalidDetentionField("accessorialChargeId")
	}

	entity := &detention.DetentionPolicy{
		ID:                      id,
		OrganizationID:          tenantInfo.OrgID,
		BusinessUnitID:          tenantInfo.BuID,
		Name:                    input.Name,
		Code:                    input.Code,
		Status:                  detention.PolicyStatusDraft,
		ClockStartBasis:         detention.ClockStartBasisLaterOfArrivalOrAppointment,
		LateArrivalRule:         detention.LateArrivalRuleNoEffect,
		BillingFreeMinutes:      120,
		BillingIncrementMinutes: 15,
		RoundingMode:            detention.RoundingModeUp,
		RateSource:              detention.RateSourceAccessorial,
		AccessorialChargeID:     accessorialChargeID,
		DayBoundaryMode:         detention.CapScopePerStop,
		NotificationRequirement: detention.NotificationRequirementNone,
		NotificationLeadMinutes: 30,
		UnnotifiedBehavior:      detention.UnnotifiedBehaviorBill,
		Currency:                "USD",
	}

	if input.Description != nil {
		entity.Description = *input.Description
	}
	if input.Status != nil {
		entity.Status = *input.Status
	}
	if input.IsOrgDefault != nil {
		entity.IsOrgDefault = *input.IsOrgDefault
	}
	if input.Priority != nil {
		entity.Priority = intutils.SafeToInt16(*input.Priority)
	}

	if entity.CustomerID, err = pulidPtrFromStringPtr(
		input.CustomerID, "customerId",
	); err != nil {
		return nil, err
	}
	if entity.LocationID, err = pulidPtrFromStringPtr(
		input.LocationID, "locationId",
	); err != nil {
		return nil, err
	}
	if entity.ShipmentTypeIDs, err = pulidsFromStrings(
		input.ShipmentTypeIds, "shipmentTypeIds",
	); err != nil {
		return nil, err
	}
	if entity.ServiceTypeIDs, err = pulidsFromStrings(
		input.ServiceTypeIds, "serviceTypeIds",
	); err != nil {
		return nil, err
	}
	if entity.CommodityIDs, err = pulidsFromStrings(
		input.CommodityIds, "commodityIds",
	); err != nil {
		return nil, err
	}

	if len(input.StopTypes) > 0 {
		entity.StopTypes = make([]shipment.StopType, 0, len(input.StopTypes))
		for _, stopType := range input.StopTypes {
			entity.StopTypes = append(entity.StopTypes, shipment.StopType(stopType))
		}
	}

	if input.AppointmentStopsOnly != nil {
		entity.AppointmentStopsOnly = *input.AppointmentStopsOnly
	}
	entity.EffectiveStartDate = intPtrToInt64Ptr(input.EffectiveStartDate)
	entity.EffectiveEndDate = intPtrToInt64Ptr(input.EffectiveEndDate)

	if input.ClockStartBasis != nil {
		entity.ClockStartBasis = *input.ClockStartBasis
	}
	if input.LateArrivalRule != nil {
		entity.LateArrivalRule = *input.LateArrivalRule
	}
	if input.LateArrivalGraceMinutes != nil {
		entity.LateArrivalGraceMinutes = intutils.SafeToInt16(*input.LateArrivalGraceMinutes)
	}
	if input.BillingFreeMinutes != nil {
		entity.BillingFreeMinutes = intutils.SafeToInt32(*input.BillingFreeMinutes)
	}
	entity.PickupFreeMinutes = intPtrToInt32Ptr(input.PickupFreeMinutes)
	entity.DeliveryFreeMinutes = intPtrToInt32Ptr(input.DeliveryFreeMinutes)
	entity.PayFreeMinutes = intPtrToInt32Ptr(input.PayFreeMinutes)
	if input.MinimumBillableMinutes != nil {
		entity.MinimumBillableMinutes = intutils.SafeToInt32(*input.MinimumBillableMinutes)
	}
	if input.BillingIncrementMinutes != nil {
		entity.BillingIncrementMinutes = intutils.SafeToInt16(*input.BillingIncrementMinutes)
	}
	if input.RoundingMode != nil {
		entity.RoundingMode = *input.RoundingMode
	}
	if input.RateSource != nil {
		entity.RateSource = *input.RateSource
	}

	if entity.Tiers, err = detentionTiersFromInput(input.Tiers, tenantInfo); err != nil {
		return nil, err
	}

	entity.MaxBillableMinutesPerStop = intPtrToInt32Ptr(input.MaxBillableMinutesPerStop)
	if entity.MaxChargePerStop, err = nullDecimalFromStringPtr(
		input.MaxChargePerStop, "maxChargePerStop",
	); err != nil {
		return nil, err
	}
	if entity.MaxChargePerDay, err = nullDecimalFromStringPtr(
		input.MaxChargePerDay, "maxChargePerDay",
	); err != nil {
		return nil, err
	}
	if entity.MaxChargePerShipment, err = nullDecimalFromStringPtr(
		input.MaxChargePerShipment, "maxChargePerShipment",
	); err != nil {
		return nil, err
	}
	if input.DayBoundaryMode != nil {
		entity.DayBoundaryMode = *input.DayBoundaryMode
	}
	entity.ConvertToLayoverAtMinutes = intPtrToInt32Ptr(input.ConvertToLayoverAtMinutes)
	if entity.LayoverAccessorialChargeID, err = pulidPtrFromStringPtr(
		input.LayoverAccessorialChargeID, "layoverAccessorialChargeId",
	); err != nil {
		return nil, err
	}

	if input.NotificationRequirement != nil {
		entity.NotificationRequirement = *input.NotificationRequirement
	}
	if input.NotificationLeadMinutes != nil {
		entity.NotificationLeadMinutes = intutils.SafeToInt16(*input.NotificationLeadMinutes)
	}
	if input.NotificationDeadlineMinutes != nil {
		entity.NotificationDeadlineMinutes = intutils.SafeToInt16(
			*input.NotificationDeadlineMinutes,
		)
	}
	if input.UnnotifiedBehavior != nil {
		entity.UnnotifiedBehavior = *input.UnnotifiedBehavior
	}
	if input.AutoSendNotice != nil {
		entity.AutoSendNotice = *input.AutoSendNotice
	}
	if input.AttachNoticePDF != nil {
		entity.AttachNoticePDF = *input.AttachNoticePDF
	}
	if input.SendDepartureSummary != nil {
		entity.SendDepartureSummary = *input.SendDepartureSummary
	}
	if entity.RequireApprovalOverAmount, err = nullDecimalFromStringPtr(
		input.RequireApprovalOverAmount, "requireApprovalOverAmount",
	); err != nil {
		return nil, err
	}
	if entity.AutoApproveUnderAmount, err = nullDecimalFromStringPtr(
		input.AutoApproveUnderAmount, "autoApproveUnderAmount",
	); err != nil {
		return nil, err
	}
	if input.Currency != nil && *input.Currency != "" {
		entity.Currency = *input.Currency
	}
	if input.Comments != nil {
		entity.Comments = *input.Comments
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return entity, nil
}

func detentionTiersFromInput(
	inputs []*gqlmodel.DetentionTierInput,
	tenantInfo pagination.TenantInfo,
) ([]*detention.DetentionPolicyTier, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	tiers := make([]*detention.DetentionPolicyTier, 0, len(inputs))
	for index, input := range inputs {
		if input == nil {
			continue
		}

		rate, err := decimalFromString(input.Rate, "tiers")
		if err != nil {
			return nil, err
		}

		tier := &detention.DetentionPolicyTier{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			FromMinute:     intutils.SafeToInt32(input.FromMinute),
			ToMinute:       intPtrToInt32Ptr(input.ToMinute),
			Rate:           rate,
			RateUnit:       detention.TierRateUnitHour,
			SortOrder:      int32(index),
		}

		if input.RateUnit != nil {
			tier.RateUnit = *input.RateUnit
		}
		if input.Label != nil {
			tier.Label = *input.Label
		}
		if input.SortOrder != nil {
			tier.SortOrder = intutils.SafeToInt32(*input.SortOrder)
		}

		tiers = append(tiers, tier)
	}

	return tiers, nil
}

func detentionOccurrenceToModel(
	entity *detention.DetentionOccurrence,
) *gqlmodel.DetentionOccurrence {
	if entity == nil {
		return nil
	}

	return &gqlmodel.DetentionOccurrence{
		ID:                  entity.ID.String(),
		BusinessUnitID:      entity.BusinessUnitID.String(),
		OrganizationID:      entity.OrganizationID.String(),
		ShipmentID:          entity.ShipmentID.String(),
		ShipmentMoveID:      entity.ShipmentMoveID.String(),
		StopID:              entity.StopID.String(),
		CustomerID:          entity.CustomerID.String(),
		LocationID:          entity.LocationID.String(),
		DetentionPolicyID:   idPtrFromPulidPtr(entity.DetentionPolicyID),
		PolicySnapshot:      detentionJSONMap(entity.PolicySnapshot),
		CalculationTrace:    detentionJSONMap(entity.CalculationTrace),
		StopType:            string(entity.StopType),
		ScheduleType:        string(entity.ScheduleType),
		AppointmentStart:    int64PtrToIntPtr(entity.AppointmentStart),
		AppointmentEnd:      int64PtrToIntPtr(entity.AppointmentEnd),
		ArrivedAt:           int64PtrToIntPtr(entity.ArrivedAt),
		DepartedAt:          int64PtrToIntPtr(entity.DepartedAt),
		ClockStartAt:        int(entity.ClockStartAt),
		ClockStopAt:         int64PtrToIntPtr(entity.ClockStopAt),
		FreeTimeExpiresAt:   int(entity.FreeTimeExpiresAt),
		NoticeDueAt:         int64PtrToIntPtr(entity.NoticeDueAt),
		NoticeDeadlineAt:    int64PtrToIntPtr(entity.NoticeDeadlineAt),
		IsOpen:              entity.IsOpen,
		ArrivedLate:         entity.ArrivedLate,
		LateByMinutes:       int(entity.LateByMinutes),
		FreeMinutesGranted:  int(entity.FreeMinutesGranted),
		RawDwellMinutes:     int(entity.RawDwellMinutes),
		BillableMinutes:     int(entity.BillableMinutes),
		RoundedMinutes:      int(entity.RoundedMinutes),
		BillableUnits:       entity.BillableUnits.String(),
		GrossAmount:         entity.GrossAmount.String(),
		BillableAmount:      entity.BillableAmount.String(),
		DriverPayMinutes:    int(entity.DriverPayMinutes),
		DriverPayAmount:     entity.DriverPayAmount.String(),
		NetMargin:           entity.NetMargin.String(),
		CapApplied:          entity.CapApplied,
		ConvertedToLayover:  entity.ConvertedToLayover,
		Currency:            entity.Currency,
		Status:              entity.Status,
		NotificationStatus:  entity.NotificationStatus,
		NoticeSentAt:        int64PtrToIntPtr(entity.NoticeSentAt),
		SuppressedByGate:    entity.SuppressedByGate,
		RequiresApproval:    entity.RequiresApproval,
		WaiverReason:        entity.WaiverReason,
		WaiverNote:          entity.WaiverNote,
		WaivedAt:            int64PtrToIntPtr(entity.WaivedAt),
		WaivedAmount:        entity.WaivedAmount.String(),
		DisputeNote:         entity.DisputeNote,
		DisputedAt:          int64PtrToIntPtr(entity.DisputedAt),
		CollectabilityScore: int(entity.CollectabilityScore),
		EvidenceHead:        entity.EvidenceHead,
		AdditionalChargeID:  idPtrFromPulidPtr(entity.AdditionalChargeID),
		LocationName:        entity.LocationName,
		CustomerName:        entity.CustomerName,
		ShipmentProNumber:   entity.ShipmentProNumber,
		Version:             int(entity.Version),
		CreatedAt:           int(entity.CreatedAt),
		UpdatedAt:           int(entity.UpdatedAt),
	}
}

func detentionOccurrencesToModel(
	entities []*detention.DetentionOccurrence,
) []*gqlmodel.DetentionOccurrence {
	result := make([]*gqlmodel.DetentionOccurrence, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		result = append(result, detentionOccurrenceToModel(entity))
	}
	return result
}

func detentionEvidenceToModel(
	entity *detention.DetentionEvidence,
) *gqlmodel.DetentionEvidence {
	if entity == nil {
		return nil
	}

	return &gqlmodel.DetentionEvidence{
		ID:                    entity.ID.String(),
		DetentionOccurrenceID: entity.DetentionOccurrenceID.String(),
		Sequence:              int(entity.Sequence),
		Kind:                  entity.Kind,
		Source:                entity.Source,
		Summary:               entity.Summary,
		ObservedAt:            int(entity.ObservedAt),
		RecordedAt:            int(entity.RecordedAt),
		RecordedByID:          idPtrFromPulidPtr(entity.RecordedByID),
		DocumentID:            idPtrFromPulidPtr(entity.DocumentID),
		Payload:               entity.Payload,
		PrevHash:              entity.PrevHash,
		Hash:                  entity.Hash,
		CreatedAt:             int(entity.CreatedAt),
	}
}

func detentionEvidenceListToModel(
	entities []*detention.DetentionEvidence,
) []*gqlmodel.DetentionEvidence {
	result := make([]*gqlmodel.DetentionEvidence, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		result = append(result, detentionEvidenceToModel(entity))
	}
	return result
}

func detentionNoticeToModel(entity *detention.DetentionNotice) *gqlmodel.DetentionNotice {
	if entity == nil {
		return nil
	}

	return &gqlmodel.DetentionNotice{
		ID:                    entity.ID.String(),
		DetentionOccurrenceID: entity.DetentionOccurrenceID.String(),
		ThreadKey:             entity.ThreadKey,
		Kind:                  entity.Kind,
		Channel:               entity.Channel,
		DeliveryStatus:        entity.DeliveryStatus,
		Recipients:            entity.Recipients,
		Subject:               entity.Subject,
		Body:                  entity.Body,
		ScheduledFor:          int(entity.ScheduledFor),
		SentAt:                int64PtrToIntPtr(entity.SentAt),
		DeliveredAt:           int64PtrToIntPtr(entity.DeliveredAt),
		OpenedAt:              int64PtrToIntPtr(entity.OpenedAt),
		FailedAt:              int64PtrToIntPtr(entity.FailedAt),
		FailureReason:         entity.FailureReason,
		SentByID:              idPtrFromPulidPtr(entity.SentByID),
		WasAutomatic:          entity.WasAutomatic,
		SatisfiesRequirement:  entity.SatisfiesRequirement,
		QuotedFreeMinutes:     int(entity.QuotedFreeMinutes),
		QuotedRate:            nullDecimalToStringPtr(entity.QuotedRate),
		QuotedAmount:          nullDecimalToStringPtr(entity.QuotedAmount),
		CreatedAt:             int(entity.CreatedAt),
	}
}

func detentionNoticesToModel(
	entities []*detention.DetentionNotice,
) []*gqlmodel.DetentionNotice {
	result := make([]*gqlmodel.DetentionNotice, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		result = append(result, detentionNoticeToModel(entity))
	}
	return result
}

func detentionCollectabilityToModel(
	assessment detention.CollectabilityAssessment,
) *gqlmodel.DetentionCollectability {
	factors := make([]*gqlmodel.DetentionScoreFactor, 0, len(assessment.Factors))
	for _, factor := range assessment.Factors {
		factors = append(factors, &gqlmodel.DetentionScoreFactor{
			Key:      factor.Key,
			Label:    factor.Label,
			Earned:   factor.Earned,
			Possible: factor.Possible,
			Detail:   factor.Detail,
			Remedy:   factor.Remedy,
		})
	}

	return &gqlmodel.DetentionCollectability{
		Score:      int(assessment.Score),
		Band:       assessment.Band,
		Factors:    factors,
		ChainValid: assessment.ChainValid,
		Summary:    assessment.Summary,
	}
}

func detentionDeskEntryToModel(
	entry *detentionservice.DeskEntry,
) *gqlmodel.DetentionDeskEntry {
	if entry == nil {
		return nil
	}

	model := &gqlmodel.DetentionDeskEntry{
		Occurrence:           detentionOccurrenceToModel(entry.Occurrence),
		MinutesUntilFreeEnds: int(entry.MinutesUntilFreeEnds),
		NoticeWindowOpen:     entry.NoticeWindowOpen,
		AmountAtRisk:         entry.AmountAtRisk.String(),
		Urgency:              gqlmodel.DetentionDeskUrgency(entry.Urgency),
	}

	if entry.MinutesUntilNoticeDue != nil {
		minutes := int(*entry.MinutesUntilNoticeDue)
		model.MinutesUntilNoticeDue = &minutes
	}

	return model
}

func detentionOccurrenceDetailToModel(
	detail *detentionservice.OccurrenceDetail,
) *gqlmodel.DetentionOccurrenceDetail {
	if detail == nil {
		return nil
	}

	return &gqlmodel.DetentionOccurrenceDetail{
		Occurrence:     detentionOccurrenceToModel(detail.Occurrence),
		Evidence:       detentionEvidenceListToModel(detail.Evidence),
		Notices:        detentionNoticesToModel(detail.Notices),
		Collectability: detentionCollectabilityToModel(detail.Collectability),
		Receipt:        detail.Receipt,
	}
}

func detentionDisputePacketToModel(
	packet *detentionservice.DisputePacket,
) *gqlmodel.DetentionDisputePacket {
	if packet == nil {
		return nil
	}

	return &gqlmodel.DetentionDisputePacket{
		Occurrence:     detentionOccurrenceToModel(packet.Occurrence),
		PolicySnapshot: detentionJSONMap(packet.Snapshot),
		Receipt:        packet.Receipt,
		Evidence:       detentionEvidenceListToModel(packet.Evidence),
		Notices:        detentionNoticesToModel(packet.Notices),
		Collectability: detentionCollectabilityToModel(packet.Collectability),
		ChainVerified:  packet.ChainVerified,
		GeneratedAt:    int(packet.GeneratedAt),
	}
}

func facilityDetentionStatToModel(
	stat *repositories.FacilityDetentionStat,
) *gqlmodel.FacilityDetentionStat {
	if stat == nil {
		return nil
	}

	return &gqlmodel.FacilityDetentionStat{
		LocationID:         stat.LocationID.String(),
		LocationName:       stat.LocationName,
		StopCount:          stat.StopCount,
		BreachCount:        stat.BreachCount,
		AvgDwellMinutes:    stat.AvgDwellMinutes,
		MedianDwellMinutes: stat.MedianDwellMinutes,
		P90DwellMinutes:    stat.P90DwellMinutes,
		BilledAmount:       stat.BilledAmount.String(),
		DriverPayAmount:    stat.DriverPayAmount.String(),
		NetMargin:          stat.NetMargin.String(),
		WaivedAmount:       stat.WaivedAmount.String(),
		DisputeCount:       stat.DisputeCount,
		SuppressedCount:    stat.SuppressedCount,
	}
}

func customerDetentionStatToModel(
	stat *repositories.CustomerDetentionStat,
) *gqlmodel.CustomerDetentionStat {
	if stat == nil {
		return nil
	}

	return &gqlmodel.CustomerDetentionStat{
		CustomerID:      stat.CustomerID.String(),
		CustomerName:    stat.CustomerName,
		StopCount:       stat.StopCount,
		BreachCount:     stat.BreachCount,
		BilledAmount:    stat.BilledAmount.String(),
		DriverPayAmount: stat.DriverPayAmount.String(),
		NetMargin:       stat.NetMargin.String(),
		WaivedAmount:    stat.WaivedAmount.String(),
		DisputeCount:    stat.DisputeCount,
		SuppressedCount: stat.SuppressedCount,
	}
}

func detentionWaiverStatToModel(
	stat *repositories.WaiverLeakageStat,
) *gqlmodel.DetentionWaiverLeakageStat {
	if stat == nil {
		return nil
	}

	return &gqlmodel.DetentionWaiverLeakageStat{
		Reason:        stat.Reason,
		WaiverCount:   stat.WaiverCount,
		ApproverCount: stat.ApproverCount,
		WaivedAmount:  stat.WaivedAmount.String(),
	}
}

func detentionBacktestBucketToModel(
	bucket detentionservice.BacktestBucket,
) *gqlmodel.DetentionBacktestBucket {
	return &gqlmodel.DetentionBacktestBucket{
		Key:             bucket.Key,
		Label:           bucket.Label,
		StopCount:       bucket.StopCount,
		BillableCount:   bucket.BillableCount,
		ProposedAmount:  bucket.ProposedAmount.String(),
		BaselineAmount:  bucket.BaselineAmount.String(),
		Delta:           bucket.Delta.String(),
		DriverPayAmount: bucket.DriverPayAmount.String(),
		NetMargin:       bucket.NetMargin.String(),
	}
}

func detentionBacktestResultToModel(
	result *detentionservice.BacktestResult,
) *gqlmodel.DetentionBacktestResult {
	if result == nil {
		return nil
	}

	byCustomer := make([]*gqlmodel.DetentionBacktestBucket, 0, len(result.ByCustomer))
	for _, bucket := range result.ByCustomer {
		byCustomer = append(byCustomer, detentionBacktestBucketToModel(bucket))
	}

	byFacility := make([]*gqlmodel.DetentionBacktestBucket, 0, len(result.ByFacility))
	for _, bucket := range result.ByFacility {
		byFacility = append(byFacility, detentionBacktestBucketToModel(bucket))
	}

	return &gqlmodel.DetentionBacktestResult{
		StopsEvaluated:      result.StopsEvaluated,
		StopsMatched:        result.StopsMatched,
		StopsBillable:       result.StopsBillable,
		StopsForfeited:      result.StopsForfeited,
		StopsSuppressed:     result.StopsSuppressed,
		ProposedRevenue:     result.ProposedRevenue.String(),
		BaselineRevenue:     result.BaselineRevenue.String(),
		RevenueDelta:        result.RevenueDelta.String(),
		ProposedDriverPay:   result.ProposedDriverPay.String(),
		ProposedNetMargin:   result.ProposedNetMargin.String(),
		NegativeMarginStops: result.NegativeMarginStops,
		ByCustomer:          byCustomer,
		ByFacility:          byFacility,
		From:                int(result.From),
		To:                  int(result.To),
		Truncated:           result.Truncated,
	}
}

func detentionPreviewResultToModel(
	result *detentionservice.PreviewResult,
) *gqlmodel.DetentionPreviewResult {
	if result == nil {
		return nil
	}

	return &gqlmodel.DetentionPreviewResult{
		PolicySnapshot:     detentionJSONMap(result.Snapshot),
		RawDwellMinutes:    int(result.RawDwellMinutes),
		FreeMinutesGranted: int(result.FreeMinutesGranted),
		BillableMinutes:    int(result.BillableMinutes),
		RoundedMinutes:     int(result.RoundedMinutes),
		BillableAmount:     result.BillableAmount.String(),
		GrossAmount:        result.GrossAmount.String(),
		DriverPayAmount:    result.DriverPayAmount.String(),
		NetMargin:          result.NetMargin.String(),
		ArrivedLate:        result.ArrivedLate,
		CapApplied:         result.CapApplied,
		Status:             result.Status,
		NotificationStatus: result.NotificationStatus,
		SuppressedByGate:   result.SuppressedByGate,
		CalculationTrace:   detentionJSONMap(result.Trace),
		Receipt:            result.Receipt,
	}
}

func detentionPreviewScenarioFromInput(
	input *gqlmodel.DetentionPreviewScenarioInput,
) detentionservice.PreviewScenario {
	return detentionservice.PreviewScenario{
		ArrivedAt:        int64(input.ArrivedAt),
		DepartedAt:       intPtrToInt64Ptr(input.DepartedAt),
		AppointmentStart: intPtrToInt64Ptr(input.AppointmentStart),
		AppointmentEnd:   intPtrToInt64Ptr(input.AppointmentEnd),
		StopType:         shipment.StopType(input.StopType),
		ScheduleType:     shipment.StopScheduleType(input.ScheduleType),
		NoticeSentAt:     intPtrToInt64Ptr(input.NoticeSentAt),
		DriverPayRate:    input.DriverPayRate,
	}
}

func intPtrToInt64Ptr(value *int) *int64 {
	if value == nil {
		return nil
	}
	v := int64(*value)
	return &v
}

func intPtrToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	v := intutils.SafeToInt32(*value)
	return &v
}

func int32PtrToIntPtr(value *int32) *int {
	if value == nil {
		return nil
	}
	v := int(*value)
	return &v
}

func pulidPtrFromStringPtr(value *string, field string) (*pulid.ID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}

	id, err := pulid.MustParse(*value)
	if err != nil {
		return nil, invalidDetentionField(field)
	}

	return &id, nil
}

func invalidDetentionField(field string) error {
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, "Invalid identifier")
}

func detentionStatsRequest(
	input *gqlmodel.DetentionStatsInput,
	tenantInfo pagination.TenantInfo,
) *repositories.DetentionStatsRequest {
	req := &repositories.DetentionStatsRequest{
		TenantInfo: tenantInfo,
		From:       int64(input.From),
		To:         int64(input.To),
	}

	if input.Limit != nil {
		req.Limit = *input.Limit
	}

	return req
}
