package customerpaymentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

type invoiceLoader func(
	ctx context.Context,
	req repositories.GetInvoiceByIDRequest,
) (*invoice.Invoice, error)

type creditPlan struct {
	memoBefore     *invoice.Invoice
	memo           *invoice.Invoice
	targetsBefore  []*invoice.Invoice
	targets        []*invoice.Invoice
	applications   []*customerpayment.CreditMemoApplication
	applicationWas *customerpayment.CreditMemoApplication
}

func (s *Service) checkCreditRequest(
	ctx context.Context,
	req *serviceports.ApplyCreditMemoRequest,
	actor *serviceports.RequestActor,
) error {
	if err := requireRequest(req == nil); err != nil {
		return err
	}
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(
			"Applying a credit memo requires an authenticated user",
		)
	}
	if multiErr := s.validateCreditApplications(req); multiErr != nil {
		return multiErr
	}
	if _, err := s.validator.fiscalPeriodRepo.GetPeriodByDate(
		ctx,
		repositories.GetPeriodByDateRequest{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
			Date:  req.AccountingDate,
		},
	); err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"accountingDate",
			errortypes.ErrInvalid,
			"Accounting date must fall within a fiscal period",
		)
		return multiErr
	}

	return nil
}

func (s *Service) planCreditApplication(
	ctx context.Context,
	req *serviceports.ApplyCreditMemoRequest,
	actor *serviceports.RequestActor,
	load invoiceLoader,
) (*creditPlan, error) {
	memo, err := load(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.CreditMemoID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if multiErr := validateCreditMemoSource(memo); multiErr != nil {
		return nil, multiErr
	}

	var total int64
	for _, app := range req.Applications {
		total += app.AppliedAmountMinor
	}
	if total > memo.CreditRemainingMinor() {
		return nil, errortypes.NewValidationError(
			"applications",
			errortypes.ErrInvalid,
			"Applications exceed the credit memo's remaining balance by {0} minor units",
			total-memo.CreditRemainingMinor(),
		)
	}

	memoBefore := *memo
	plan := &creditPlan{
		memoBefore:    &memoBefore,
		memo:          memo,
		targetsBefore: make([]*invoice.Invoice, 0, len(req.Applications)),
		targets:       make([]*invoice.Invoice, 0, len(req.Applications)),
		applications:  make([]*customerpayment.CreditMemoApplication, 0, len(req.Applications)),
	}
	for idx, app := range req.Applications {
		target, loadErr := load(ctx, repositories.GetInvoiceByIDRequest{
			ID:         app.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if loadErr != nil {
			return nil, loadErr
		}
		if multiErr := validateCreditTarget(
			memo,
			target,
			app.AppliedAmountMinor,
			idx,
		); multiErr != nil {
			return nil, multiErr
		}

		targetBefore := *target
		target.ApplyPaymentMinor(app.AppliedAmountMinor)
		memo.ApplyCreditMinor(app.AppliedAmountMinor)
		plan.targetsBefore = append(plan.targetsBefore, &targetBefore)
		plan.targets = append(plan.targets, target)
		plan.applications = append(plan.applications, &customerpayment.CreditMemoApplication{
			OrganizationID:      req.TenantInfo.OrgID,
			BusinessUnitID:      req.TenantInfo.BuID,
			CreditMemoInvoiceID: memo.ID,
			InvoiceID:           target.ID,
			AppliedAmountMinor:  app.AppliedAmountMinor,
			AccountingDate:      req.AccountingDate,
			LineNumber:          idx + 1,
			Status:              customerpayment.CreditApplicationStatusApplied,
			CreatedByID:         actor.UserID,
		})
	}

	return plan, nil
}

func (s *Service) planUnapplyCredit(
	ctx context.Context,
	req *serviceports.UnapplyCreditMemoApplicationRequest,
	actor *serviceports.RequestActor,
	load invoiceLoader,
) (*creditPlan, error) {
	application, err := s.repo.GetCreditMemoApplicationByID(
		ctx,
		repositories.GetCreditMemoApplicationRequest{
			ID:         req.ApplicationID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if !application.IsApplied() {
		return nil, errortypes.NewValidationError(
			"applicationId",
			errortypes.ErrInvalidOperation,
			"This credit memo application has already been unapplied",
		)
	}

	memo, err := load(ctx, repositories.GetInvoiceByIDRequest{
		ID:         application.CreditMemoInvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	target, err := load(ctx, repositories.GetInvoiceByIDRequest{
		ID:         application.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if target.Status == invoice.StatusVoided {
		return nil, errortypes.NewValidationError(
			"applicationId",
			errortypes.ErrInvalidOperation,
			"The invoice this credit settled has been voided",
		)
	}

	memoBefore := *memo
	targetBefore := *target
	applicationBefore := *application
	target.RemovePaymentMinor(application.AppliedAmountMinor)
	memo.ReleaseCreditMinor(application.AppliedAmountMinor)

	now := timeutils.NowUnix()
	application.Status = customerpayment.CreditApplicationStatusUnapplied
	application.UnappliedAt = &now
	application.UnappliedByID = actor.UserID
	application.UnappliedReason = strings.TrimSpace(req.Reason)

	return &creditPlan{
		memoBefore:     &memoBefore,
		memo:           memo,
		targetsBefore:  []*invoice.Invoice{&targetBefore},
		targets:        []*invoice.Invoice{target},
		applications:   []*customerpayment.CreditMemoApplication{application},
		applicationWas: &applicationBefore,
	}, nil
}

func checkUnapplyRequest(
	req *serviceports.UnapplyCreditMemoApplicationRequest,
	actor *serviceports.RequestActor,
) error {
	if err := requireRequest(req == nil); err != nil {
		return err
	}
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(
			"Unapplying a credit memo requires an authenticated user",
		)
	}
	if req.ApplicationID.IsNil() {
		return errortypes.NewValidationError(
			"applicationId",
			errortypes.ErrRequired,
			"Application is required",
		)
	}

	return nil
}

func (p *creditPlan) preview() *serviceports.CreditMemoApplicationPreview {
	return &serviceports.CreditMemoApplicationPreview{
		CreditMemoBefore:  p.memoBefore,
		CreditMemoAfter:   p.memo,
		InvoicesBefore:    p.targetsBefore,
		InvoicesAfter:     p.targets,
		Applications:      p.applications,
		ApplicationBefore: p.applicationWas,
	}
}

func (s *Service) PreviewApplyCreditMemo(
	ctx context.Context,
	req *serviceports.ApplyCreditMemoRequest,
	actor *serviceports.RequestActor,
) (*serviceports.CreditMemoApplicationPreview, error) {
	if err := s.checkCreditRequest(ctx, req, actor); err != nil {
		return nil, err
	}

	plan, err := s.planCreditApplication(ctx, req, actor, s.invoiceRepo.GetByID)
	if err != nil {
		return nil, err
	}

	return plan.preview(), nil
}

func (s *Service) PreviewUnapplyCreditMemoApplication(
	ctx context.Context,
	req *serviceports.UnapplyCreditMemoApplicationRequest,
	actor *serviceports.RequestActor,
) (*serviceports.CreditMemoApplicationPreview, error) {
	if err := checkUnapplyRequest(req, actor); err != nil {
		return nil, err
	}

	plan, err := s.planUnapplyCredit(ctx, req, actor, s.invoiceRepo.GetByID)
	if err != nil {
		return nil, err
	}

	return plan.preview(), nil
}
