package customerpaymentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	DB             ports.DBConnection
	Repo           repositories.CustomerPaymentRepository
	InvoiceRepo    repositories.InvoiceRepository
	AccountingRepo repositories.AccountingControlRepository
	JournalRepo    repositories.JournalPostingRepository
	Generator      seqgen.Generator
	Validator      *Validator
	AuditService   serviceports.AuditService
}

type Service struct {
	l              *zap.Logger
	db             ports.DBConnection
	repo           repositories.CustomerPaymentRepository
	invoiceRepo    repositories.InvoiceRepository
	accountingRepo repositories.AccountingControlRepository
	journalRepo    repositories.JournalPostingRepository
	generator      seqgen.Generator
	validator      *Validator
	auditService   serviceports.AuditService
}

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.customer-payment"), db: p.DB, repo: p.Repo, invoiceRepo: p.InvoiceRepo, accountingRepo: p.AccountingRepo, journalRepo: p.JournalRepo, generator: p.Generator, validator: p.Validator, auditService: p.AuditService}
}

func (s *Service) Get(ctx context.Context, req *serviceports.GetCustomerPaymentRequest) (*customerpayment.Payment, error) {
	return s.repo.GetByID(ctx, repositories.GetCustomerPaymentByIDRequest{ID: req.PaymentID, TenantInfo: req.TenantInfo})
}

func (s *Service) PostAndApply(ctx context.Context, req *serviceports.PostCustomerPaymentRequest, actor *serviceports.RequestActor) (*customerpayment.Payment, error) {
	if req == nil {
		return nil, errortypes.NewValidationError("request", errortypes.ErrRequired, "Request is required")
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Customer payment posting requires an authenticated user")
	}
	control, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	entity := &customerpayment.Payment{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		CustomerID:      req.CustomerID,
		PaymentDate:     req.PaymentDate,
		AccountingDate:  req.AccountingDate,
		AmountMinor:     req.AmountMinor,
		Status:          customerpayment.StatusPosted,
		PaymentMethod:   req.PaymentMethod,
		ReferenceNumber: req.ReferenceNumber,
		Memo:            req.Memo,
		CurrencyCode:    req.CurrencyCode,
		CreatedByID:     actor.UserID,
		UpdatedByID:     actor.UserID,
		Applications:    mapApplications(req.Applications),
	}

	invoices, period, me := s.validator.ValidatePostAndApply(ctx, entity, repositories.GetInvoiceByIDRequest{TenantInfo: req.TenantInfo}, control)
	if me != nil {
		return nil, me
	}

	batchNumber, err := s.generator.GenerateJournalBatchNumber(ctx, entity.OrganizationID, entity.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.generator.GenerateJournalEntryNumber(ctx, entity.OrganizationID, entity.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt := paymentPostingWorkflow(control, actor.UserID, now)
	originalInvoices := cloneInvoices(invoices)
	var created *customerpayment.Payment
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		created, txErr = s.repo.Create(txCtx, entity)
		if txErr != nil {
			return txErr
		}

		for idx, inv := range invoices {
			inv.ApplyPaymentMinor(entity.Applications[idx].AppliedAmountMinor)
			inv, txErr = s.invoiceRepo.Update(txCtx, inv)
			if txErr != nil {
				return txErr
			}
			invoices[idx] = inv
		}

		batchID := pulid.MustNew("jb_")
		entryID := pulid.MustNew("je_")
		sourceID := pulid.MustNew("jsrc_")
		entryDescription := fmt.Sprintf("Customer payment %s", created.ID.String())
		appliedCreditAccountID := control.DefaultARAccountID
		if control.AccountingBasis == tenant.AccountingBasisCash || control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt {
			appliedCreditAccountID = control.DefaultRevenueAccountID
			entryDescription = fmt.Sprintf("Customer cash receipt %s", created.ID.String())
		}
		lines := make([]repositories.JournalPostingLine, 0, 2+len(created.Applications))
		lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: control.DefaultCashAccountID, LineNumber: 1, Description: entryDescription, DebitAmount: created.AmountMinor, NetAmount: created.AmountMinor, CustomerID: entity.CustomerID})
		lineNumber := int16(2)
		for _, app := range created.Applications {
			if app == nil || app.AppliedAmountMinor <= 0 {
				continue
			}
			lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: appliedCreditAccountID, LineNumber: lineNumber, Description: entryDescription, CreditAmount: app.AppliedAmountMinor, NetAmount: -app.AppliedAmountMinor, CustomerID: entity.CustomerID})
			lineNumber++
		}
		if created.UnappliedAmountMinor > 0 {
			lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: control.DefaultUnappliedCashAccountID, LineNumber: lineNumber, Description: entryDescription, CreditAmount: created.UnappliedAmountMinor, NetAmount: -created.UnappliedAmountMinor, CustomerID: entity.CustomerID})
		}

		if txErr = s.journalRepo.CreatePosting(txCtx, repositories.CreateJournalPostingParams{
			BatchID:              batchID,
			OrganizationID:       entity.OrganizationID,
			BusinessUnitID:       entity.BusinessUnitID,
			BatchNumber:          batchNumber,
			BatchType:            "System",
			BatchStatus:          batchStatus,
			BatchDescription:     entryDescription,
			FiscalYearID:         period.FiscalYearID,
			FiscalPeriodID:       period.ID,
			AccountingDate:       entity.AccountingDate,
			PostedAt:             postedAt,
			PostedByID:           postedByID,
			CreatedByID:          actor.UserID,
			UpdatedByID:          actor.UserID,
			EntryID:              entryID,
			EntryNumber:          entryNumber,
			EntryType:            "Standard",
			EntryStatus:          entryStatus,
			ReferenceNumber:      entity.ReferenceNumber,
			ReferenceType:        tenant.JournalSourceEventCustomerPaymentPosted.String(),
			ReferenceID:          created.ID.String(),
			EntryDescription:     entryDescription,
			TotalDebit:           created.AmountMinor,
			TotalCredit:          created.AmountMinor,
			IsPosted:             postedAt != nil,
			IsAutoGenerated:      false,
			RequiresApproval:     requiresApproval,
			IsApproved:           isApproved,
			ApprovedByID:         approvedByID,
			ApprovedAt:           approvedAt,
			SourceID:             sourceID,
			SourceObjectType:     "CustomerPayment",
			SourceObjectID:       created.ID.String(),
			SourceEventType:      tenant.JournalSourceEventCustomerPaymentPosted.String(),
			SourceStatus:         entryStatus,
			SourceDocumentNumber: entity.ReferenceNumber,
			SourceIdempotencyKey: "customer-payment-posted:" + created.ID.String(),
			Lines:                lines,
		}); txErr != nil {
			return txErr
		}

		created.PostedBatchID = batchID
		created.UpdatedByID = actor.UserID
		created, txErr = s.repo.Update(txCtx, created)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(created, nil, actor.UserID, permission.OpCreate, "Customer payment posted")
	for idx, inv := range invoices {
		s.logInvoiceAudit(originalInvoices[idx], inv, actor.UserID)
	}
	return created, nil
}

func (s *Service) ApplyUnapplied(ctx context.Context, req *serviceports.ApplyCustomerPaymentRequest, actor *serviceports.RequestActor) (*customerpayment.Payment, error) {
	if req == nil {
		return nil, errortypes.NewValidationError("request", errortypes.ErrRequired, "Request is required")
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Customer payment application requires an authenticated user")
	}
	payment, err := s.repo.GetByID(ctx, repositories.GetCustomerPaymentByIDRequest{ID: req.PaymentID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	control, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	applications := mapApplications(req.Applications)
	invoices, period, me := s.validator.ValidateApplyUnapplied(ctx, payment, req.AccountingDate, applications, repositories.GetInvoiceByIDRequest{TenantInfo: req.TenantInfo}, control)
	if me != nil {
		return nil, me
	}
	entryNumber, err := s.generator.GenerateJournalEntryNumber(ctx, payment.OrganizationID, payment.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	batchNumber, err := s.generator.GenerateJournalBatchNumber(ctx, payment.OrganizationID, payment.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()
	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt := paymentPostingWorkflow(control, actor.UserID, now)
	originalPayment := *payment
	originalInvoices := cloneInvoices(invoices)
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		payment.Applications = append(payment.Applications, applications...)
		payment.UpdatedByID = actor.UserID
		for idx, inv := range invoices {
			inv.ApplyPaymentMinor(applications[idx].AppliedAmountMinor)
			updatedInvoice, txErr := s.invoiceRepo.Update(txCtx, inv)
			if txErr != nil {
				return txErr
			}
			invoices[idx] = updatedInvoice
		}
		updatedPayment, txErr := s.repo.Update(txCtx, payment)
		if txErr != nil {
			return txErr
		}
		payment = updatedPayment

		entryDescription := fmt.Sprintf("Customer payment application %s", payment.ID.String())
		creditAccountID := control.DefaultARAccountID
		if control.AccountingBasis == tenant.AccountingBasisCash || control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt {
			creditAccountID = control.DefaultRevenueAccountID
			entryDescription = fmt.Sprintf("Customer payment revenue recognition %s", payment.ID.String())
		}
		lines := make([]repositories.JournalPostingLine, 0, len(applications)+1)
		var totalApplied int64
		lineNumber := int16(1)
		lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: control.DefaultUnappliedCashAccountID, LineNumber: lineNumber, Description: entryDescription, DebitAmount: payment.AppliedAmountMinor - originalPayment.AppliedAmountMinor, NetAmount: payment.AppliedAmountMinor - originalPayment.AppliedAmountMinor, CustomerID: payment.CustomerID})
		lineNumber++
		for _, app := range applications {
			if app == nil {
				continue
			}
			totalApplied += app.AppliedAmountMinor
			lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: creditAccountID, LineNumber: lineNumber, Description: entryDescription, CreditAmount: app.AppliedAmountMinor, NetAmount: -app.AppliedAmountMinor, CustomerID: payment.CustomerID})
			lineNumber++
		}
		if txErr = s.journalRepo.CreatePosting(txCtx, repositories.CreateJournalPostingParams{BatchID: pulid.MustNew("jb_"), OrganizationID: payment.OrganizationID, BusinessUnitID: payment.BusinessUnitID, BatchNumber: batchNumber, BatchType: "System", BatchStatus: batchStatus, BatchDescription: entryDescription, FiscalYearID: period.FiscalYearID, FiscalPeriodID: period.ID, AccountingDate: req.AccountingDate, PostedAt: postedAt, PostedByID: postedByID, CreatedByID: actor.UserID, UpdatedByID: actor.UserID, EntryID: pulid.MustNew("je_"), EntryNumber: entryNumber, EntryType: "Standard", EntryStatus: entryStatus, ReferenceNumber: payment.ReferenceNumber, ReferenceType: "CustomerPaymentApplied", ReferenceID: payment.ID.String(), EntryDescription: entryDescription, TotalDebit: totalApplied, TotalCredit: totalApplied, IsPosted: postedAt != nil, IsAutoGenerated: false, RequiresApproval: requiresApproval, IsApproved: isApproved, ApprovedByID: approvedByID, ApprovedAt: approvedAt, SourceID: pulid.MustNew("jsrc_"), SourceObjectType: "CustomerPayment", SourceObjectID: payment.ID.String(), SourceEventType: "CustomerPaymentApplied", SourceStatus: entryStatus, SourceDocumentNumber: payment.ReferenceNumber, SourceIdempotencyKey: "customer-payment-applied:" + payment.ID.String() + ":" + fmt.Sprint(payment.AppliedAmountMinor), Lines: lines}); txErr != nil {
			return txErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logAudit(payment, &originalPayment, actor.UserID, permission.OpUpdate, "Customer payment unapplied amount applied")
	for idx, inv := range invoices {
		s.logInvoiceAudit(originalInvoices[idx], inv, actor.UserID)
	}
	return payment, nil
}

func (s *Service) Reverse(ctx context.Context, req *serviceports.ReverseCustomerPaymentRequest, actor *serviceports.RequestActor) (*customerpayment.Payment, error) {
	if req == nil {
		return nil, errortypes.NewValidationError("request", errortypes.ErrRequired, "Request is required")
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Customer payment reversal requires an authenticated user")
	}
	payment, err := s.repo.GetByID(ctx, repositories.GetCustomerPaymentByIDRequest{ID: req.PaymentID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	control, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	invoices, period, me := s.validator.ValidateReverse(ctx, payment, req.AccountingDate, control, repositories.GetInvoiceByIDRequest{TenantInfo: req.TenantInfo})
	if me != nil {
		return nil, me
	}
	batchNumber, err := s.generator.GenerateJournalBatchNumber(ctx, payment.OrganizationID, payment.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.generator.GenerateJournalEntryNumber(ctx, payment.OrganizationID, payment.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()
	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt := paymentPostingWorkflow(control, actor.UserID, now)
	originalPayment := *payment
	originalInvoices := cloneInvoices(invoices)
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		for idx, app := range payment.Applications {
			if app == nil {
				continue
			}
			invoices[idx].RemovePaymentMinor(app.AppliedAmountMinor)
			updatedInvoice, txErr := s.invoiceRepo.Update(txCtx, invoices[idx])
			if txErr != nil {
				return txErr
			}
			invoices[idx] = updatedInvoice
		}

		entryDescription := fmt.Sprintf("Customer payment reversal %s", payment.ID.String())
		debitAccountID := control.DefaultARAccountID
		if control.AccountingBasis == tenant.AccountingBasisCash || control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt {
			debitAccountID = control.DefaultRevenueAccountID
			entryDescription = fmt.Sprintf("Customer cash receipt reversal %s", payment.ID.String())
		}
		lines := make([]repositories.JournalPostingLine, 0, 3)
		lineNumber := int16(1)
		if payment.AppliedAmountMinor > 0 {
			lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: debitAccountID, LineNumber: lineNumber, Description: entryDescription, DebitAmount: payment.AppliedAmountMinor, NetAmount: payment.AppliedAmountMinor, CustomerID: payment.CustomerID})
			lineNumber++
		}
		if payment.UnappliedAmountMinor > 0 {
			lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: control.DefaultUnappliedCashAccountID, LineNumber: lineNumber, Description: entryDescription, DebitAmount: payment.UnappliedAmountMinor, NetAmount: payment.UnappliedAmountMinor, CustomerID: payment.CustomerID})
			lineNumber++
		}
		lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: control.DefaultCashAccountID, LineNumber: lineNumber, Description: entryDescription, CreditAmount: payment.AmountMinor, NetAmount: -payment.AmountMinor, CustomerID: payment.CustomerID})

		batchID := pulid.MustNew("jb_")
		if txErr := s.journalRepo.CreatePosting(txCtx, repositories.CreateJournalPostingParams{BatchID: batchID, OrganizationID: payment.OrganizationID, BusinessUnitID: payment.BusinessUnitID, BatchNumber: batchNumber, BatchType: "Reversal", BatchStatus: batchStatus, BatchDescription: entryDescription, FiscalYearID: period.FiscalYearID, FiscalPeriodID: period.ID, AccountingDate: req.AccountingDate, PostedAt: postedAt, PostedByID: postedByID, CreatedByID: actor.UserID, UpdatedByID: actor.UserID, EntryID: pulid.MustNew("je_"), EntryNumber: entryNumber, EntryType: "Reversal", EntryStatus: entryStatus, ReferenceNumber: payment.ReferenceNumber, ReferenceType: tenant.JournalSourceEventCustomerPaymentReversed.String(), ReferenceID: payment.ID.String(), EntryDescription: entryDescription, TotalDebit: payment.AmountMinor, TotalCredit: payment.AmountMinor, IsPosted: postedAt != nil, IsAutoGenerated: false, ReversalDate: postedAt, ReversalReason: req.Reason, RequiresApproval: requiresApproval, IsApproved: isApproved, ApprovedByID: approvedByID, ApprovedAt: approvedAt, SourceID: pulid.MustNew("jsrc_"), SourceObjectType: "CustomerPayment", SourceObjectID: payment.ID.String(), SourceEventType: tenant.JournalSourceEventCustomerPaymentReversed.String(), SourceStatus: entryStatus, SourceDocumentNumber: payment.ReferenceNumber, SourceIdempotencyKey: "customer-payment-reversed:" + payment.ID.String(), Lines: lines}); txErr != nil {
			return txErr
		}

		nowCopy := now
		payment.Status = customerpayment.StatusReversed
		payment.ReversalBatchID = batchID
		payment.ReversedByID = actor.UserID
		payment.ReversedAt = &nowCopy
		payment.ReversalReason = req.Reason
		payment.UpdatedByID = actor.UserID
		updatedPayment, txErr := s.repo.Update(txCtx, payment)
		if txErr != nil {
			return txErr
		}
		payment = updatedPayment
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logAudit(payment, &originalPayment, actor.UserID, permission.OpUpdate, "Customer payment reversed")
	for idx, inv := range invoices {
		s.logInvoiceAudit(originalInvoices[idx], inv, actor.UserID)
	}
	return payment, nil
}

func mapApplications(inputs []*serviceports.CustomerPaymentApplicationInput) []*customerpayment.Application {
	apps := make([]*customerpayment.Application, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			apps = append(apps, nil)
			continue
		}
		apps = append(apps, &customerpayment.Application{LineNumber: idx + 1, InvoiceID: input.InvoiceID, AppliedAmountMinor: input.AppliedAmountMinor})
	}
	return apps
}

func paymentPostingWorkflow(control *tenant.AccountingControl, userID pulid.ID, now int64) (string, string, *int64, pulid.ID, bool, bool, pulid.ID, *int64) {
	entryStatus := "Posted"
	batchStatus := "Posted"
	postedAt := &now
	postedByID := userID
	requiresApproval := false
	isApproved := true
	approvedByID := userID
	approvedAt := &now
	if control != nil && control.JournalPostingMode == tenant.JournalPostingModeManual {
		entryStatus = "Pending"
		batchStatus = "Pending"
		postedAt = nil
		postedByID = pulid.Nil
		requiresApproval = control.RequireManualJEApproval
		isApproved = !control.RequireManualJEApproval
		if !requiresApproval {
			entryStatus = "Approved"
			batchStatus = "Approved"
		} else {
			approvedByID = pulid.Nil
			approvedAt = nil
		}
	}
	return entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt
}

func cloneInvoices(invoices []*invoice.Invoice) []*invoice.Invoice {
	clones := make([]*invoice.Invoice, 0, len(invoices))
	for _, inv := range invoices {
		if inv == nil {
			clones = append(clones, nil)
			continue
		}
		copy := *inv
		clones = append(clones, &copy)
	}
	return clones
}

func (s *Service) logAudit(current *customerpayment.Payment, previous *customerpayment.Payment, userID pulid.ID, operation permission.Operation, comment string) {
	params := &serviceports.LogActionParams{Resource: permission.ResourceCustomerPayment, ResourceID: current.ID.String(), Operation: operation, UserID: userID, CurrentState: jsonutils.MustToJSON(current), OrganizationID: current.OrganizationID, BusinessUnitID: current.BusinessUnitID}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log customer payment audit action", zap.Error(err), zap.String("paymentId", current.ID.String()))
	}
}

func (s *Service) logInvoiceAudit(previous *invoice.Invoice, current *invoice.Invoice, userID pulid.ID) {
	if previous == nil || current == nil {
		return
	}
	params := &serviceports.LogActionParams{Resource: permission.ResourceInvoice, ResourceID: current.ID.String(), Operation: permission.OpUpdate, UserID: userID, CurrentState: jsonutils.MustToJSON(current), PreviousState: jsonutils.MustToJSON(previous), OrganizationID: current.OrganizationID, BusinessUnitID: current.BusinessUnitID}
	if err := s.auditService.LogAction(params, auditservice.WithComment("Invoice updated from customer payment application"), auditservice.WithDiff(previous, current)); err != nil {
		s.l.Error("failed to log invoice payment application audit action", zap.Error(err), zap.String("invoiceId", current.ID.String()))
	}
}
