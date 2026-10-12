package bankreceiptservice

import (
	"context"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/uptrace/bun"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	DB                  kernelports.DBConnection
	Repo                bankreceiptports.BankReceiptRepository
	WorkItemRepo        bankreceiptworkitemports.BankReceiptWorkItemRepository
	PaymentRepo         customerpaymentports.CustomerPaymentRepository
	AccountingRepo      tenantports.AccountingControlRepository
	NotificationService *notificationservice.Service
	AuditService        auditports.AuditService
	// Events is told about each receipt that could not be matched on its
	// own, so an agent subscribed to bank_receipt.exception can work it. It
	// is optional because an import is worth running with nobody listening.
	Events agentports.AgentEventPublisher `optional:"true"`
}

type Service struct {
	l                   *zap.Logger
	db                  kernelports.DBConnection
	repo                bankreceiptports.BankReceiptRepository
	workItemRepo        bankreceiptworkitemports.BankReceiptWorkItemRepository
	paymentRepo         customerpaymentports.CustomerPaymentRepository
	accountingRepo      tenantports.AccountingControlRepository
	notificationService *notificationservice.Service
	auditService        auditports.AuditService
	events              agentports.AgentEventPublisher
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.bank-receipt"),
		db:                  p.DB,
		repo:                p.Repo,
		workItemRepo:        p.WorkItemRepo,
		paymentRepo:         p.PaymentRepo,
		accountingRepo:      p.AccountingRepo,
		notificationService: p.NotificationService,
		auditService:        p.AuditService,
		events:              p.Events,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *bankreceiptports.GetBankReceiptRequest,
) (*bankreceipt.BankReceipt, error) {
	return s.repo.GetByID(
		ctx,
		bankreceiptports.GetBankReceiptByIDRequest{ID: req.ReceiptID, TenantInfo: req.TenantInfo},
	)
}

func (s *Service) ListExceptions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*bankreceipt.BankReceipt, error) {
	return s.repo.ListExceptions(ctx, tenantInfo)
}

func (s *Service) GetSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	asOfDate int64,
) (*bankreceiptports.BankReceiptReconciliationSummary, error) {
	if asOfDate == 0 {
		asOfDate = timeutils.NowUnix()
	}
	return s.repo.GetSummary(
		ctx,
		bankreceiptports.GetBankReceiptSummaryRequest{TenantInfo: tenantInfo, AsOfDate: asOfDate},
	)
}

func (s *Service) SuggestMatches(
	ctx context.Context,
	req *bankreceiptports.GetBankReceiptRequest,
) ([]*bankreceiptports.BankReceiptMatchSuggestion, error) {
	receipt, err := s.repo.GetByID(
		ctx,
		bankreceiptports.GetBankReceiptByIDRequest{ID: req.ReceiptID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	candidates, err := s.paymentRepo.FindSuggestedMatchCandidates(
		ctx,
		customerpaymentports.FindCustomerPaymentMatchCandidatesRequest{
			TenantInfo:      req.TenantInfo,
			ReferenceNumber: receipt.ReferenceNumber,
			AmountMinor:     receipt.AmountMinor,
		},
	)
	if err != nil {
		return nil, err
	}
	return buildMatchSuggestions(receipt, candidates), nil
}

func (s *Service) Import(
	ctx context.Context,
	req *bankreceiptports.ImportBankReceiptRequest,
	actor *permissionports.RequestActor,
) (*bankreceipt.BankReceipt, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Bank receipt import requires an authenticated user",
		)
	}
	entity := &bankreceipt.BankReceipt{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		ImportBatchID:   req.BatchID,
		ReceiptDate:     req.ReceiptDate,
		AmountMinor:     req.AmountMinor,
		ReferenceNumber: strings.TrimSpace(req.ReferenceNumber),
		Memo:            strings.TrimSpace(req.Memo),
		Status:          bankreceipt.StatusImported,
		CreatedByID:     actor.UserID,
		UpdatedByID:     actor.UserID,
	}
	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}
	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	created, err = s.applyReconciliationPolicy(ctx, created, actor)
	if err != nil {
		return nil, err
	}
	if !req.SkipAudit {
		s.logAudit(created, nil, actor.UserID, permission.OpCreate, "Bank receipt imported")
	}
	return created, nil
}

func (s *Service) Match(
	ctx context.Context,
	req *bankreceiptports.MatchBankReceiptRequest,
	actor *permissionports.RequestActor,
) (*bankreceipt.BankReceipt, error) {
	var (
		plan    *MatchPreview
		updated *bankreceipt.BankReceipt
	)
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		plan, txErr = s.planStoredMatch(txCtx, req, actor, s.repo.LockForUpdate)
		if txErr != nil {
			return txErr
		}
		if updated, txErr = s.repo.Update(txCtx, plan.ReceiptAfter); txErr != nil {
			return txErr
		}
		if plan.WorkItemAfter != nil {
			if _, txErr = s.workItemRepo.Update(txCtx, plan.WorkItemAfter); txErr != nil {
				return txErr
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !req.SkipAudit {
		s.logAudit(
			updated,
			plan.ReceiptBefore,
			actor.UserID,
			permission.OpUpdate,
			"Bank receipt matched to customer payment",
		)
	}
	return updated, nil
}

// ReleasePaymentMatch returns the deposits matched to a payment that is being
// reversed to the reconciliation queue. The money is still in the bank, but
// the payment it was reconciled against no longer stands, so the receipt needs
// a person to decide what it was for. It runs inside the caller's transaction.
func (s *Service) ReleasePaymentMatch(
	ctx context.Context,
	req *bankreceiptports.ReleasePaymentMatchRequest,
	actor *permissionports.RequestActor,
) ([]*bankreceipt.BankReceipt, error) {
	matched, err := s.repo.ListMatchedToPayment(ctx, bankreceiptports.ListBankReceiptsMatchedToPaymentRequest{
		PaymentID:  req.PaymentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	released := make([]*bankreceipt.BankReceipt, 0, len(matched))
	for _, receipt := range matched {
		if receipt == nil {
			continue
		}
		receipt.MatchedCustomerPaymentID = pulid.Nil
		receipt.MatchedAt = nil
		receipt.MatchedByID = pulid.Nil
		updated, markErr := s.markException(ctx, receipt, actor, releasedMatchReason(req), false)
		if markErr != nil {
			return nil, markErr
		}
		released = append(released, updated)
	}

	return released, nil
}

func releasedMatchReason(req *bankreceiptports.ReleasePaymentMatchRequest) string {
	reason := "The customer payment " + req.PaymentReference + " matched to this deposit was reversed"
	if req.Reason != "" {
		reason += ": " + req.Reason
	}

	return reason
}

func (s *Service) applyReconciliationPolicy(
	ctx context.Context,
	receipt *bankreceipt.BankReceipt,
	actor *permissionports.RequestActor,
) (*bankreceipt.BankReceipt, error) {
	if receipt == nil || s.accountingRepo == nil {
		return receipt, nil
	}
	control, err := s.accountingRepo.GetByOrgID(ctx, receipt.OrganizationID)
	if err != nil {
		return receipt, err
	}
	if control.ReconciliationMode == tenant.ReconciliationModeDisabled {
		return receipt, nil
	}
	if strings.TrimSpace(receipt.ReferenceNumber) == "" {
		return s.markException(
			ctx,
			receipt,
			actor,
			"Bank receipt has no reference number for automatic matching",
			!receipt.ImportBatchID.IsNil(),
		)
	}
	candidates, err := s.paymentRepo.FindSuggestedMatchCandidates(
		ctx,
		customerpaymentports.FindCustomerPaymentMatchCandidatesRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: receipt.OrganizationID,
				BuID:  receipt.BusinessUnitID,
			},
			ReferenceNumber: receipt.ReferenceNumber,
			AmountMinor:     receipt.AmountMinor,
			ReceiptDate:     receipt.ReceiptDate,
		},
	)
	if err != nil {
		return nil, err
	}
	suggestions := buildMatchSuggestions(receipt, candidates)
	if shouldAutoMatchSuggestion(suggestions) {
		return s.Match(
			ctx,
			&bankreceiptports.MatchBankReceiptRequest{
				ReceiptID: receipt.ID,
				PaymentID: suggestions[0].CustomerPaymentID,
				SkipAudit: !receipt.ImportBatchID.IsNil(),
				TenantInfo: pagination.TenantInfo{
					OrgID: receipt.OrganizationID,
					BuID:  receipt.BusinessUnitID,
				},
			},
			actor,
		)
	}
	return s.markException(
		ctx,
		receipt,
		actor,
		"No unique customer payment match found for bank receipt",
		!receipt.ImportBatchID.IsNil(),
	)
}

func (s *Service) markException(
	ctx context.Context,
	receipt *bankreceipt.BankReceipt,
	actor *permissionports.RequestActor,
	reason string,
	skipAudit bool,
) (*bankreceipt.BankReceipt, error) {
	original := *receipt
	receipt.Status = bankreceipt.StatusException
	receipt.ExceptionReason = reason
	receipt.UpdatedByID = actor.UserID
	updated, err := s.repo.Update(ctx, receipt)
	if err != nil {
		return nil, err
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: receipt.OrganizationID,
		BuID:  receipt.BusinessUnitID,
	}
	if s.workItemRepo != nil && s.activeWorkItem(ctx, tenantInfo, receipt.ID) == nil {
		if _, createErr := s.workItemRepo.Create(
			ctx,
			&bankreceiptworkitem.WorkItem{
				OrganizationID: receipt.OrganizationID,
				BusinessUnitID: receipt.BusinessUnitID,
				BankReceiptID:  receipt.ID,
				Status:         bankreceiptworkitem.StatusOpen,
				CreatedByID:    actor.UserID,
				UpdatedByID:    actor.UserID,
			},
		); createErr != nil {
			s.l.Error("failed to open bank receipt work item",
				zap.String("receiptId", receipt.ID.String()), zap.Error(createErr))
		}
	}
	if !skipAudit {
		s.logAudit(
			updated,
			&original,
			actor.UserID,
			permission.OpUpdate,
			"Bank receipt marked as reconciliation exception",
		)
	}
	if s.notificationService != nil && s.accountingRepo != nil {
		control, controlErr := s.accountingRepo.GetByOrgID(ctx, receipt.OrganizationID)
		if controlErr == nil && control.NotifyOnReconciliationException {
			_, _ = s.notificationService.Create(
				ctx,
				&notification.Notification{
					OrganizationID: receipt.OrganizationID,
					BusinessUnitID: &receipt.BusinessUnitID,
					EventType:      "bank_receipt_reconciliation_exception",
					Priority:       notification.PriorityMedium,
					Channel:        notification.ChannelGlobal,
					Title:          "Bank receipt requires reconciliation",
					Message:        reason,
					Data: map[string]any{
						"amountMinor":     receipt.AmountMinor,
						"referenceNumber": receipt.ReferenceNumber,
					},
					RelatedEntities: map[string]any{"bankReceiptId": receipt.ID.String()},
					Source:          "bankreceiptservice.Import",
				},
			)
		}
	}
	agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
		Kind:       agent.EventBankReceiptException,
		SubjectID:  updated.ID,
		TenantInfo: tenantInfo,
	})
	return updated, nil
}

// activeWorkItem reads the open queue entry for a receipt, and reads "none"
// as none: the repository reports the absence as a not-found, which is the
// ordinary state of a receipt that was never an exception.
func (s *Service) activeWorkItem(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	receiptID pulid.ID,
) *bankreceiptworkitem.WorkItem {
	item, err := s.workItemRepo.GetActiveByReceiptID(ctx, tenantInfo, receiptID)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("failed to read bank receipt work item",
				zap.String("receiptId", receiptID.String()), zap.Error(err))
		}

		return nil
	}

	return item
}

func (s *Service) logAudit(
	current *bankreceipt.BankReceipt,
	previous *bankreceipt.BankReceipt,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceBankReceipt,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	options := []auditports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error(
			"failed to log bank receipt audit action",
			zap.Error(err),
			zap.String("receiptId", current.ID.String()),
		)
	}
}

func scoreMatchSuggestion(
	receipt *bankreceipt.BankReceipt,
	payment *customerpayment.Payment,
) (score int, reason string) {
	if receipt == nil || payment == nil {
		return 0, ""
	}
	receiptRef := stringutils.NormalizeIdentifier(receipt.ReferenceNumber)
	paymentRef := stringutils.NormalizeIdentifier(payment.ReferenceNumber)
	refExact := receiptRef != "" && receiptRef == paymentRef
	refContains := receiptRef != "" && paymentRef != "" &&
		(strings.Contains(receiptRef, paymentRef) || strings.Contains(paymentRef, receiptRef))
	amountExact := receipt.AmountMinor == payment.AmountMinor
	amountNear := receipt.AmountMinor != payment.AmountMinor &&
		intutils.AbsDiff(receipt.AmountMinor, payment.AmountMinor) <= 100
	dateNear := receipt.ReceiptDate > 0 && payment.PaymentDate > 0 &&
		intutils.AbsDiff(receipt.ReceiptDate, payment.PaymentDate) <= 86400*3
	return resolveMatchScore(refExact, refContains, amountExact, amountNear, dateNear)
}

func resolveMatchScore(
	refExact, refContains, amountExact, amountNear, dateNear bool,
) (score int, reason string) {
	if refExact && amountExact && dateNear {
		return 100, "Exact normalized reference, exact amount, and close date match"
	}
	if refExact && amountExact {
		return 95, "Exact normalized reference and amount match"
	}
	if refExact && amountNear {
		return 80, "Exact normalized reference and near amount match"
	}
	if refContains && amountExact {
		return 75, "Partial normalized reference and exact amount match"
	}
	if refExact {
		return 60, "Exact reference match"
	}
	if refContains && amountNear {
		return 55, "Partial normalized reference and near amount match"
	}
	if amountExact {
		return 40, "Exact amount match"
	}
	if amountNear {
		return 20, "Near amount match"
	}
	return 0, ""
}

func buildMatchSuggestions(
	receipt *bankreceipt.BankReceipt,
	candidates []*customerpayment.Payment,
) []*bankreceiptports.BankReceiptMatchSuggestion {
	suggestions := make([]*bankreceiptports.BankReceiptMatchSuggestion, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		score, reason := scoreMatchSuggestion(receipt, candidate)
		if score == 0 {
			continue
		}
		suggestions = append(
			suggestions,
			&bankreceiptports.BankReceiptMatchSuggestion{
				CustomerPaymentID: candidate.ID,
				ReferenceNumber:   candidate.ReferenceNumber,
				AmountMinor:       candidate.AmountMinor,
				CustomerID:        candidate.CustomerID,
				Score:             score,
				Reason:            reason,
			},
		)
	}
	sort.SliceStable(suggestions, func(i, j int) bool {
		if suggestions[i].Score == suggestions[j].Score {
			return suggestions[i].CustomerPaymentID.String() < suggestions[j].CustomerPaymentID.String()
		}
		return suggestions[i].Score > suggestions[j].Score
	})
	return suggestions
}

func shouldAutoMatchSuggestion(suggestions []*bankreceiptports.BankReceiptMatchSuggestion) bool {
	if len(suggestions) == 0 {
		return false
	}
	if suggestions[0].Score < 90 {
		return false
	}
	if len(suggestions) == 1 {
		return true
	}
	return suggestions[0].Score-suggestions[1].Score >= 15
}
