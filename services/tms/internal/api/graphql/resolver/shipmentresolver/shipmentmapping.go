package shipmentresolver

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// shipmentFormulaTemplateID parses the rating method, which a shipment being
// written for the first time is allowed not to have yet.
//
// A shipment carries either a rate agreement or a formula template, and which
// one it gets is settled by the rate engine after this mapping runs: the
// contract preview asks what an agreement would charge precisely because
// nothing has been chosen yet, and requiring one here refused that question
// before it could be asked. Whether the shipment ends up priced at all is the
// rate coverage rule's to answer, once the engine has spoken.
//
// An existing shipment is held to the stricter reading. Its rating method is a
// field a rater owns and nothing restores on save, so a payload that arrives
// without one — an older client, an integration echoing back the fields it
// cares about — would otherwise silently unprice a load somebody had rated.
func shipmentFormulaTemplateID(shipmentID pulid.ID, value string) (pulid.ID, error) {
	if shipmentID.IsNil() {
		return base.OptionalScopedID("formulaTemplateId", &value)
	}

	return base.RequiredID("formulaTemplateId", value)
}

func shipmentFromInput(
	input gqlmodel.ShipmentInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) (*shipmentdomain.Shipment, error) {
	serviceTypeID, err := base.RequiredID("serviceTypeId", input.ServiceTypeID)
	if err != nil {
		return nil, err
	}
	shipmentTypeID, err := base.RequiredID("shipmentTypeId", input.ShipmentTypeID)
	if err != nil {
		return nil, err
	}
	customerID, err := base.RequiredID("customerId", input.CustomerID)
	if err != nil {
		return nil, err
	}
	formulaTemplateID, err := shipmentFormulaTemplateID(id, input.FormulaTemplateID)
	if err != nil {
		return nil, err
	}
	tractorTypeID, err := base.OptionalID(input.TractorTypeID)
	if err != nil {
		return nil, err
	}
	trailerTypeID, err := base.OptionalID(input.TrailerTypeID)
	if err != nil {
		return nil, err
	}
	ownerID, err := base.OptionalID(input.OwnerID)
	if err != nil {
		return nil, err
	}
	enteredByID, err := base.OptionalID(input.EnteredByID)
	if err != nil {
		return nil, err
	}
	canceledByID, err := base.OptionalID(input.CanceledByID)
	if err != nil {
		return nil, err
	}
	consolidationGroupID, err := base.OptionalID(input.ConsolidationGroupID)
	if err != nil {
		return nil, err
	}
	orderID, err := base.OptionalID(input.OrderID)
	if err != nil {
		return nil, err
	}
	otherChargeAmount, err := nullDecimalFromInput(input.OtherChargeAmount)
	if err != nil {
		return nil, err
	}
	freightChargeAmount, err := nullDecimalFromInput(input.FreightChargeAmount)
	if err != nil {
		return nil, err
	}
	baseRate, err := nullDecimalFromInput(input.BaseRate)
	if err != nil {
		return nil, err
	}
	totalChargeAmount, err := nullDecimalFromInput(input.TotalChargeAmount)
	if err != nil {
		return nil, err
	}
	temperatureMin, err := int16PtrFromInput("temperatureMin", input.TemperatureMin)
	if err != nil {
		return nil, err
	}
	temperatureMax, err := int16PtrFromInput("temperatureMax", input.TemperatureMax)
	if err != nil {
		return nil, err
	}

	status := shipmentdomain.StatusNew
	if input.Status != nil {
		status = shipmentdomain.Status(*input.Status)
	}
	entryMethod := shipmentdomain.EntryMethodManual
	if input.EntryMethod != nil {
		entryMethod = shipmentdomain.EntryMethod(*input.EntryMethod)
	}

	entity := &shipmentdomain.Shipment{
		ID:                   id,
		BusinessUnitID:       authCtx.BusinessUnitID,
		OrganizationID:       authCtx.OrganizationID,
		ServiceTypeID:        serviceTypeID,
		ShipmentTypeID:       shipmentTypeID,
		CustomerID:           customerID,
		TractorTypeID:        tractorTypeID,
		TrailerTypeID:        trailerTypeID,
		OwnerID:              ownerID,
		EnteredByID:          enteredByID,
		CanceledByID:         canceledByID,
		FormulaTemplateID:    formulaTemplateID,
		ConsolidationGroupID: consolidationGroupID,
		OrderID:              orderID,
		Status:               status,
		EntryMethod:          entryMethod,
		ProNumber:            base.StringValue(input.ProNumber),
		BOL:                  base.StringValue(input.BOL),
		ExternalReference:    base.StringValue(input.ExternalReference),
		CancelReason:         base.StringValue(input.CancelReason),
		OtherChargeAmount:    otherChargeAmount,
		FreightChargeAmount:  freightChargeAmount,
		BaseRate:             baseRate,
		TotalChargeAmount:    totalChargeAmount,
		Pieces:               base.Int64Ptr(input.Pieces),
		Weight:               base.Int64Ptr(input.Weight),
		TemperatureMin:       temperatureMin,
		TemperatureMax:       temperatureMax,
		ActualDeliveryDate:   base.Int64Ptr(input.ActualDeliveryDate),
		ActualShipDate:       base.Int64Ptr(input.ActualShipDate),
		CanceledAt:           base.Int64Ptr(input.CanceledAt),
		BillingTransferStatus: shipmentdomain.BillingTransferStatus(
			base.StringValue(input.BillingTransferStatus),
		),
		TransferredToBillingAt: base.Int64Ptr(input.TransferredToBillingAt),
		MarkedReadyToBillAt:    base.Int64Ptr(input.MarkedReadyToBillAt),
		BilledAt:               base.Int64Ptr(input.BilledAt),
		RatingUnit:             base.Int64Value(input.RatingUnit),
		FuelSurchargeLocked:    base.BoolValue(input.FuelSurchargeLocked),
		RateOverrideReason:     base.StringValue(input.RateOverrideReason),
		SourceDocumentID:       base.StringValue(input.SourceDocumentID),
		FreightTerms:           shipmentdomain.FreightTermsPrepaid,
	}
	if input.FreightTerms != nil {
		entity.FreightTerms = *input.FreightTerms
	}
	billToCustomerID, err := base.OptionalScopedID("billToCustomerId", input.BillToCustomerID)
	if err != nil {
		return nil, err
	}
	if billToCustomerID.IsNotNil() {
		entity.BillToCustomerID = &billToCustomerID
	}
	if entity.RatingUnit == 0 {
		entity.RatingUnit = 1
	}
	if input.TenderStatus != nil {
		tenderStatus := shipmentdomain.TenderStatus(*input.TenderStatus)
		entity.TenderStatus = &tenderStatus
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	moves, err := shipmentMovesFromInput(input.Moves, authCtx)
	if err != nil {
		return nil, err
	}
	additionalCharges, err := shipmentAdditionalChargesFromInput(input.AdditionalCharges, authCtx)
	if err != nil {
		return nil, err
	}
	commodities, err := shipmentCommoditiesFromInput(input.Commodities, authCtx)
	if err != nil {
		return nil, err
	}
	allocations, err := shipmentChargeAllocationsFromInput(&input, authCtx)
	if err != nil {
		return nil, err
	}
	entity.Moves = moves
	entity.AdditionalCharges = additionalCharges
	entity.Commodities = commodities
	entity.ChargeAllocations = allocations

	return entity, nil
}

func shipmentMovesFromInput(
	inputs []*gqlmodel.ShipmentMoveInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.ShipmentMove, error) {
	moves := make([]*shipmentdomain.ShipmentMove, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		path := fmt.Sprintf("moves[%d]", idx)
		id, err := base.OptionalScopedID(path+".id", input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := base.OptionalScopedID(path+".shipmentId", input.ShipmentID)
		if err != nil {
			return nil, err
		}
		status := shipmentdomain.MoveStatusNew
		if input.Status != nil {
			status = shipmentdomain.MoveStatus(*input.Status)
		}
		loaded := true
		if input.Loaded != nil {
			loaded = *input.Loaded
		}
		move := &shipmentdomain.ShipmentMove{
			ID:                     id,
			BusinessUnitID:         authCtx.BusinessUnitID,
			OrganizationID:         authCtx.OrganizationID,
			ShipmentID:             shipmentID,
			Status:                 status,
			Loaded:                 loaded,
			Sequence:               base.Int64Value(input.Sequence),
			Distance:               input.Distance,
			DistanceSource:         base.StringValue(input.DistanceSource),
			DistanceProvider:       base.StringValue(input.DistanceProvider),
			DistanceCalculatedAt:   base.Int64Ptr(input.DistanceCalculatedAt),
			DistanceRouteSignature: base.StringValue(input.DistanceRouteSignature),
			DistanceDataVersion:    base.StringValue(input.DistanceDataVersion),
			DistanceRoutingType:    base.StringValue(input.DistanceRoutingType),
			DistanceUnits:          base.StringValue(input.DistanceUnits),
			DistanceMetadata:       input.DistanceMetadata,
		}
		if input.Version != nil {
			move.Version = int64(*input.Version)
		}
		stops, err := shipmentStopsFromInput(input.Stops, path, authCtx)
		if err != nil {
			return nil, err
		}
		move.Stops = stops
		moves = append(moves, move)
	}
	return moves, nil
}

func shipmentStopsFromInput(
	inputs []*gqlmodel.ShipmentStopInput,
	movePath string,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.Stop, error) {
	stops := make([]*shipmentdomain.Stop, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		path := fmt.Sprintf("%s.stops[%d]", movePath, idx)
		id, err := base.OptionalScopedID(path+".id", input.ID)
		if err != nil {
			return nil, err
		}
		shipmentMoveID, err := base.OptionalScopedID(path+".shipmentMoveId", input.ShipmentMoveID)
		if err != nil {
			return nil, err
		}
		locationID, err := base.RequiredID(path+".locationId", input.LocationID)
		if err != nil {
			return nil, err
		}
		status := shipmentdomain.StopStatusNew
		if input.Status != nil {
			status = shipmentdomain.StopStatus(*input.Status)
		}
		stopType := shipmentdomain.StopTypePickup
		if input.Type != nil {
			stopType = shipmentdomain.StopType(*input.Type)
		}
		scheduleType := shipmentdomain.StopScheduleTypeOpen
		if input.ScheduleType != nil {
			scheduleType = shipmentdomain.StopScheduleType(*input.ScheduleType)
		}
		stop := &shipmentdomain.Stop{
			ID:                     id,
			BusinessUnitID:         authCtx.BusinessUnitID,
			OrganizationID:         authCtx.OrganizationID,
			ShipmentMoveID:         shipmentMoveID,
			LocationID:             locationID,
			Status:                 status,
			Type:                   stopType,
			ScheduleType:           scheduleType,
			Sequence:               base.Int64Value(input.Sequence),
			Pieces:                 base.Int64Ptr(input.Pieces),
			Weight:                 base.Int64Ptr(input.Weight),
			ScheduledWindowStart:   base.Int64Value(input.ScheduledWindowStart),
			ScheduledWindowEnd:     base.Int64Ptr(input.ScheduledWindowEnd),
			ActualArrival:          base.Int64Ptr(input.ActualArrival),
			ActualDeparture:        base.Int64Ptr(input.ActualDeparture),
			CountLateOverride:      input.CountLateOverride,
			CountDetentionOverride: input.CountDetentionOverride,
			AddressLine:            base.StringValue(input.AddressLine),
		}
		if input.Version != nil {
			stop.Version = int64(*input.Version)
		}
		stops = append(stops, stop)
	}
	return stops, nil
}

func shipmentAdditionalChargesFromInput(
	inputs []*gqlmodel.ShipmentAdditionalChargeInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.AdditionalCharge, error) {
	charges := make([]*shipmentdomain.AdditionalCharge, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		path := fmt.Sprintf("additionalCharges[%d]", idx)
		id, err := base.OptionalScopedID(path+".id", input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := base.OptionalScopedID(path+".shipmentId", input.ShipmentID)
		if err != nil {
			return nil, err
		}
		accessorialChargeID, err := base.RequiredID(
			path+".accessorialChargeId",
			input.AccessorialChargeID,
		)
		if err != nil {
			return nil, err
		}
		amount, err := decimalFromInput(input.Amount)
		if err != nil {
			return nil, err
		}
		unit := int16(1)
		if input.Unit != nil {
			parsedUnit, err := int16FromInput("unit", *input.Unit)
			if err != nil {
				return nil, err
			}
			unit = parsedUnit
		}
		isSystemGenerated := false
		if input.IsSystemGenerated != nil {
			isSystemGenerated = *input.IsSystemGenerated
		}
		method := accessorialcharge.MethodFlat
		if input.Method != nil {
			method = accessorialcharge.Method(*input.Method)
		}
		fuelSurchargeProgramID, err := base.OptionalScopedID(
			path+".fuelSurchargeProgramId",
			input.FuelSurchargeProgramID,
		)
		if err != nil {
			return nil, err
		}
		charge := &shipmentdomain.AdditionalCharge{
			ID:                  id,
			BusinessUnitID:      authCtx.BusinessUnitID,
			OrganizationID:      authCtx.OrganizationID,
			ShipmentID:          shipmentID,
			AccessorialChargeID: accessorialChargeID,
			IsSystemGenerated:   isSystemGenerated,
			Method:              method,
			Amount:              amount,
			Unit:                unit,
		}
		if !fuelSurchargeProgramID.IsNil() {
			charge.FuelSurchargeProgramID = &fuelSurchargeProgramID
		}
		if input.Version != nil {
			charge.Version = int64(*input.Version)
		}
		charges = append(charges, charge)
	}
	return charges, nil
}

func shipmentCommoditiesFromInput(
	inputs []*gqlmodel.ShipmentCommodityInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.ShipmentCommodity, error) {
	commodities := make([]*shipmentdomain.ShipmentCommodity, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		path := fmt.Sprintf("commodities[%d]", idx)
		id, err := base.OptionalScopedID(path+".id", input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := base.OptionalScopedID(path+".shipmentId", input.ShipmentID)
		if err != nil {
			return nil, err
		}
		commodityID, err := base.RequiredID(path+".commodityId", input.CommodityID)
		if err != nil {
			return nil, err
		}
		commodity := &shipmentdomain.ShipmentCommodity{
			ID:             id,
			BusinessUnitID: authCtx.BusinessUnitID,
			OrganizationID: authCtx.OrganizationID,
			ShipmentID:     shipmentID,
			CommodityID:    commodityID,
			Pieces:         base.Int64Value(input.Pieces),
			Weight:         base.Int64Value(input.Weight),
			LengthFeet:     input.LengthFeet,
			WidthFeet:      input.WidthFeet,
			HeightFeet:     input.HeightFeet,
		}
		if commodity.Pieces == 0 {
			commodity.Pieces = 1
		}
		if input.Version != nil {
			commodity.Version = int64(*input.Version)
		}
		commodities = append(commodities, commodity)
	}
	return commodities, nil
}

// contractRateToModel renders what the rate agreements charged, for both the
// preview the billing panel shows and the account the re-rate dialog reads out.
func contractRateToModel(
	application *services.ContractRateApplication,
) *gqlmodel.ShipmentContractRate {
	if application == nil {
		return nil
	}

	accessorials := make(
		[]*gqlmodel.ShipmentContractRateAccessorial,
		0,
		len(application.Accessorials),
	)
	for _, accessorial := range application.Accessorials {
		accessorials = append(accessorials, &gqlmodel.ShipmentContractRateAccessorial{
			AccessorialChargeID: accessorial.AccessorialChargeID.String(),
			Description:         accessorial.Description,
			Method:              accessorial.Method,
			Amount:              accessorial.Amount.String(),
			Unit:                int(accessorial.Unit),
		})
	}

	return &gqlmodel.ShipmentContractRate{
		Applied:                application.Applied,
		Outcome:                string(application.Outcome),
		AgreementID:            base.IDPtrFromPtr(application.AgreementID),
		AgreementName:          application.AgreementName,
		RuleID:                 base.IDPtrFromPtr(application.RuleID),
		RuleLabel:              application.RuleLabel,
		FormulaTemplateID:      base.IDPtrFromPtr(application.FormulaTemplateID),
		FormulaTemplateName:    application.FormulaTemplateName,
		BaseRate:               base.NullDecimalStringPtr(application.BaseRate),
		LinehaulAmount:         application.LinehaulAmount.String(),
		OtherChargeAmount:      application.OtherChargeAmount.String(),
		TotalChargeAmount:      application.TotalChargeAmount.String(),
		PreviousLinehaulAmount: application.PreviousLinehaulAmount.String(),
		Accessorials:           accessorials,
		Explanation:            application.Explanation,
	}
}

func shipmentCommentToModel(
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
			OrganizationID:  base.IDPtr(mention.OrganizationID),
			BusinessUnitID:  base.IDPtr(mention.BusinessUnitID),
			ShipmentID:      base.IDPtr(mention.ShipmentID),
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
			OriginalName: base.StringPtrFromValue(attachment.OriginalName),
			FileSize:     int(attachment.FileSize),
			MimeType:     attachment.MimeType,
			PreviewURL:   base.StringPtrFromValue(attachment.PreviewURL),
			DownloadURL:  attachment.DownloadURL,
			CreatedAt:    int(attachment.CreatedAt),
		})
	}
	return &gqlmodel.ShipmentComment{
		ID:                     entity.ID.String(),
		BusinessUnitID:         base.IDPtr(entity.BusinessUnitID),
		OrganizationID:         base.IDPtr(entity.OrganizationID),
		ShipmentID:             entity.ShipmentID.String(),
		UserID:                 base.IDPtr(entity.UserID),
		ParentCommentID:        base.IDPtrFromPulidPtr(entity.ParentCommentID),
		ReplyCount:             int(entity.ReplyCount),
		Comment:                entity.Comment,
		Body:                   entity.Body,
		Type:                   gqlmodel.ShipmentCommentType(entity.Type),
		Visibility:             gqlmodel.ShipmentCommentVisibility(entity.Visibility),
		Priority:               gqlmodel.ShipmentCommentPriority(entity.Priority),
		Source:                 gqlmodel.ShipmentCommentSource(entity.Source),
		Metadata:               entity.Metadata,
		EditedAt:               base.IntPtr(entity.EditedAt),
		PinnedAt:               base.IntPtr(entity.PinnedAt),
		PinnedByID:             base.IDPtrFromPulidPtr(entity.PinnedByID),
		ResolvedAt:             base.IntPtr(entity.ResolvedAt),
		ResolvedByID:           base.IDPtrFromPulidPtr(entity.ResolvedByID),
		RequiresAcknowledgment: entity.RequiresAcknowledgment,
		DeletedAt:              base.IntPtr(entity.DeletedAt),
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

func (r *MutationResolver) toggleShipmentComment(
	ctx context.Context,
	shipmentID string,
	commentID string,
	op permission.Operation,
	action func(
		ctx context.Context,
		req *services.ToggleShipmentCommentRequest,
		actor *services.RequestActor,
	) (*shipmentdomain.ShipmentComment, error),
) (*gqlmodel.ShipmentComment, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipmentComment, op)
	if err != nil {
		return nil, err
	}

	parsedShipmentID, err := pulid.MustParse(shipmentID)
	if err != nil {
		return nil, err
	}
	parsedCommentID, err := pulid.MustParse(commentID)
	if err != nil {
		return nil, err
	}

	entity, err := action(ctx, &services.ToggleShipmentCommentRequest{
		TenantInfo: base.TenantInfo(authCtx),
		ShipmentID: parsedShipmentID,
		CommentID:  parsedCommentID,
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		return nil, err
	}

	return shipmentCommentToModel(entity)
}

func shipmentCommentFiltersFromGraphQL(
	filter *gqlmodel.ShipmentCommentsFilterInput,
) (repositories.ShipmentCommentListFilters, error) {
	filters := repositories.ShipmentCommentListFilters{}
	if filter == nil {
		return filters, nil
	}

	if len(filter.Types) > 0 {
		filters.Types = make([]shipmentdomain.CommentType, 0, len(filter.Types))
		for _, commentType := range filter.Types {
			filters.Types = append(filters.Types, shipmentdomain.CommentType(commentType))
		}
	}
	if len(filter.Priorities) > 0 {
		filters.Priorities = make([]shipmentdomain.CommentPriority, 0, len(filter.Priorities))
		for _, priority := range filter.Priorities {
			filters.Priorities = append(
				filters.Priorities,
				shipmentdomain.CommentPriority(priority),
			)
		}
	}
	if len(filter.AuthorIds) > 0 {
		authorIDs, err := base.ParseIDs(filter.AuthorIds)
		if err != nil {
			return filters, err
		}
		filters.AuthorIDs = authorIDs
	}

	mentionsUserID, err := base.PulidPtrFromOptionalString(filter.MentionsUserID)
	if err != nil {
		return filters, err
	}
	filters.MentionsUserID = mentionsUserID
	filters.PinnedOnly = base.BoolValue(filter.PinnedOnly)
	filters.UnresolvedOnly = base.BoolValue(filter.UnresolvedOnly)
	filters.Search = strings.TrimSpace(base.StringValue(filter.Search))

	return filters, nil
}

func shipmentDistanceToModel(
	response *services.DistanceCalculationResponse,
) *gqlmodel.ShipmentDistanceResponse {
	moves := make([]*gqlmodel.ShipmentDistanceMoveResult, 0, len(response.Moves))
	for _, move := range response.Moves {
		moves = append(moves, &gqlmodel.ShipmentDistanceMoveResult{
			MoveID:              base.IDPtr(move.MoveID),
			MoveIndex:           move.MoveIndex,
			Distance:            move.Distance,
			Source:              move.Source,
			Provider:            base.StringPtrFromValue(move.Provider),
			RoutingType:         base.StringPtrFromValue(move.RoutingType),
			DataVersion:         base.StringPtrFromValue(move.DataVersion),
			DistanceUnits:       base.StringPtrFromValue(move.DistanceUnits),
			DistanceProfileID:   base.StringPtrFromValue(move.DistanceProfileID),
			DistanceProfileName: base.StringPtrFromValue(move.DistanceProfileName),
			Warnings:            move.Warnings,
			CalculatedAt:        int(move.CalculatedAt),
		})
	}
	return &gqlmodel.ShipmentDistanceResponse{
		ShipmentID:    base.IDPtr(response.ShipmentID),
		TotalDistance: response.TotalDistance,
		Moves:         moves,
	}
}

func shipmentTotalsToModel(
	response *repositories.ShipmentTotalsResponse,
) *gqlmodel.ShipmentTotalsResponse {
	return &gqlmodel.ShipmentTotalsResponse{
		FreightChargeAmount: response.FreightChargeAmount.String(),
		OtherChargeAmount:   response.OtherChargeAmount.String(),
		TotalChargeAmount:   response.TotalChargeAmount.String(),
		FuelSurcharge:       base.ShipmentAdditionalChargeToModel(response.FuelSurcharge),
	}
}

func previousRatesToModel(
	result *pagination.ListResult[*repositories.PreviousRateSummary],
) *gqlmodel.ShipmentPreviousRatesResponse {
	items := make([]*gqlmodel.ShipmentPreviousRateSummary, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &gqlmodel.ShipmentPreviousRateSummary{
			ShipmentID:          item.ShipmentID.String(),
			ProNumber:           item.ProNumber,
			CustomerID:          item.CustomerID.String(),
			ServiceTypeID:       item.ServiceTypeID.String(),
			ShipmentTypeID:      item.ShipmentTypeID.String(),
			FormulaTemplateID:   item.FormulaTemplateID.String(),
			FreightChargeAmount: item.FreightChargeAmount.String(),
			OtherChargeAmount:   item.OtherChargeAmount.String(),
			TotalChargeAmount:   item.TotalChargeAmount.String(),
			RatingUnit:          int(item.RatingUnit),
			Pieces:              base.IntPtr(item.Pieces),
			Weight:              base.IntPtr(item.Weight),
			CreatedAt:           int(item.CreatedAt),
		})
	}
	return &gqlmodel.ShipmentPreviousRatesResponse{
		Items: items,
		Total: result.Total,
	}
}

func loadingOptimizationRequestFromInput(
	input gqlmodel.ShipmentLoadingOptimizationInput,
	authCtx *authctx.AuthContext,
) (*repositories.LoadingOptimizationRequest, error) {
	commodities := make([]repositories.LoadingCommodityInput, 0, len(input.Commodities))
	for _, commodity := range input.Commodities {
		if commodity == nil {
			continue
		}
		commodityID, err := pulid.MustParse(commodity.CommodityID)
		if err != nil {
			return nil, err
		}
		commodities = append(commodities, repositories.LoadingCommodityInput{
			CommodityID: commodityID,
			Pieces:      int64(commodity.Pieces),
			Weight:      int64(commodity.Weight),
		})
	}
	var equipmentTypeID *pulid.ID
	if input.EquipmentTypeID != nil && *input.EquipmentTypeID != "" {
		parsed, err := pulid.MustParse(*input.EquipmentTypeID)
		if err != nil {
			return nil, err
		}
		equipmentTypeID = &parsed
	}
	stops := make([]repositories.StopInfo, 0, len(input.Stops))
	for _, stop := range input.Stops {
		if stop == nil {
			continue
		}
		stops = append(stops, repositories.StopInfo{
			Sequence:     stop.Sequence,
			LocationName: stop.LocationName,
			LocationCity: stop.LocationCity,
		})
	}
	return &repositories.LoadingOptimizationRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		Commodities:     commodities,
		EquipmentTypeID: equipmentTypeID,
		Stops:           stops,
	}, nil
}

func loadingOptimizationToModel(
	result *repositories.LoadingOptimizationResult,
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
			HazmatClass:         base.StringPtrFromValue(item.HazmatClass),
			MinTemp:             item.MinTemp,
			MaxTemp:             item.MaxTemp,
			LoadingInstructions: base.StringPtrFromValue(item.LoadingInstructions),
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
			Impact:       base.StringPtrFromValue(item.Impact),
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
		AiAnalysis:        base.StringPtrFromValue(result.AIAnalysis),
	}
}

func nullDecimalFromInput(value *string) (decimal.NullDecimal, error) {
	parsed, err := decimalFromInput(value)
	if err != nil {
		return decimal.NullDecimal{}, err
	}
	return decimal.NewNullDecimal(parsed), nil
}

func decimalFromInput(value *string) (decimal.Decimal, error) {
	if value == nil || *value == "" {
		return decimal.Zero, nil
	}
	parsed, err := decimal.NewFromString(*value)
	if err != nil {
		return decimal.Decimal{}, errortypes.NewValidationError(
			"amount",
			errortypes.ErrInvalidFormat,
			"Amount must be a valid decimal",
		)
	}
	return parsed, nil
}

func int16PtrFromInput(field string, value *int) (*int16, error) {
	if value == nil {
		return nil, nil
	}
	converted, err := int16FromInput(field, *value)
	if err != nil {
		return nil, err
	}
	return &converted, nil
}

func int16FromInput(field string, value int) (int16, error) {
	if value < math.MinInt16 || value > math.MaxInt16 {
		return 0, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"{0} is outside the allowed range", field,
		)
	}
	return int16(value), nil
}

func optionalJSON(value any) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}
	return jsonutils.ToJSON(value)
}

func intPtrFromValue(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func (r *ShipmentResolver) loadShipmentOrder(
	ctx context.Context,
	obj *gqlmodel.Shipment,
) (*order.Order, error) {
	if obj == nil || obj.OrderID == nil || *obj.OrderID == "" {
		return nil, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Order loader is not configured")
	}

	parent, err := loadersForRequest.OrderByID.Load(ctx, *obj.OrderID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return parent, nil
}
