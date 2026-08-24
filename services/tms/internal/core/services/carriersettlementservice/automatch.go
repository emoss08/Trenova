package carriersettlementservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const autoAcceptResolutionNote = "Auto-accepted: within variance tolerance"

func (s *Service) AutoMatchInboundInvoice(
	ctx context.Context,
	req *serviceports.AutoMatchInboundInvoiceRequest,
) (*serviceports.AutoMatchInboundInvoiceResult, error) {
	if req == nil || req.EDICarrierInvoiceID.IsNil() {
		return nil, errortypes.NewValidationError(
			"ediCarrierInvoiceId",
			errortypes.ErrRequired,
			"An EDI carrier invoice is required for auto-matching",
		)
	}
	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if !control.AutoMatchInboundInvoices {
		return &serviceports.AutoMatchInboundInvoiceResult{}, nil
	}

	existing, err := s.invoiceMatchRepo.GetOpenByEDIInvoiceID(
		ctx,
		req.TenantInfo,
		req.EDICarrierInvoiceID,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &serviceports.AutoMatchInboundInvoiceResult{
			Status: string(existing.Status),
		}, nil
	}

	invoice, err := s.ediInvoiceRepo.GetCarrierInvoiceByID(
		ctx,
		repositories.GetEDICarrierInvoiceByIDRequest{
			ID:         req.EDICarrierInvoiceID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if invoice.CarrierID.IsNil() {
		return autoMatchSkipped(invoice, "it is not linked to a carrier"), nil
	}
	if invoice.ShipmentID.IsNil() && invoice.ProNumber == "" {
		return autoMatchSkipped(
			invoice,
			"it carries no pro number or shipment reference",
		), nil
	}
	if !invoice.TotalAmount.Valid {
		return autoMatchSkipped(invoice, "it does not carry an invoice total"), nil
	}

	assignment, err := s.assignmentRepo.FindForMatching(
		ctx,
		&repositories.FindCarrierAssignmentForMatchingRequest{
			TenantInfo: req.TenantInfo,
			CarrierID:  invoice.CarrierID,
			ProNumber:  invoice.ProNumber,
			ShipmentID: invoice.ShipmentID,
		},
	)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return autoMatchSkipped(
			invoice,
			"no carrier assignment matches its pro number or shipment reference",
		), nil
	}

	invoiceID := invoice.ID
	created, err := s.createInvoiceMatch(ctx, &invoiceMatchSeed{
		tenantInfo:          req.TenantInfo,
		ediCarrierInvoiceID: &invoiceID,
		carrierID:           invoice.CarrierID,
		invoiceNumber:       invoice.InvoiceNumber,
		invoiceTotalMinor:   money.MinorUnits(invoice.TotalAmount.Decimal),
		matchedVia:          carriersettlement.MatchViaAuto,
	}, assignment, control.VarianceToleranceMinor)
	if err != nil {
		return nil, err
	}
	s.logInvoiceMatchAudit(ctx, created, nil, pulid.Nil, permission.OpCreate,
		"Carrier invoice auto-matched from inbound EDI 210")

	result := &serviceports.AutoMatchInboundInvoiceResult{
		Matched: true,
		Status:  string(created.Status),
	}
	final := created
	if created.Status == carriersettlement.InvoiceMatchStatusMatched &&
		control.AutoAcceptWithinTolerance {
		resolved, resolveErr := s.autoAcceptMatch(ctx, created)
		if resolveErr != nil {
			s.l.Error("failed to auto-accept carrier invoice match",
				zap.Error(resolveErr), zap.String("matchId", created.ID.String()))
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"carrier invoice %s was matched but could not be auto-accepted: %s",
				invoice.InvoiceNumber,
				resolveErr.Error(),
			))
		} else {
			final = resolved
			result.AutoAccepted = true
			result.Status = string(resolved.Status)
		}
	}
	s.syncEDIInvoiceReconciliation(ctx, invoice, final)
	return result, nil
}

func (s *Service) autoAcceptMatch(
	ctx context.Context,
	match *carriersettlement.InvoiceMatch,
) (*carriersettlement.InvoiceMatch, error) {
	previous := *match
	now := timeutils.NowUnix()
	match.Status = carriersettlement.InvoiceMatchStatusResolved
	match.ResolutionNote = autoAcceptResolutionNote
	match.ResolvedByID = pulid.Nil
	match.ResolvedAt = &now

	updated, err := s.invoiceMatchRepo.Update(ctx, match)
	if err != nil {
		return nil, err
	}
	s.logInvoiceMatchAudit(ctx, updated, &previous, pulid.Nil, permission.OpApprove,
		"Carrier invoice match auto-accepted: within variance tolerance")
	return updated, nil
}

func autoMatchSkipped(
	invoice *edi.CarrierInvoice,
	reason string,
) *serviceports.AutoMatchInboundInvoiceResult {
	return &serviceports.AutoMatchInboundInvoiceResult{
		Warnings: []string{fmt.Sprintf(
			"carrier invoice %s was not auto-matched: %s",
			invoice.InvoiceNumber,
			reason,
		)},
	}
}
