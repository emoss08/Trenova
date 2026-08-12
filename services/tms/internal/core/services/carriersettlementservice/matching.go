package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) ListInvoiceMatches(
	ctx context.Context,
	req *repositories.ListCarrierInvoiceMatchesRequest,
) (*pagination.ListResult[*carriersettlement.InvoiceMatch], error) {
	return s.invoiceMatchRepo.List(ctx, req)
}

func (s *Service) GetInvoiceMatch(
	ctx context.Context,
	req repositories.GetCarrierInvoiceMatchByIDRequest,
) (*carriersettlement.InvoiceMatch, error) {
	return s.invoiceMatchRepo.GetByID(ctx, req)
}

func (s *Service) ListUnmatchedEDIInvoices(
	ctx context.Context,
	req *repositories.ListEDICarrierInvoicesRequest,
) (*pagination.ListResult[*edi.CarrierInvoice], error) {
	return s.ediInvoiceRepo.ListCarrierInvoices(ctx, req)
}

// SuggestCarrierForInvoice looks up a carrier for an EDI carrier invoice by the
// SCAC or DOT number carried in the invoice's reference numbers.
func (s *Service) SuggestCarrierForInvoice(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID pulid.ID,
) (*carrier.Carrier, error) {
	invoice, err := s.ediInvoiceRepo.GetCarrierInvoiceByID(
		ctx,
		repositories.GetEDICarrierInvoiceByIDRequest{ID: invoiceID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if !invoice.CarrierID.IsNil() {
		return s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
			ID:         invoice.CarrierID,
			TenantInfo: tenantInfo,
		})
	}

	scac := referenceValue(invoice.ReferenceNumbers, "SCAC", "scac")
	dot := referenceValue(invoice.ReferenceNumbers, "DOT", "dot", "DOT_NUMBER")
	if scac == "" && dot == "" {
		return nil, nil //nolint:nilnil // no identity on the invoice means no suggestion
	}
	return s.carrierRepo.FindByIdentity(ctx, &repositories.FindCarrierByIdentityRequest{
		TenantInfo: tenantInfo,
		SCAC:       scac,
		DOTNumber:  dot,
	})
}

func referenceValue(refs map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := refs[key]; ok && value != "" {
			return value
		}
	}
	return ""
}

func (s *Service) LinkInvoiceToCarrier(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID, carrierID pulid.ID,
	actor *serviceports.RequestActor,
) (*edi.CarrierInvoice, error) {
	if err := requireActor(actor, "Carrier invoice linking"); err != nil {
		return nil, err
	}
	if _, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         carrierID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
	}

	invoice, err := s.ediInvoiceRepo.GetCarrierInvoiceByID(
		ctx,
		repositories.GetEDICarrierInvoiceByIDRequest{ID: invoiceID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	invoice.CarrierID = carrierID
	updated, err := s.ediInvoiceRepo.UpdateCarrierInvoice(ctx, invoice)
	if err != nil {
		return nil, err
	}
	s.publishRealtimeInvalidation(
		ctx,
		permission.ResourceCarrierInvoiceMatch.String(),
		permission.OpUpdate,
		updated.ID,
		updated.OrganizationID,
		updated.BusinessUnitID,
		actor.UserID,
	)
	return updated, nil
}

type CreateMatchRequest struct {
	TenantInfo             pagination.TenantInfo
	EDICarrierInvoiceID    *pulid.ID
	DocumentAIExtractionID *pulid.ID
	CarrierID              pulid.ID
	CarrierAssignmentID    *pulid.ID
	InvoiceNumber          string
	InvoiceTotalMinor      int64
	ProNumber              string
	ShipmentID             pulid.ID
}

//nolint:cyclop,funlen // match assembly enumerates both invoice sources explicitly
func (s *Service) CreateMatch(
	ctx context.Context,
	req *CreateMatchRequest,
	actor *serviceports.RequestActor,
) (*carriersettlement.InvoiceMatch, error) {
	if err := requireActor(actor, "Carrier invoice match creation"); err != nil {
		return nil, err
	}
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

	created, err := s.createInvoiceMatch(ctx, &invoiceMatchSeed{
		tenantInfo:             req.TenantInfo,
		ediCarrierInvoiceID:    req.EDICarrierInvoiceID,
		documentAIExtractionID: req.DocumentAIExtractionID,
		carrierID:              carrierID,
		invoiceNumber:          invoiceNumber,
		invoiceTotalMinor:      invoiceTotalMinor,
		matchedVia:             carriersettlement.MatchViaManual,
	}, assignment, control.VarianceToleranceMinor)
	if err != nil {
		return nil, err
	}
	if invoice != nil {
		s.syncEDIInvoiceReconciliation(ctx, invoice, created)
	}
	s.logInvoiceMatchAudit(ctx, created, nil, actor.UserID, permission.OpCreate,
		"Carrier invoice match created")
	return created, nil
}

type invoiceMatchSeed struct {
	tenantInfo             pagination.TenantInfo
	ediCarrierInvoiceID    *pulid.ID
	documentAIExtractionID *pulid.ID
	carrierID              pulid.ID
	invoiceNumber          string
	invoiceTotalMinor      int64
	matchedVia             carriersettlement.MatchVia
}

func matchStatusForVariance(
	varianceMinor, toleranceMinor int64,
) carriersettlement.InvoiceMatchStatus {
	if intutils.AbsDiff(varianceMinor, 0) > toleranceMinor {
		return carriersettlement.InvoiceMatchStatusVariance
	}
	return carriersettlement.InvoiceMatchStatusMatched
}

func (s *Service) createInvoiceMatch(
	ctx context.Context,
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
	return s.invoiceMatchRepo.Create(ctx, match)
}

func (s *Service) resolveMatchAssignment(
	ctx context.Context,
	req *CreateMatchRequest,
	carrierID pulid.ID,
	proNumber string,
	shipmentID pulid.ID,
) (*shipment.CarrierAssignment, error) {
	if req.CarrierAssignmentID != nil && !req.CarrierAssignmentID.IsNil() {
		return s.assignmentRepo.GetByID(ctx, &repositories.GetCarrierAssignmentByIDRequest{
			TenantInfo:          req.TenantInfo,
			CarrierAssignmentID: *req.CarrierAssignmentID,
		})
	}
	assignment, err := s.assignmentRepo.FindForMatching(
		ctx,
		&repositories.FindCarrierAssignmentForMatchingRequest{
			TenantInfo: req.TenantInfo,
			CarrierID:  carrierID,
			ProNumber:  proNumber,
			ShipmentID: shipmentID,
		},
	)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return nil, errortypes.NewValidationError(
			"carrierAssignmentId",
			errortypes.ErrInvalid,
			"No carrier assignment found matching the invoice's pro number or shipment reference",
		)
	}
	return assignment, nil
}

func (s *Service) AcceptMatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	matchID pulid.ID,
	note string,
	actor *serviceports.RequestActor,
) (*carriersettlement.InvoiceMatch, error) {
	if err := requireActor(actor, "Carrier invoice match acceptance"); err != nil {
		return nil, err
	}
	match, err := s.invoiceMatchRepo.GetByID(
		ctx,
		repositories.GetCarrierInvoiceMatchByIDRequest{ID: matchID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if match.Status != carriersettlement.InvoiceMatchStatusMatched &&
		match.Status != carriersettlement.InvoiceMatchStatusSuggested {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only suggested or matched invoices can be accepted without a variance; use accept-with-variance instead",
		)
	}

	previous := *match
	now := timeutils.NowUnix()
	match.Status = carriersettlement.InvoiceMatchStatusResolved
	match.ResolutionNote = note
	match.ResolvedByID = actor.UserID
	match.ResolvedAt = &now

	updated, err := s.invoiceMatchRepo.Update(ctx, match)
	if err != nil {
		return nil, err
	}
	s.markEDIInvoiceReconciled(ctx, tenantInfo, updated)
	s.logInvoiceMatchAudit(ctx, updated, &previous, actor.UserID, permission.OpApprove,
		"Carrier invoice match accepted")
	return updated, nil
}

// AcceptWithVariance resolves a variance match by creating an Adjustment cost
// event for the variance amount, which flows into the carrier's next
// settlement pool.
func (s *Service) AcceptWithVariance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	matchID pulid.ID,
	note string,
	actor *serviceports.RequestActor,
) (*carriersettlement.InvoiceMatch, error) {
	if err := requireActor(actor, "Carrier invoice match acceptance"); err != nil {
		return nil, err
	}
	match, err := s.invoiceMatchRepo.GetByID(
		ctx,
		repositories.GetCarrierInvoiceMatchByIDRequest{ID: matchID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if match.Status != carriersettlement.InvoiceMatchStatusVariance {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only variance matches can be accepted with a variance adjustment",
		)
	}
	if match.VarianceMinor == 0 {
		return nil, errortypes.NewValidationError(
			"varianceMinor",
			errortypes.ErrInvalid,
			"The match carries no variance to adjust",
		)
	}

	previous := *match
	adjustment, err := s.createVarianceAdjustmentEvent(ctx, tenantInfo, match)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	adjustmentID := adjustment.ID
	match.Status = carriersettlement.InvoiceMatchStatusResolved
	match.AdjustmentCostEventID = &adjustmentID
	match.ResolutionNote = note
	match.ResolvedByID = actor.UserID
	match.ResolvedAt = &now

	updated, err := s.invoiceMatchRepo.Update(ctx, match)
	if err != nil {
		return nil, err
	}
	s.markEDIInvoiceReconciled(ctx, tenantInfo, updated)
	s.logInvoiceMatchAudit(ctx, updated, &previous, actor.UserID, permission.OpApprove,
		"Carrier invoice match accepted with variance adjustment")
	return updated, nil
}

func (s *Service) createVarianceAdjustmentEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	match *carriersettlement.InvoiceMatch,
) (*carriersettlement.CostEvent, error) {
	key := "carrier-invoice-variance:" + match.ID.String()
	existing, err := s.costEventRepo.GetByIdempotencyKey(ctx, tenantInfo, key)
	if err == nil && existing != nil {
		return existing, nil
	}

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
		IdempotencyKey:      key,
		EventDate:           timeutils.NowUnix(),
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

	multiErr := errortypes.NewMultiError()
	event.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	created, err := s.costEventRepo.Create(ctx, event)
	if err != nil {
		return nil, err
	}
	s.publishCostEventInvalidation(ctx, created, permission.OpCreate, pulid.Nil)
	return created, nil
}

func (s *Service) RejectMatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	matchID pulid.ID,
	note string,
	actor *serviceports.RequestActor,
) (*carriersettlement.InvoiceMatch, error) {
	if err := requireActor(actor, "Carrier invoice match rejection"); err != nil {
		return nil, err
	}
	if note == "" {
		return nil, errortypes.NewValidationError(
			"resolutionNote",
			errortypes.ErrRequired,
			"A rejection note is required",
		)
	}
	match, err := s.invoiceMatchRepo.GetByID(
		ctx,
		repositories.GetCarrierInvoiceMatchByIDRequest{ID: matchID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if !match.Status.IsOpen() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only open matches can be rejected",
		)
	}

	previous := *match
	now := timeutils.NowUnix()
	match.Status = carriersettlement.InvoiceMatchStatusRejected
	match.ResolutionNote = note
	match.ResolvedByID = actor.UserID
	match.ResolvedAt = &now

	updated, err := s.invoiceMatchRepo.Update(ctx, match)
	if err != nil {
		return nil, err
	}
	s.logInvoiceMatchAudit(ctx, updated, &previous, actor.UserID, permission.OpReject,
		"Carrier invoice match rejected: "+note)
	return updated, nil
}

func (s *Service) syncEDIInvoiceReconciliation(
	ctx context.Context,
	invoice *edi.CarrierInvoice,
	match *carriersettlement.InvoiceMatch,
) {
	status := edi.CarrierInvoiceReconciliationStatusMatched
	if match.Status == carriersettlement.InvoiceMatchStatusVariance {
		status = edi.CarrierInvoiceReconciliationStatusVariance
	}
	invoice.ReconciliationStatus = status
	invoice.ExpectedAmount = decimalFromMinorNull(match.ExpectedTotalMinor)
	invoice.VarianceAmount = decimalFromMinorNull(match.VarianceMinor)
	if _, err := s.ediInvoiceRepo.UpdateCarrierInvoice(ctx, invoice); err != nil {
		s.l.Error("failed to sync EDI carrier invoice reconciliation status",
			zap.Error(err), zap.String("invoiceId", invoice.ID.String()))
	}
}

func (s *Service) markEDIInvoiceReconciled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	match *carriersettlement.InvoiceMatch,
) {
	if !match.HasEDISource() {
		return
	}
	invoice, err := s.ediInvoiceRepo.GetCarrierInvoiceByID(
		ctx,
		repositories.GetEDICarrierInvoiceByIDRequest{
			ID:         *match.EDICarrierInvoiceID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		s.l.Error("failed to load EDI carrier invoice for reconciliation",
			zap.Error(err), zap.String("invoiceId", match.EDICarrierInvoiceID.String()))
		return
	}
	invoice.ReconciliationStatus = edi.CarrierInvoiceReconciliationStatusMatched
	if _, err = s.ediInvoiceRepo.UpdateCarrierInvoice(ctx, invoice); err != nil {
		s.l.Error("failed to mark EDI carrier invoice reconciled",
			zap.Error(err), zap.String("invoiceId", invoice.ID.String()))
	}
}

func (s *Service) logInvoiceMatchAudit(
	ctx context.Context,
	current, previous *carriersettlement.InvoiceMatch,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	s.publishRealtimeInvalidation(
		ctx,
		permission.ResourceCarrierInvoiceMatch.String(),
		operation,
		current.ID,
		current.OrganizationID,
		current.BusinessUnitID,
		userID,
	)
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceCarrierInvoiceMatch,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if userID.IsNil() {
		params.PrincipalType = serviceports.PrincipalTypeSystem
		params.PrincipalID = serviceports.SystemPrincipalID
	}
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log carrier invoice match audit action", zap.Error(err))
	}
}

func decimalFromMinorNull(minor int64) decimal.NullDecimal {
	return decimal.NullDecimal{Decimal: money.DecimalFromMinor(minor), Valid: true}
}
