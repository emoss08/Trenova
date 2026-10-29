package shipmentmapping

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/tariffmapping"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// contractRateToModel renders what the rate agreements charged, for both the
// preview the billing panel shows and the account the re-rate dialog reads out.
func ContractRateToModel(
	application *shipmentserviceports.ContractRateApplication,
) (*gqlmodel.ShipmentContractRate, error) {
	if application == nil {
		return nil, nil //nolint:nilnil // no application maps to a null field
	}

	inputs, err := tariffmapping.TariffInputsToModel(application.Inputs)
	if err != nil {
		return nil, err
	}

	charges := make([]*gqlmodel.ShipmentContractRateCharge, 0, len(application.Charges))
	for i := range application.Charges {
		charge := &application.Charges[i]
		charges = append(charges, &gqlmodel.ShipmentContractRateCharge{
			AccessorialChargeID: charge.AccessorialChargeID.String(),
			Code:                charge.Code,
			Description:         charge.Description,
			Owner:               string(charge.Owner),
			Label:               charge.Label,
			Amount:              charge.Amount.String(),
			Quantity:            gqldecimal.NullDecimalStringPtr(charge.Quantity),
			Rate:                gqldecimal.NullDecimalStringPtr(charge.Rate),
			RateUnit:            charge.RateUnit,
		})
	}

	return &gqlmodel.ShipmentContractRate{
		Applied:               application.Applied,
		Outcome:               string(application.Outcome),
		AgreementID:           gqlid.IDPtrFromPtr(application.AgreementID),
		AgreementName:         application.AgreementName,
		RuleID:                gqlid.IDPtrFromPtr(application.RuleID),
		RuleLabel:             application.RuleLabel,
		FormulaTemplateID:     gqlid.IDPtrFromPtr(application.FormulaTemplateID),
		FormulaTemplateName:   application.FormulaTemplateName,
		TemplateVersion:       int(application.TemplateVersion),
		Inputs:                inputs,
		Refusal:               application.Refusal,
		FreightLines:          tariffmapping.FreightLinesToModel(application.FreightLines),
		FreightAmount:         application.FreightAmount.String(),
		OtherChargeAmount:     application.OtherChargeAmount.String(),
		TotalChargeAmount:     application.TotalChargeAmount.String(),
		PreviousFreightAmount: application.PreviousFreightAmount.String(),
		Charges:               charges,
		Explanation:           application.Explanation,
	}, nil
}

func ShipmentCommentToModel(
	entity *shipmentdomain.ShipmentComment,
) (*gqlmodel.ShipmentComment, error) {
	if entity == nil {
		return nil, nil
	}
	mentions := make([]*gqlmodel.ShipmentCommentMention, 0, len(entity.MentionedUsers))
	for _, mention := range entity.MentionedUsers {
		if mention == nil {
			continue
		}
		mentions = append(mentions, &gqlmodel.ShipmentCommentMention{
			ID:              mention.ID.String(),
			CommentID:       mention.CommentID.String(),
			MentionedUserID: mention.MentionedUserID.String(),
			OrganizationID:  gqlid.IDPtr(mention.OrganizationID),
			BusinessUnitID:  gqlid.IDPtr(mention.BusinessUnitID),
			ShipmentID:      gqlid.IDPtr(mention.ShipmentID),
			CreatedAt:       int(mention.CreatedAt),
			MentionedUser:   mention.MentionedUser,
		})
	}
	mentionedUserIDs := make([]string, 0, len(entity.MentionedUserIDs))
	for _, id := range entity.MentionedUserIDs {
		mentionedUserIDs = append(mentionedUserIDs, id.String())
	}
	acknowledgments := make(
		[]*gqlmodel.ShipmentCommentAcknowledgment,
		0,
		len(entity.Acknowledgments),
	)
	for _, ack := range entity.Acknowledgments {
		if ack == nil {
			continue
		}
		acknowledgments = append(acknowledgments, &gqlmodel.ShipmentCommentAcknowledgment{
			ID:             ack.ID.String(),
			CommentID:      ack.CommentID.String(),
			UserID:         ack.UserID.String(),
			AcknowledgedAt: int(ack.AcknowledgedAt),
			User:           ack.User,
		})
	}
	attachments := make([]*gqlmodel.ShipmentCommentAttachment, 0, len(entity.Attachments))
	for _, attachment := range entity.Attachments {
		if attachment == nil {
			continue
		}
		attachments = append(attachments, &gqlmodel.ShipmentCommentAttachment{
			DocumentID:   attachment.DocumentID.String(),
			FileName:     attachment.FileName,
			OriginalName: gqlscalar.StringPtrFromValue(attachment.OriginalName),
			FileSize:     int(attachment.FileSize),
			MimeType:     attachment.MimeType,
			PreviewURL:   gqlscalar.StringPtrFromValue(attachment.PreviewURL),
			DownloadURL:  attachment.DownloadURL,
			CreatedAt:    int(attachment.CreatedAt),
		})
	}
	return &gqlmodel.ShipmentComment{
		ID:                     entity.ID.String(),
		BusinessUnitID:         gqlid.IDPtr(entity.BusinessUnitID),
		OrganizationID:         gqlid.IDPtr(entity.OrganizationID),
		ShipmentID:             entity.ShipmentID.String(),
		UserID:                 gqlid.IDPtr(entity.UserID),
		ParentCommentID:        gqlid.IDPtrFromPulidPtr(entity.ParentCommentID),
		ReplyCount:             int(entity.ReplyCount),
		Comment:                entity.Comment,
		Body:                   entity.Body,
		Type:                   gqlmodel.ShipmentCommentType(entity.Type),
		Visibility:             gqlmodel.ShipmentCommentVisibility(entity.Visibility),
		Priority:               gqlmodel.ShipmentCommentPriority(entity.Priority),
		Source:                 gqlmodel.ShipmentCommentSource(entity.Source),
		Metadata:               entity.Metadata,
		EditedAt:               gqlscalar.IntPtr(entity.EditedAt),
		PinnedAt:               gqlscalar.IntPtr(entity.PinnedAt),
		PinnedByID:             gqlid.IDPtrFromPulidPtr(entity.PinnedByID),
		ResolvedAt:             gqlscalar.IntPtr(entity.ResolvedAt),
		ResolvedByID:           gqlid.IDPtrFromPulidPtr(entity.ResolvedByID),
		RequiresAcknowledgment: entity.RequiresAcknowledgment,
		DeletedAt:              gqlscalar.IntPtr(entity.DeletedAt),
		Version:                int(entity.Version),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
		MentionedUserIds:       mentionedUserIDs,
		User:                   entity.User,
		PinnedBy:               entity.PinnedBy,
		ResolvedBy:             entity.ResolvedBy,
		MentionedUsers:         mentions,
		Acknowledgments:        acknowledgments,
		Attachments:            attachments,
	}, nil
}

func ShipmentDistanceToModel(
	response *shipmentdistanceports.DistanceCalculationResponse,
) *gqlmodel.ShipmentDistanceResponse {
	moves := make([]*gqlmodel.ShipmentDistanceMoveResult, 0, len(response.Moves))
	for _, move := range response.Moves {
		moves = append(moves, &gqlmodel.ShipmentDistanceMoveResult{
			MoveID:              gqlid.IDPtr(move.MoveID),
			MoveIndex:           move.MoveIndex,
			Distance:            move.Distance,
			Source:              move.Source,
			Provider:            gqlscalar.StringPtrFromValue(move.Provider),
			RoutingType:         gqlscalar.StringPtrFromValue(move.RoutingType),
			DataVersion:         gqlscalar.StringPtrFromValue(move.DataVersion),
			DistanceUnits:       gqlscalar.StringPtrFromValue(move.DistanceUnits),
			DistanceProfileID:   gqlscalar.StringPtrFromValue(move.DistanceProfileID),
			DistanceProfileName: gqlscalar.StringPtrFromValue(move.DistanceProfileName),
			Warnings:            move.Warnings,
			CalculatedAt:        int(move.CalculatedAt),
		})
	}
	return &gqlmodel.ShipmentDistanceResponse{
		ShipmentID:    gqlid.IDPtr(response.ShipmentID),
		TotalDistance: response.TotalDistance,
		Moves:         moves,
	}
}

func ShipmentTotalsToModel(
	response *shipmentports.ShipmentTotalsResponse,
) *gqlmodel.ShipmentTotalsResponse {
	return &gqlmodel.ShipmentTotalsResponse{
		FreightChargeAmount: response.FreightChargeAmount.String(),
		OtherChargeAmount:   response.OtherChargeAmount.String(),
		TotalChargeAmount:   response.TotalChargeAmount.String(),
		FuelSurcharge: ShipmentAdditionalChargeToModel(
			response.FuelSurcharge,
		),
	}
}

func PreviousRatesToModel(
	result *pagination.ListResult[*shipmentports.PreviousRateSummary],
) *gqlmodel.ShipmentPreviousRatesResponse {
	items := make([]*gqlmodel.ShipmentPreviousRateSummary, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &gqlmodel.ShipmentPreviousRateSummary{
			ShipmentID:          item.ShipmentID.String(),
			ProNumber:           item.ProNumber,
			CustomerID:          item.CustomerID.String(),
			ServiceTypeID:       item.ServiceTypeID.String(),
			ShipmentTypeID:      item.ShipmentTypeID.String(),
			FormulaTemplateID:   gqlid.IDPtr(item.FormulaTemplateID),
			FreightChargeAmount: item.FreightChargeAmount.String(),
			OtherChargeAmount:   item.OtherChargeAmount.String(),
			TotalChargeAmount:   item.TotalChargeAmount.String(),
			Pieces:              gqlscalar.IntPtr(item.Pieces),
			Weight:              gqlscalar.IntPtr(item.Weight),
			CreatedAt:           int(item.CreatedAt),
		})
	}
	return &gqlmodel.ShipmentPreviousRatesResponse{
		Items:      items,
		Total:      result.Total,
		TotalExact: result.TotalExact,
	}
}

func LoadingOptimizationToModel(
	result *shipmentserviceports.LoadingOptimizationResult,
) *gqlmodel.ShipmentLoadingOptimizationResponse {
	placements := make([]*gqlmodel.ShipmentLoadingCommodity, 0, len(result.Placements))
	for _, item := range result.Placements {
		placements = append(placements, &gqlmodel.ShipmentLoadingCommodity{
			CommodityID:         item.CommodityID.String(),
			CommodityName:       item.CommodityName,
			PositionFeet:        item.PositionFeet,
			LengthFeet:          item.LengthFeet,
			Weight:              int(item.Weight),
			Pieces:              int(item.Pieces),
			Stackable:           item.Stackable,
			Fragile:             item.Fragile,
			IsHazmat:            item.IsHazmat,
			HazmatClass:         gqlscalar.StringPtrFromValue(item.HazmatClass),
			MinTemp:             item.MinTemp,
			MaxTemp:             item.MaxTemp,
			LoadingInstructions: gqlscalar.StringPtrFromValue(item.LoadingInstructions),
			EstimatedLength:     item.EstimatedLength,
			StopNumber:          intPtrFromValue(item.StopNumber),
		})
	}
	hazmatZones := make([]*gqlmodel.ShipmentHazmatZone, 0, len(result.HazmatZones))
	for _, item := range result.HazmatZones {
		hazmatZones = append(hazmatZones, &gqlmodel.ShipmentHazmatZone{
			CommodityAId:         item.CommodityAID.String(),
			CommodityBId:         item.CommodityBID.String(),
			CommodityAName:       item.CommodityAName,
			CommodityBName:       item.CommodityBName,
			RuleName:             item.RuleName,
			SegregationType:      item.SegregationType,
			RequiredDistanceFeet: item.RequiredDistanceFeet,
			ActualDistanceFeet:   item.ActualDistanceFeet,
			Satisfied:            item.Satisfied,
		})
	}
	warnings := make([]*gqlmodel.ShipmentLoadingWarning, 0, len(result.Warnings))
	for _, item := range result.Warnings {
		warnings = append(warnings, &gqlmodel.ShipmentLoadingWarning{
			Type:         item.Type,
			Message:      item.Message,
			Severity:     item.Severity,
			CommodityIds: item.CommodityIDs,
		})
	}
	axleWeights := make([]*gqlmodel.ShipmentAxleWeight, 0, len(result.AxleWeights))
	for _, item := range result.AxleWeights {
		axleWeights = append(axleWeights, &gqlmodel.ShipmentAxleWeight{
			Axle:       item.Axle,
			Weight:     int(item.Weight),
			Limit:      int(item.Limit),
			Percentage: item.Percentage,
			Compliant:  item.Compliant,
		})
	}
	recommendations := make(
		[]*gqlmodel.ShipmentLoadingRecommendation,
		0,
		len(result.Recommendations),
	)
	for _, item := range result.Recommendations {
		recommendations = append(recommendations, &gqlmodel.ShipmentLoadingRecommendation{
			Type:         item.Type,
			Priority:     item.Priority,
			Title:        item.Title,
			Description:  item.Description,
			Impact:       gqlscalar.StringPtrFromValue(item.Impact),
			CommodityIds: item.CommodityIDs,
		})
	}
	stopDividers := make([]*gqlmodel.ShipmentStopDivider, 0, len(result.StopDividers))
	for _, item := range result.StopDividers {
		stopDividers = append(stopDividers, &gqlmodel.ShipmentStopDivider{
			PositionFeet: item.PositionFeet,
			StopNumber:   item.StopNumber,
			Label:        item.Label,
		})
	}
	return &gqlmodel.ShipmentLoadingOptimizationResponse{
		TrailerLengthFeet: result.TrailerLengthFeet,
		TotalLinearFeet:   result.TotalLinearFeet,
		TotalWeight:       int(result.TotalWeight),
		MaxWeight:         int(result.MaxWeight),
		LinearFeetUtil:    result.LinearFeetUtil,
		WeightUtil:        result.WeightUtil,
		UtilizationScore:  result.UtilizationScore,
		UtilizationGrade:  result.UtilizationGrade,
		Placements:        placements,
		HazmatZones:       hazmatZones,
		Warnings:          warnings,
		AxleWeights:       axleWeights,
		Recommendations:   recommendations,
		StopDividers:      stopDividers,
		AiAnalysis:        gqlscalar.StringPtrFromValue(result.AIAnalysis),
	}
}

func intPtrFromValue(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func ShipmentEtaToModel(eta *shipmenttrackingports.ShipmentEta) *gqlmodel.ShipmentEta {
	if eta == nil {
		return nil
	}

	out := &gqlmodel.ShipmentEta{
		Verdict:    eta.Verdict,
		InputStale: eta.InputStale,
		Stops:      eta.Stops,
	}
	if out.Stops == nil {
		out.Stops = []*stopeta.StopETA{}
	}
	if eta.StopID.IsNotNil() {
		stopID := eta.StopID.String()
		out.StopID = &stopID
	}
	if eta.Basis != "" {
		basis := eta.Basis
		out.Basis = &basis
	}
	if eta.ComputedAt > 0 {
		computedAt := int(eta.ComputedAt)
		out.ComputedAt = &computedAt
	}
	if eta.EstimatedArrival != nil {
		arrival := int(*eta.EstimatedArrival)
		out.EstimatedArrival = &arrival
	}
	if eta.SlackMinutes != nil {
		slack := int(*eta.SlackMinutes)
		out.SlackMinutes = &slack
	}
	if eta.Reason != "" {
		reason := eta.Reason
		out.Reason = &reason
	}

	return out
}
