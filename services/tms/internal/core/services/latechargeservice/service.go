package latechargeservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/latecharge"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	Repo                repositories.LateChargeRepository
	BillingControlRepo  repositories.BillingControlRepository
	InvoiceService      servicesports.InvoiceService
	AuditService        servicesports.AuditService
	NotificationService *notificationservice.Service `optional:"true"`
}

// Service assesses late charges. Each customer gets one debit memo per run
// carrying a line per (invoice, period); the assessment rows are written
// first so a concurrent run cannot charge the same period twice.
type Service struct {
	l                   *zap.Logger
	repo                repositories.LateChargeRepository
	billingRepo         repositories.BillingControlRepository
	invoiceService      servicesports.InvoiceService
	auditService        servicesports.AuditService
	notificationService *notificationservice.Service
}

func New(p Params) servicesports.LateChargeService {
	return &Service{
		l:                   p.Logger.Named("service.late-charge"),
		repo:                p.Repo,
		billingRepo:         p.BillingControlRepo,
		invoiceService:      p.InvoiceService,
		auditService:        p.AuditService,
		notificationService: p.NotificationService,
	}
}

func refuseDisabled(mode tenant.LateChargeAssessmentMode) error {
	if mode != tenant.LateChargeAssessmentModeDisabled {
		return nil
	}

	return errortypes.NewValidationError(
		"mode",
		errortypes.ErrInvalidOperation,
		"Late charge assessment is disabled for this organization; enable it in billing control or run a preview",
	)
}

func memosAutoPost(control *tenant.BillingControl) bool {
	return control.InvoicePostingMode == tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions
}

func (s *Service) PlanAssess(
	ctx context.Context,
	req *servicesports.LateChargeAssessmentRequest,
	actor *servicesports.RequestActor,
) (*servicesports.LateChargeAssessmentResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	planned := *req
	planned.Preview = true
	result, err := s.Assess(ctx, &planned, actor)
	if err != nil {
		return nil, err
	}
	if err = refuseDisabled(result.Mode); err != nil {
		return nil, err
	}

	return result, nil
}

// customerPlan is what one customer is owed before anything is written.
type customerPlan struct {
	result      *servicesports.LateChargeCustomerResult
	assessments []*latecharge.LateChargeAssessment
}

func (s *Service) Assess(
	ctx context.Context,
	req *servicesports.LateChargeAssessmentRequest,
	actor *servicesports.RequestActor,
) (*servicesports.LateChargeAssessmentResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	asOf := req.AsOfDate
	if asOf <= 0 {
		asOf = timeutils.NowUnix()
	}

	control, err := s.billingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	mode := control.LateChargeAssessmentMode
	if !mode.IsValid() {
		mode = tenant.LateChargeAssessmentModeDisabled
	}
	result := &servicesports.LateChargeAssessmentResult{
		AsOfDate:  asOf,
		Preview:   req.Preview,
		Mode:      mode,
		AutoPost:  memosAutoPost(control),
		Customers: make([]*servicesports.LateChargeCustomerResult, 0),
	}
	if !req.Preview {
		if err = refuseDisabled(mode); err != nil {
			return nil, err
		}
	}

	candidates, err := s.repo.ListCandidates(ctx, &repositories.ListLateChargeCandidatesRequest{
		TenantInfo:  req.TenantInfo,
		CustomerIDs: req.CustomerIDs,
		AsOfDate:    asOf,
	})
	if err != nil {
		return nil, err
	}

	minimumMinor := money.MinorUnits(control.LateChargeMinimumAmount)
	plans := buildPlans(candidates, asOf, minimumMinor, actor)
	for _, plan := range plans {
		result.Customers = append(result.Customers, plan.result)
		if plan.result.Skipped {
			result.CustomersSkipped++
			continue
		}
		if req.Preview {
			result.TotalChargeMinor += plan.result.TotalChargeMinor
			continue
		}
		if err = s.raiseMemo(ctx, req.TenantInfo, control, plan, actor); err != nil {
			s.l.Error(
				"late charge memo failed",
				zap.String("customerId", plan.result.CustomerID.String()),
				zap.Error(err),
			)
			plan.result.Skipped = true
			plan.result.SkipReason = err.Error()
			result.CustomersSkipped++
			continue
		}
		if plan.result.Skipped {
			result.CustomersSkipped++
			continue
		}
		result.MemosCreated++
		if plan.result.Posted {
			result.MemosPosted++
		}
		result.TotalChargeMinor += plan.result.TotalChargeMinor
	}

	return result, nil
}

// buildPlans turns candidates into one plan per customer, in candidate order,
// with every unassessed period that has begun by asOf.
func buildPlans(
	candidates []*repositories.LateChargeCandidate,
	asOf int64,
	minimumMinor int64,
	actor *servicesports.RequestActor,
) []*customerPlan {
	byCustomer := make(map[pulid.ID]*customerPlan, len(candidates))
	ordered := make([]*customerPlan, 0, len(candidates))
	for _, candidate := range candidates {
		plan, ok := byCustomer[candidate.CustomerID]
		if !ok {
			plan = &customerPlan{
				result: &servicesports.LateChargeCustomerResult{
					CustomerID:   candidate.CustomerID,
					CustomerName: candidate.CustomerName,
					CurrencyCode: candidate.CurrencyCode,
					Lines:        make([]*servicesports.LateChargeLine, 0, 4),
				},
			}
			byCustomer[candidate.CustomerID] = plan
			ordered = append(ordered, plan)
		}
		overdueStart := latecharge.OverdueStart(candidate.DueDate, candidate.GracePeriodDays)
		for _, idx := range latecharge.PendingPeriods(overdueStart, asOf, candidate.AssessedPeriods) {
			charge := latecharge.ChargeMinor(candidate.OpenBalanceMinor, candidate.RatePercent)
			if charge <= 0 {
				continue
			}
			start, end := latecharge.PeriodBounds(overdueStart, idx)
			plan.result.Lines = append(plan.result.Lines, &servicesports.LateChargeLine{
				InvoiceID:             candidate.InvoiceID,
				InvoiceNumber:         candidate.InvoiceNumber,
				PeriodIndex:           idx,
				PeriodStart:           start,
				PeriodEnd:             end,
				BasisOpenBalanceMinor: candidate.OpenBalanceMinor,
				RatePercent:           candidate.RatePercent,
				ChargeMinor:           charge,
			})
			plan.result.TotalChargeMinor += charge
			plan.assessments = append(plan.assessments, &latecharge.LateChargeAssessment{
				CustomerID:            candidate.CustomerID,
				SourceInvoiceID:       candidate.InvoiceID,
				PeriodIndex:           idx,
				PeriodStart:           start,
				PeriodEnd:             end,
				AsOfDate:              asOf,
				BasisOpenBalanceMinor: candidate.OpenBalanceMinor,
				RatePercent:           candidate.RatePercent,
				ChargeMinor:           charge,
				CreatedByID:           actorUserID(actor),
			})
		}
	}

	for _, plan := range ordered {
		switch {
		case len(plan.result.Lines) == 0:
			plan.result.Skipped = true
			plan.result.SkipReason = "Every overdue period has already been assessed"
		case plan.result.TotalChargeMinor < minimumMinor:
			plan.result.Skipped = true
			plan.result.SkipReason = fmt.Sprintf(
				"Total late charge %s is below the organization minimum %s",
				money.DecimalFromMinor(plan.result.TotalChargeMinor).StringFixed(2),
				money.DecimalFromMinor(minimumMinor).StringFixed(2),
			)
		}
	}

	return ordered
}

// raiseMemo writes the assessment rows, then raises the memo they add up to.
// Rows another run got to first are not returned by the insert, so the memo
// carries only what this run charged; if nothing was inserted, there is no
// memo, and if the memo fails, the rows are taken back.
func (s *Service) raiseMemo(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *tenant.BillingControl,
	plan *customerPlan,
	actor *servicesports.RequestActor,
) error {
	memoID := pulid.MustNew("inv_")
	runKey := pulid.MustNew("lcr_").String()
	for _, assessment := range plan.assessments {
		assessment.OrganizationID = tenantInfo.OrgID
		assessment.BusinessUnitID = tenantInfo.BuID
		assessment.DebitMemoInvoiceID = memoID
		assessment.RunKey = runKey
	}

	inserted, err := s.repo.InsertAssessments(ctx, plan.assessments)
	if err != nil {
		return err
	}
	if len(inserted) == 0 {
		plan.result.Skipped = true
		plan.result.SkipReason = "Another run assessed these periods first"
		plan.result.Lines = plan.result.Lines[:0]
		plan.result.TotalChargeMinor = 0
		return nil
	}
	plan.result.Lines, plan.result.TotalChargeMinor = linesFromAssessments(
		inserted,
		plan.result.Lines,
	)

	memo, err := s.invoiceService.CreateMemo(ctx, &servicesports.CreateMemoRequest{
		ID:          memoID,
		TenantInfo:  tenantInfo,
		CustomerID:  plan.result.CustomerID,
		BillType:    billingqueue.BillTypeDebitMemo,
		Lines:       memoLines(plan.result.Lines),
		Reason:      "Late charges assessed as of " + formatDate(inserted[0].AsOfDate),
		InvoiceDate: inserted[0].AsOfDate,
		Memo:        "Late charge run " + runKey,
		MemoKind:    invoice.MemoKindLateCharge,
		AutoPost:    memosAutoPost(control),
	}, actor)
	if err != nil {
		if _, delErr := s.repo.DeleteByRunKey(ctx, tenantInfo, runKey); delErr != nil {
			s.l.Error(
				"failed to roll back late charge assessments",
				zap.String("runKey", runKey),
				zap.Error(delErr),
			)
		}
		return err
	}

	stampMemoLines(inserted, memo)
	if err = s.repo.SetDebitMemoLines(ctx, inserted); err != nil {
		s.l.Warn(
			"failed to stamp late charge memo lines",
			zap.String("runKey", runKey),
			zap.Error(err),
		)
	}

	plan.result.DebitMemoID = memo.ID
	plan.result.DebitMemoNumber = memo.Number
	plan.result.Posted = memo.Status == invoice.StatusPosted
	s.logAudit(memo, inserted, actor)
	s.notify(ctx, memo, plan.result)

	return nil
}

// linesFromAssessments keeps only the lines whose assessment this run wrote,
// in the assessment order, so the memo and the rows agree line for line.
func linesFromAssessments(
	inserted []*latecharge.LateChargeAssessment,
	planned []*servicesports.LateChargeLine,
) ([]*servicesports.LateChargeLine, int64) {
	type key struct {
		invoiceID pulid.ID
		period    int
	}
	byKey := make(map[key]*servicesports.LateChargeLine, len(planned))
	for _, line := range planned {
		byKey[key{line.InvoiceID, line.PeriodIndex}] = line
	}
	lines := make([]*servicesports.LateChargeLine, 0, len(inserted))
	var total int64
	for _, assessment := range inserted {
		line, ok := byKey[key{assessment.SourceInvoiceID, assessment.PeriodIndex}]
		if !ok {
			continue
		}
		lines = append(lines, line)
		total += line.ChargeMinor
	}

	return lines, total
}

func memoLines(lines []*servicesports.LateChargeLine) []*servicesports.CreateMemoLineInput {
	inputs := make([]*servicesports.CreateMemoLineInput, 0, len(lines))
	for _, line := range lines {
		inputs = append(inputs, &servicesports.CreateMemoLineInput{
			Description: fmt.Sprintf(
				"Late charge on %s, period %d (%s to %s), %s%% of %s",
				line.InvoiceNumber,
				line.PeriodIndex,
				formatDate(line.PeriodStart),
				formatDate(line.PeriodEnd),
				line.RatePercent.String(),
				money.DecimalFromMinor(line.BasisOpenBalanceMinor).StringFixed(2),
			),
			Amount:   money.DecimalFromMinor(line.ChargeMinor),
			Quantity: decimal.NewFromInt(1),
		})
	}

	return inputs
}

// stampMemoLines pairs assessments with memo lines by position: the memo was
// built from the assessments in this order.
func stampMemoLines(inserted []*latecharge.LateChargeAssessment, memo *invoice.Invoice) {
	if memo == nil {
		return
	}
	for idx, assessment := range inserted {
		if idx >= len(memo.Lines) || memo.Lines[idx] == nil {
			break
		}
		assessment.DebitMemoInvoiceID = memo.ID
		assessment.DebitMemoLineID = memo.Lines[idx].ID
	}
}

func (s *Service) logAudit(
	memo *invoice.Invoice,
	inserted []*latecharge.LateChargeAssessment,
	actor *servicesports.RequestActor,
) {
	if s.auditService == nil || memo == nil {
		return
	}
	auditActor := actor.AuditActorOrSystem()
	params := &servicesports.LogActionParams{
		Resource:       permission.ResourceInvoice,
		ResourceID:     memo.ID.String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		OrganizationID: memo.OrganizationID,
		BusinessUnitID: memo.BusinessUnitID,
		CurrentState: jsonutils.MustToJSON(
			map[string]any{"invoiceId": memo.ID.String(), "assessments": inserted},
		),
	}
	if err := s.auditService.LogAction(
		params,
		auditservice.WithComment(fmt.Sprintf("Late charge debit memo raised for %d period(s)", len(inserted))),
	); err != nil {
		s.l.Warn("failed to log late charge audit", zap.Error(err))
	}
}

func (s *Service) notify(
	ctx context.Context,
	memo *invoice.Invoice,
	result *servicesports.LateChargeCustomerResult,
) {
	if s.notificationService == nil || memo == nil {
		return
	}
	if _, err := s.notificationService.Create(ctx, &notification.Notification{
		OrganizationID: memo.OrganizationID,
		BusinessUnitID: &memo.BusinessUnitID,
		EventType:      "late_charges_assessed",
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          "Late charges assessed for " + result.CustomerName,
		Message: fmt.Sprintf(
			"Debit memo %s raised for %s across %d period(s).",
			memo.Number,
			money.DecimalFromMinor(result.TotalChargeMinor).StringFixed(2),
			len(result.Lines),
		),
		Data: map[string]any{
			"invoiceId":        memo.ID.String(),
			"invoiceNumber":    memo.Number,
			"customerId":       result.CustomerID.String(),
			"totalChargeMinor": result.TotalChargeMinor,
			"posted":           result.Posted,
		},
		Source: "latechargeservice.Assess",
	}); err != nil {
		s.l.Warn("failed to create late charge notification", zap.Error(err))
	}
}

func actorUserID(actor *servicesports.RequestActor) pulid.ID {
	if actor == nil {
		return pulid.Nil
	}

	return actor.UserID
}

func formatDate(unix int64) string {
	return time.Unix(unix, 0).UTC().Format("2006-01-02")
}
