package invoiceservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingcontrolpolicyservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// planPost decides whether the invoice may post and, when it may, sets what
// posting sets on it: Posted, now. An invoice already posted is reported as
// such and left alone, since posting it again only settles its queue item.
// Post and PreviewPost both decide here; only Post saves.
func (s *Service) planPost(
	ctx context.Context,
	entity *invoice.Invoice,
	req *servicesports.PostInvoiceRequest,
	now int64,
) (bool, error) {
	if s.billingRepo != nil {
		control, controlErr := s.billingRepo.GetByOrgID(ctx, entity.OrganizationID)
		if controlErr != nil {
			if req.TriggeredBy == billingcontrolpolicyservice.AutoPostInvoiceTrigger {
				return false, controlErr
			}
		} else if policyErr := s.billingPolicyService().
			ValidateInvoicePosting(control, req.TriggeredBy); policyErr != nil {
			return false, policyErr
		}
	}

	switch entity.Status { //nolint:exhaustive // a draft is what posts
	case invoice.StatusVoided:
		return false, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Voided invoices cannot be posted",
		)
	case invoice.StatusPosted:
		return true, nil
	}

	if multiErr := s.validator.ValidatePost(ctx, entity, req.TenantInfo, now); multiErr != nil {
		return false, multiErr
	}

	entity.Status = invoice.StatusPosted
	entity.PostedAt = &now

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return false, multiErr
	}

	return false, nil
}

// PostPreview is what posting an invoice would write, planned by the code
// Post runs, without saving anything or taking a journal number.
type PostPreview struct {
	Before *invoice.Invoice
	After  *invoice.Invoice
	// Refusal is why Post would refuse the invoice as it stands; nothing
	// else is planned then.
	Refusal       error
	AlreadyPosted bool
	Legs          []LegChange
	QueueBefore   *billingqueue.BillingQueueItem
	QueueAfter    *billingqueue.BillingQueueItem
	// Journal is the ledger entry posting writes; nil when the organization's
	// accounting control creates none for this invoice.
	Journal *JournalPreview
	// AccountingSync is where the invoice is queued for the accounting
	// system; empty when no connection covers it.
	AccountingSync []servicesports.AccountingSyncDestination
	// EDI is the invoice's outbound 210 as it would stand after posting.
	EDI *servicesports.InvoiceEDISendPlan
}

// Refused reports whether Post would refuse the invoice as it stands.
func (p *PostPreview) Refused() bool {
	return p.Refusal != nil
}

// LegChange is one shipment the invoice bills, before and after posting.
type LegChange struct {
	Before *shipment.Shipment
	After  *shipment.Shipment
}

// JournalPreview is the entry posting books, with its lines in minor units.
type JournalPreview struct {
	AccountingDate   int64
	FiscalPeriodID   pulid.ID
	EntryStatus      string
	RequiresApproval bool
	Lines            []JournalLinePreview
}

type JournalLinePreview struct {
	GLAccountID pulid.ID
	Description string
	DebitMinor  int64
	CreditMinor int64
}

// PreviewPost plans Post for the invoice as it stands, for the person
// deciding whether to post it.
func (s *Service) PreviewPost(
	ctx context.Context,
	req *servicesports.PostInvoiceRequest,
	actor *servicesports.RequestActor,
) (*PostPreview, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	before := *entity
	preview := &PostPreview{Before: &before, After: entity}
	now := timeutils.NowUnix()

	alreadyPosted, err := s.planPost(ctx, entity, req, now)
	if err != nil {
		if isRefusal(err) {
			preview.Refusal = err

			return preview, nil
		}

		return nil, err
	}
	preview.AlreadyPosted = alreadyPosted

	queueBefore, err := s.billingQueueRepo.GetByID(
		ctx,
		&repositories.GetBillingQueueItemByIDRequest{
			ItemID:     entity.BillingQueueItemID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	if queueBefore != nil && queueBefore.Status != billingqueue.StatusPosted {
		queueAfter := *queueBefore
		queueAfter.Status = billingqueue.StatusPosted
		preview.QueueBefore, preview.QueueAfter = queueBefore, &queueAfter
	}
	if alreadyPosted {
		return preview, nil
	}

	if preview.Legs, err = s.planInvoicedLegs(ctx, entity, now, req.TenantInfo); err != nil {
		return nil, err
	}
	journal, err := s.previewJournal(ctx, entity, actor)
	switch {
	case errors.Is(err, errNoLedgerEntry):
	case isRefusal(err):
		preview.Refusal = err

		return preview, nil
	case err != nil:
		return nil, err
	default:
		preview.Journal = journal
	}
	if preview.AccountingSync, err = s.accountingSyncDestinations(ctx, entity); err != nil {
		return nil, err
	}
	preview.EDI = s.ediPlanFor(ctx, entity, req.TenantInfo)

	return preview, nil
}

func (s *Service) previewJournal(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *servicesports.RequestActor,
) (*JournalPreview, error) {
	plan, err := s.planInvoiceJournal(ctx, entity, actor)
	if err != nil {
		return nil, err
	}

	journal := &JournalPreview{
		AccountingDate:   plan.postingDate,
		FiscalPeriodID:   plan.period.ID,
		EntryStatus:      plan.entryStatus,
		RequiresApproval: plan.requiresApproval,
		Lines:            make([]JournalLinePreview, 0, len(plan.lines)),
	}
	for _, line := range plan.lines {
		journal.Lines = append(journal.Lines, JournalLinePreview{
			GLAccountID: line.GLAccountID,
			Description: line.Description,
			DebitMinor:  line.DebitAmount,
			CreditMinor: line.CreditAmount,
		})
	}

	return journal, nil
}

func (s *Service) accountingSyncDestinations(
	ctx context.Context,
	entity *invoice.Invoice,
) ([]servicesports.AccountingSyncDestination, error) {
	planner, plans := s.accountingSync.(servicesports.AccountingSyncPlanner)
	if !plans {
		return nil, nil
	}

	return planner.Destinations(ctx, servicesports.InvoiceSyncRequest(
		entity,
		accountingsync.PostedSourceEvent(entity.BillType),
	))
}

func isRefusal(err error) bool {
	return errortypes.IsError(err) ||
		errortypes.IsBusinessError(err) ||
		errortypes.IsConflictError(err)
}
