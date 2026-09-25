package detentionbillingservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	OccurrenceRepo repositories.DetentionOccurrenceRepository
	EvidenceRepo   repositories.DetentionEvidenceRepository
	InvoiceRepo    repositories.InvoiceRepository
	AuditService   services.AuditService
	Watchtower     services.WatchtowerProjector `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	occurrenceRepo repositories.DetentionOccurrenceRepository
	evidenceRepo   repositories.DetentionEvidenceRepository
	invoiceRepo    repositories.InvoiceRepository
	auditService   services.AuditService
	watchtower     services.WatchtowerProjector
	now            func() int64
}

var _ services.DetentionBillingService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) services.DetentionBillingService {
	return &Service{
		l:              p.Logger.Named("service.detention-billing"),
		occurrenceRepo: p.OccurrenceRepo,
		evidenceRepo:   p.EvidenceRepo,
		invoiceRepo:    p.InvoiceRepo,
		auditService:   p.AuditService,
		watchtower:     p.Watchtower,
		now:            timeutils.NowUnix,
	}
}

func (s *Service) HoldsForShipments(
	ctx context.Context,
	req *services.DetentionBillingHoldsRequest,
) ([]*detention.DetentionOccurrence, error) {
	if req == nil {
		return []*detention.DetentionOccurrence{}, nil
	}

	shipmentIDs := distinctIDs(req.ShipmentIDs)
	if len(shipmentIDs) == 0 {
		return []*detention.DetentionOccurrence{}, nil
	}

	return s.occurrenceRepo.ListBillingHoldsByShipments(
		ctx,
		&repositories.ListDetentionBillingHoldsRequest{
			TenantInfo:  req.TenantInfo,
			ShipmentIDs: shipmentIDs,
		},
	)
}

func (s *Service) GuardShipments(
	ctx context.Context,
	req *services.DetentionBillingHoldsRequest,
) error {
	holds, err := s.HoldsForShipments(ctx, req)
	if err != nil {
		return err
	}

	return detention.BillingHoldError(holds)
}

func (s *Service) SyncInvoiceBilling(
	ctx context.Context,
	req *services.SyncDetentionInvoiceBillingRequest,
) error {
	if req == nil {
		return nil
	}

	invoiceIDs := distinctIDs(req.InvoiceIDs)
	if len(invoiceIDs) == 0 {
		return nil
	}

	charges, err := s.invoiceRepo.ListLineCharges(ctx, &repositories.ListInvoiceLineChargesRequest{
		TenantInfo: req.TenantInfo,
		InvoiceIDs: invoiceIDs,
	})
	if err != nil {
		return err
	}
	if len(charges) == 0 {
		return nil
	}

	chargeIDs := make([]pulid.ID, 0, len(charges))
	shipmentIDs := make([]pulid.ID, 0, len(charges))
	for _, charge := range charges {
		chargeIDs = append(chargeIDs, charge.AdditionalChargeID)
		shipmentIDs = append(shipmentIDs, charge.ShipmentID)
	}
	chargeIDs = distinctIDs(chargeIDs)

	occurrences, err := s.occurrenceRepo.ListByAdditionalCharges(
		ctx,
		&repositories.ListOccurrencesByChargesRequest{
			TenantInfo:  req.TenantInfo,
			ShipmentIDs: distinctIDs(shipmentIDs),
			ChargeIDs:   chargeIDs,
		},
	)
	if err != nil {
		return err
	}
	if len(occurrences) == 0 {
		return nil
	}

	net, err := s.invoiceRepo.NetBilledByCharge(ctx, &repositories.NetBilledByChargeRequest{
		TenantInfo: req.TenantInfo,
		ChargeIDs:  chargeIDs,
	})
	if err != nil {
		return err
	}

	now := s.now()
	for _, occurrence := range occurrences {
		if occurrence == nil || occurrence.AdditionalChargeID == nil {
			continue
		}

		billed := net[*occurrence.AdditionalChargeID].GreaterThan(decimal.Zero)
		if err = s.settle(ctx, settleParams{
			occurrence: occurrence,
			billed:     billed,
			req:        req,
			now:        now,
		}); err != nil {
			return err
		}
	}

	return nil
}

type settleParams struct {
	occurrence *detention.DetentionOccurrence
	billed     bool
	req        *services.SyncDetentionInvoiceBillingRequest
	now        int64
}

// settle moves one occurrence to where the standing invoices put it, and
// records why on its evidence chain, the way an approval or a waiver is.
func (s *Service) settle(ctx context.Context, p settleParams) error {
	occurrence := p.occurrence
	original := *occurrence

	var (
		summary string
		comment string
	)
	switch {
	case p.billed && occurrence.Status != detention.OccurrenceStatusBilled:
		if err := occurrence.MarkBilled(); err != nil {
			return errortypes.NewValidationError(
				"status", errortypes.ErrInvalidOperation, err.Error())
		}
		summary = fmt.Sprintf("Billed on invoice %s at %s %s",
			invoiceLabel(p.req.InvoiceNumber),
			occurrence.BillableAmount.StringFixed(2), occurrence.Currency)
		comment = "Detention charge billed on invoice " + invoiceLabel(p.req.InvoiceNumber)
	case !p.billed && occurrence.Status == detention.OccurrenceStatusBilled:
		if err := occurrence.ReleaseBilling(); err != nil {
			return errortypes.NewValidationError(
				"status", errortypes.ErrInvalidOperation, err.Error())
		}
		summary = releaseSummary(p.req)
		comment = summary
	default:
		return nil
	}

	saved, err := s.occurrenceRepo.Update(ctx, occurrence)
	if err != nil {
		return err
	}

	if s.evidenceRepo != nil {
		if _, err = s.evidenceRepo.Append(ctx, &repositories.AppendEvidenceRequest{
			Entry: &detention.DetentionEvidence{
				OrganizationID:        saved.OrganizationID,
				BusinessUnitID:        saved.BusinessUnitID,
				DetentionOccurrenceID: saved.ID,
				Kind:                  detention.EvidenceKindStatusChange,
				Source:                detention.EvidenceSourceSystem,
				Summary:               summary,
				ObservedAt:            p.now,
				RecordedAt:            p.now,
				RecordedByID:          pulid.PtrOrNil(p.req.ActorUserID),
			},
		}); err != nil {
			return fmt.Errorf("record detention billing evidence: %w", err)
		}
	}

	updated := *saved
	ports.AfterCommit(ctx, func(runCtx context.Context) {
		s.audit(&original, &updated, p.req.ActorUserID, comment)
		if updated.Status == detention.OccurrenceStatusBilled && s.watchtower != nil {
			s.watchtower.Resolve(
				runCtx,
				p.req.TenantInfo,
				watchtower.SourceDetentionOccurrence,
				updated.ID.String(),
			)
		}
	})

	return nil
}

func releaseSummary(req *services.SyncDetentionInvoiceBillingRequest) string {
	number := invoiceLabel(req.InvoiceNumber)
	switch req.Event {
	case services.DetentionBillingInvoiceVoided:
		return "Billing released: invoice " + number +
			" was voided and no other invoice bills this charge"
	case services.DetentionBillingInvoiceAdjusted:
		return "Billing released: invoice " + number +
			" was credited and no other invoice bills this charge"
	case services.DetentionBillingInvoiceCreated:
		return "Billing released: no standing invoice bills this charge"
	default:
		return "Billing released: no standing invoice bills this charge"
	}
}

func invoiceLabel(number string) string {
	if trimmed := strings.TrimSpace(number); trimmed != "" {
		return trimmed
	}

	return "(unnumbered)"
}

func (s *Service) audit(
	original, updated *detention.DetentionOccurrence,
	userID pulid.ID,
	comment string,
) {
	if s.auditService == nil {
		return
	}

	if err := s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDetentionPolicy,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	},
		auditservice.WithComment(comment),
		auditservice.WithDiff(original, updated),
	); err != nil {
		s.l.Error("failed to log detention billing audit action", zap.Error(err))
	}
}

func distinctIDs(ids []pulid.ID) []pulid.ID {
	return slices.DeleteFunc(sliceutils.Dedupe(ids), pulid.ID.IsNil)
}
