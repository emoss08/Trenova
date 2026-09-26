package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type MatchDecision string

const (
	MatchDecisionAccept             = MatchDecision("Accept")
	MatchDecisionAcceptWithVariance = MatchDecision("AcceptWithVariance")
	MatchDecisionReject             = MatchDecision("Reject")
)

func PlanAcceptMatch(
	match *carriersettlement.InvoiceMatch,
	note string,
	userID pulid.ID,
	now int64,
) error {
	if match.Status != carriersettlement.InvoiceMatchStatusMatched &&
		match.Status != carriersettlement.InvoiceMatchStatusSuggested {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only suggested or matched invoices can be accepted without a variance; use accept-with-variance instead",
		)
	}
	resolveMatch(match, carriersettlement.InvoiceMatchStatusResolved, note, userID, now)
	return nil
}

func CheckAcceptWithVariance(match *carriersettlement.InvoiceMatch) error {
	if match.Status != carriersettlement.InvoiceMatchStatusVariance {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only variance matches can be accepted with a variance adjustment",
		)
	}
	if match.VarianceMinor == 0 {
		return errortypes.NewValidationError(
			"varianceMinor",
			errortypes.ErrInvalid,
			"The match carries no variance to adjust",
		)
	}
	return nil
}

type VarianceResolution struct {
	Note         string
	UserID       pulid.ID
	ResolvedAt   int64
	AdjustmentID pulid.ID
}

func PlanAcceptWithVariance(
	match *carriersettlement.InvoiceMatch,
	resolution *VarianceResolution,
) error {
	if err := CheckAcceptWithVariance(match); err != nil {
		return err
	}
	resolveMatch(
		match,
		carriersettlement.InvoiceMatchStatusResolved,
		resolution.Note,
		resolution.UserID,
		resolution.ResolvedAt,
	)
	if resolution.AdjustmentID.IsNotNil() {
		adjustmentID := resolution.AdjustmentID
		match.AdjustmentCostEventID = &adjustmentID
	}
	return nil
}

func PlanRejectMatch(
	match *carriersettlement.InvoiceMatch,
	note string,
	userID pulid.ID,
	now int64,
) error {
	if note == "" {
		return errRejectionNoteRequired()
	}
	if !match.Status.IsOpen() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only open matches can be rejected",
		)
	}
	resolveMatch(match, carriersettlement.InvoiceMatchStatusRejected, note, userID, now)
	return nil
}

func resolveMatch(
	match *carriersettlement.InvoiceMatch,
	status carriersettlement.InvoiceMatchStatus,
	note string,
	userID pulid.ID,
	now int64,
) {
	match.Status = status
	match.ResolutionNote = note
	match.ResolvedByID = userID
	match.ResolvedAt = &now
}

func errRejectionNoteRequired() error {
	return errortypes.NewValidationError(
		"resolutionNote",
		errortypes.ErrRequired,
		"A rejection note is required",
	)
}

func varianceIdempotencyKey(match *carriersettlement.InvoiceMatch) string {
	return "carrier-invoice-variance:" + match.ID.String()
}

func VarianceAdjustmentEvent(
	tenantInfo pagination.TenantInfo,
	match *carriersettlement.InvoiceMatch,
	eventDate int64,
) *carriersettlement.CostEvent {
	description := "Invoice variance"
	if match.InvoiceNumber != "" {
		description += " (" + match.InvoiceNumber + ")"
	}
	assignmentID := match.CarrierAssignmentID
	event := &carriersettlement.CostEvent{
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		CarrierID:           match.CarrierID,
		CarrierAssignmentID: &assignmentID,
		EventType:           carriersettlement.CostEventTypeAdjustment,
		Status:              carriersettlement.CostEventStatusPending,
		IdempotencyKey:      varianceIdempotencyKey(match),
		EventDate:           eventDate,
		AmountMinor:         match.VarianceMinor,
		CurrencyCode:        match.CurrencyCode,
		Description:         description,
	}
	if match.CarrierAssignment != nil {
		moveID := match.CarrierAssignment.ShipmentMoveID
		event.MoveID = &moveID
		if match.CarrierAssignment.ShipmentMove != nil {
			spID := match.CarrierAssignment.ShipmentMove.ShipmentID
			event.ShipmentID = &spID
		}
	}
	return event
}

type MatchDecisionRequest struct {
	TenantInfo pagination.TenantInfo
	MatchID    pulid.ID
	Decision   MatchDecision
	Note       string
}

type MatchDecisionPlan struct {
	Before     *carriersettlement.InvoiceMatch
	After      *carriersettlement.InvoiceMatch
	Adjustment *carriersettlement.CostEvent
	Refusal    error
}

func (s *Service) PlanMatchDecision(
	ctx context.Context,
	req *MatchDecisionRequest,
) (*MatchDecisionPlan, error) {
	match, err := s.invoiceMatchRepo.GetByID(
		ctx,
		repositories.GetCarrierInvoiceMatchByIDRequest{ID: req.MatchID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	after := *match
	plan := &MatchDecisionPlan{Before: match, After: &after}
	userID := req.TenantInfo.UserID
	now := timeutils.NowUnix()
	switch req.Decision {
	case MatchDecisionAccept:
		plan.Refusal = PlanAcceptMatch(plan.After, req.Note, userID, now)
	case MatchDecisionAcceptWithVariance:
		plan.Refusal = PlanAcceptWithVariance(plan.After, &VarianceResolution{
			Note:       req.Note,
			UserID:     userID,
			ResolvedAt: now,
		})
		if plan.Refusal == nil {
			plan.Adjustment = VarianceAdjustmentEvent(req.TenantInfo, match, now)
		}
	case MatchDecisionReject:
		plan.Refusal = PlanRejectMatch(plan.After, req.Note, userID, now)
	default:
		plan.Refusal = errortypes.NewValidationError(
			"decision",
			errortypes.ErrInvalid,
			"Carrier invoice match decision is invalid",
		)
	}
	return plan, nil
}

func (s *Service) PerformMatchDecision(
	ctx context.Context,
	req *MatchDecisionRequest,
	actor *serviceports.RequestActor,
) (*carriersettlement.InvoiceMatch, error) {
	switch req.Decision {
	case MatchDecisionAccept:
		return s.AcceptMatch(ctx, req.TenantInfo, req.MatchID, req.Note, actor)
	case MatchDecisionAcceptWithVariance:
		return s.AcceptWithVariance(ctx, req.TenantInfo, req.MatchID, req.Note, actor)
	case MatchDecisionReject:
		return s.RejectMatch(ctx, req.TenantInfo, req.MatchID, req.Note, actor)
	default:
		return nil, errortypes.NewValidationError(
			"decision",
			errortypes.ErrInvalid,
			"Carrier invoice match decision is invalid",
		)
	}
}

type LinkInvoicePlan struct {
	Carrier *carrier.Carrier
	Before  *edi.CarrierInvoice
	After   *edi.CarrierInvoice
}

func (s *Service) PlanLinkInvoice(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID, carrierID pulid.ID,
) (*LinkInvoicePlan, error) {
	linked, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         carrierID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	invoice, err := s.ediInvoiceRepo.GetCarrierInvoiceByID(
		ctx,
		repositories.GetEDICarrierInvoiceByIDRequest{ID: invoiceID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	after := *invoice
	after.CarrierID = carrierID
	return &LinkInvoicePlan{Carrier: linked, Before: invoice, After: &after}, nil
}

type matchDraft struct {
	seed           *invoiceMatchSeed
	assignment     *shipment.CarrierAssignment
	invoice        *edi.CarrierInvoice
	toleranceMinor int64
}

//nolint:cyclop,funlen // match assembly enumerates both invoice sources explicitly
func (s *Service) draftMatch(ctx context.Context, req *CreateMatchRequest) (*matchDraft, error) {
	hasEDI := req.EDICarrierInvoiceID != nil && !req.EDICarrierInvoiceID.IsNil()
	hasDocAI := req.DocumentAIExtractionID != nil && !req.DocumentAIExtractionID.IsNil()
	if hasEDI == hasDocAI {
		return nil, errortypes.NewValidationError(
			"ediCarrierInvoiceId",
			errortypes.ErrInvalid,
			"Provide exactly one source: an EDI carrier invoice or a document AI extraction",
		)
	}

	carrierID := req.CarrierID
	invoiceNumber := req.InvoiceNumber
	invoiceTotalMinor := req.InvoiceTotalMinor
	proNumber := req.ProNumber
	shipmentID := req.ShipmentID
	var invoice *edi.CarrierInvoice

	if hasEDI {
		existing, err := s.invoiceMatchRepo.GetOpenByEDIInvoiceID(
			ctx,
			req.TenantInfo,
			*req.EDICarrierInvoiceID,
		)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, errortypes.NewValidationError(
				"ediCarrierInvoiceId",
				errortypes.ErrDuplicate,
				"This EDI carrier invoice already has an open match",
			)
		}

		invoice, err = s.ediInvoiceRepo.GetCarrierInvoiceByID(
			ctx,
			repositories.GetEDICarrierInvoiceByIDRequest{
				ID:         *req.EDICarrierInvoiceID,
				TenantInfo: req.TenantInfo,
			},
		)
		if err != nil {
			return nil, err
		}
		if invoice.CarrierID.IsNil() {
			return nil, errortypes.NewValidationError(
				"carrierId",
				errortypes.ErrInvalid,
				"Link the EDI invoice to a carrier before matching",
			)
		}
		carrierID = invoice.CarrierID
		invoiceNumber = invoice.InvoiceNumber
		if invoice.TotalAmount.Valid {
			invoiceTotalMinor = money.MinorUnits(invoice.TotalAmount.Decimal)
		}
		if proNumber == "" {
			proNumber = invoice.ProNumber
		}
		if shipmentID.IsNil() {
			shipmentID = invoice.ShipmentID
		}
	} else {
		if carrierID.IsNil() {
			return nil, errortypes.NewValidationError(
				"carrierId",
				errortypes.ErrRequired,
				"Carrier is required for document AI invoice matches",
			)
		}

		existing, err := s.invoiceMatchRepo.GetOpenByExtractionID(
			ctx,
			req.TenantInfo,
			*req.DocumentAIExtractionID,
		)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, errortypes.NewValidationError(
				"documentAiExtractionId",
				errortypes.ErrDuplicate,
				"This document AI extraction already has an open match",
			)
		}
	}

	assignment, err := s.resolveMatchAssignment(ctx, req, carrierID, proNumber, shipmentID)
	if err != nil {
		return nil, err
	}

	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return &matchDraft{
		seed: &invoiceMatchSeed{
			tenantInfo:             req.TenantInfo,
			ediCarrierInvoiceID:    req.EDICarrierInvoiceID,
			documentAIExtractionID: req.DocumentAIExtractionID,
			carrierID:              carrierID,
			invoiceNumber:          invoiceNumber,
			invoiceTotalMinor:      invoiceTotalMinor,
			matchedVia:             carriersettlement.MatchViaManual,
		},
		assignment:     assignment,
		invoice:        invoice,
		toleranceMinor: control.VarianceToleranceMinor,
	}, nil
}

func newInvoiceMatch(
	seed *invoiceMatchSeed,
	assignment *shipment.CarrierAssignment,
	toleranceMinor int64,
) (*carriersettlement.InvoiceMatch, error) {
	expectedTotalMinor := money.MinorUnits(assignment.TotalCost)
	varianceMinor := seed.invoiceTotalMinor - expectedTotalMinor
	match := &carriersettlement.InvoiceMatch{
		OrganizationID:         seed.tenantInfo.OrgID,
		BusinessUnitID:         seed.tenantInfo.BuID,
		EDICarrierInvoiceID:    seed.ediCarrierInvoiceID,
		DocumentAIExtractionID: seed.documentAIExtractionID,
		CarrierID:              seed.carrierID,
		CarrierAssignmentID:    assignment.ID,
		Status:                 matchStatusForVariance(varianceMinor, toleranceMinor),
		MatchedVia:             seed.matchedVia,
		InvoiceNumber:          seed.invoiceNumber,
		InvoiceTotalMinor:      seed.invoiceTotalMinor,
		ExpectedTotalMinor:     expectedTotalMinor,
		VarianceMinor:          varianceMinor,
		CurrencyCode:           assignment.CurrencyCode,
	}
	multiErr := errortypes.NewMultiError()
	match.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return match, nil
}

type CreateMatchPlan struct {
	Match      *carriersettlement.InvoiceMatch
	Invoice    *edi.CarrierInvoice
	Assignment *shipment.CarrierAssignment
	Refusal    error
}

func (s *Service) PlanCreateMatch(
	ctx context.Context,
	req *CreateMatchRequest,
) (*CreateMatchPlan, error) {
	draft, err := s.draftMatch(ctx, req)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			return &CreateMatchPlan{Refusal: err}, nil
		}
		return nil, err
	}
	match, err := newInvoiceMatch(draft.seed, draft.assignment, draft.toleranceMinor)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			return &CreateMatchPlan{Refusal: err}, nil
		}
		return nil, err
	}
	return &CreateMatchPlan{
		Match:      match,
		Invoice:    draft.invoice,
		Assignment: draft.assignment,
	}, nil
}
