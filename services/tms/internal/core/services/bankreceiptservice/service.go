package bankreceiptservice

import (
	"context"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	repositoryports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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

	Logger           *zap.Logger
	Repo             repositoryports.BankReceiptRepository
	WorkItemRepo     repositoryports.BankReceiptWorkItemRepository
	PaymentRepo      repositoryports.CustomerPaymentRepository
	AccountingRepo   repositoryports.AccountingControlRepository
	NotificationRepo repositoryports.NotificationRepository
	AuditService     serviceports.AuditService
}

type Service struct {
	l                *zap.Logger
	repo             repositoryports.BankReceiptRepository
	workItemRepo     repositoryports.BankReceiptWorkItemRepository
	paymentRepo      repositoryports.CustomerPaymentRepository
	accountingRepo   repositoryports.AccountingControlRepository
	notificationRepo repositoryports.NotificationRepository
	auditService     serviceports.AuditService
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:                p.Logger.Named("service.bank-receipt"),
		repo:             p.Repo,
		workItemRepo:     p.WorkItemRepo,
		paymentRepo:      p.PaymentRepo,
		accountingRepo:   p.AccountingRepo,
		notificationRepo: p.NotificationRepo,
		auditService:     p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *serviceports.GetBankReceiptRequest,
) (*bankreceipt.BankReceipt, error) {
	return s.repo.GetByID(
		ctx,
		repositoryports.GetBankReceiptByIDRequest{ID: req.ReceiptID, TenantInfo: req.TenantInfo},
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
) (*repositoryports.BankReceiptReconciliationSummary, error) {
	if asOfDate == 0 {
		asOfDate = timeutils.NowUnix()
	}
	return s.repo.GetSummary(
		ctx,
		repositoryports.GetBankReceiptSummaryRequest{TenantInfo: tenantInfo, AsOfDate: asOfDate},
	)
}

func (s *Service) SuggestMatches(
	ctx context.Context,
	req *serviceports.GetBankReceiptRequest,
) ([]*serviceports.BankReceiptMatchSuggestion, error) {
	receipt, err := s.repo.GetByID(
		ctx,
		repositoryports.GetBankReceiptByIDRequest{ID: req.ReceiptID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	candidates, err := s.paymentRepo.FindSuggestedMatchCandidates(
		ctx,
		repositoryports.FindCustomerPaymentMatchCandidatesRequest{
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
	req *serviceports.ImportBankReceiptRequest,
	actor *serviceports.RequestActor,
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
	req *serviceports.MatchBankReceiptRequest,
	actor *serviceports.RequestActor,
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
			"Bank receipt matching requires an authenticated user",
		)
	}
	receipt, err := s.repo.GetByID(
		ctx,
		repositoryports.GetBankReceiptByIDRequest{ID: req.ReceiptID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	payment, err := s.paymentRepo.GetByID(
		ctx,
		repositoryports.GetCustomerPaymentByIDRequest{
			ID:         req.PaymentID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if receipt.Status == bankreceipt.StatusMatched {
		return nil, errortypes.NewBusinessError("Bank receipt is already matched")
	}
	if payment.Status != customerpayment.StatusPosted {
		return nil, errortypes.NewBusinessError("Only posted customer payments can be matched")
	}
	if receipt.AmountMinor != payment.AmountMinor {
		return nil, errortypes.NewBusinessError(
			"Bank receipt amount must match customer payment amount",
		)
	}
	original := *receipt
	now := timeutils.NowUnix()
	receipt.Status = bankreceipt.StatusMatched
	receipt.MatchedCustomerPaymentID = payment.ID
	receipt.MatchedAt = &now
	receipt.MatchedByID = actor.UserID
	receipt.UpdatedByID = actor.UserID
	updated, err := s.repo.Update(ctx, receipt)
	if err != nil {
		return nil, err
	}
	if s.workItemRepo != nil {
		if item, itemErr := s.workItemRepo.GetActiveByReceiptID(
			ctx,
			req.TenantInfo,
			receipt.ID,
		); itemErr == nil &&
			item != nil {
			resolvedAt := timeutils.NowUnix()
			item.Status = bankreceiptworkitem.StatusResolved
			item.ResolutionType = bankreceiptworkitem.ResolutionMatchedToPayment
			item.ResolutionNote = "Resolved by matching bank receipt to customer payment"
			item.ResolvedByUserID = actor.UserID
			item.ResolvedAt = &resolvedAt
			item.UpdatedByID = actor.UserID
			_, _ = s.workItemRepo.Update(ctx, item)
		}
	}
	if !req.SkipAudit {
		s.logAudit(
			updated,
			&original,
			actor.UserID,
			permission.OpUpdate,
			"Bank receipt matched to customer payment",
		)
	}
	return updated, nil
}

func (s *Service) applyReconciliationPolicy(
	ctx context.Context,
	receipt *bankreceipt.BankReceipt,
	actor *serviceports.RequestActor,
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
		repositoryports.FindCustomerPaymentMatchCandidatesRequest{
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
			&serviceports.MatchBankReceiptRequest{
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
	actor *serviceports.RequestActor,
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
	if s.workItemRepo != nil {
		if item, itemErr := s.workItemRepo.GetActiveByReceiptID(
			ctx,
			pagination.TenantInfo{
				OrgID: receipt.OrganizationID,
				BuID:  receipt.BusinessUnitID,
			},
			receipt.ID); itemErr == nil &&
			item == nil {
			_, _ = s.workItemRepo.Create(
				ctx,
				&bankreceiptworkitem.WorkItem{
					OrganizationID: receipt.OrganizationID,
					BusinessUnitID: receipt.BusinessUnitID,
					BankReceiptID:  receipt.ID,
					Status:         bankreceiptworkitem.StatusOpen,
					CreatedByID:    actor.UserID,
					UpdatedByID:    actor.UserID,
				},
			)
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
	if s.notificationRepo != nil && s.accountingRepo != nil {
		control, controlErr := s.accountingRepo.GetByOrgID(ctx, receipt.OrganizationID)
		if controlErr == nil && control.NotifyOnReconciliationException {
			_, _ = s.notificationRepo.Create(
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
	return updated, nil
}

func (s *Service) logAudit(
	current *bankreceipt.BankReceipt,
	previous *bankreceipt.BankReceipt,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &serviceports.LogActionParams{
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
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
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
) []*serviceports.BankReceiptMatchSuggestion {
	suggestions := make([]*serviceports.BankReceiptMatchSuggestion, 0, len(candidates))
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
			&serviceports.BankReceiptMatchSuggestion{
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

func shouldAutoMatchSuggestion(suggestions []*serviceports.BankReceiptMatchSuggestion) bool {
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
