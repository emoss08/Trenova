//nolint:cyclop // existing legacy workflow/API shape is intentionally kept stable
package customerpaymentservice

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"

	"github.com/emoss08/trenova/internal/core/domain/customerledger"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
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

	Logger             *zap.Logger
	DB                 ports.DBConnection
	Repo               repositories.CustomerPaymentRepository
	InvoiceRepo        repositories.InvoiceRepository
	CustomerLedgerRepo repositories.CustomerLedgerProjectionRepository
	AccountingRepo     repositories.AccountingControlRepository
	JournalRepo        repositories.JournalPostingRepository
	Generator          seqgen.Generator
	Validator          *Validator
	AuditService       serviceports.AuditService
	AccountingSync     serviceports.AccountingSyncEnqueuer `optional:"true"`
}

type Service struct {
	l                  *zap.Logger
	db                 ports.DBConnection
	repo               repositories.CustomerPaymentRepository
	invoiceRepo        repositories.InvoiceRepository
	customerLedgerRepo repositories.CustomerLedgerProjectionRepository
	accountingRepo     repositories.AccountingControlRepository
	journalRepo        repositories.JournalPostingRepository
	generator          seqgen.Generator
	validator          *Validator
	auditService       serviceports.AuditService
	accountingSync     serviceports.AccountingSyncEnqueuer
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:                  p.Logger.Named("service.customer-payment"),
		db:                 p.DB,
		repo:               p.Repo,
		invoiceRepo:        p.InvoiceRepo,
		customerLedgerRepo: p.CustomerLedgerRepo,
		accountingRepo:     p.AccountingRepo,
		journalRepo:        p.JournalRepo,
		generator:          p.Generator,
		validator:          p.Validator,
		auditService:       p.AuditService,
		accountingSync:     p.AccountingSync,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListCustomerPaymentsRequest,
) (*pagination.ListResult[*customerpayment.Payment], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *serviceports.GetCustomerPaymentRequest,
) (*customerpayment.Payment, error) {
	return s.repo.GetByID(
		ctx,
		repositories.GetCustomerPaymentByIDRequest{ID: req.PaymentID, TenantInfo: req.TenantInfo},
	)
}

func (s *Service) PostAndApply( //nolint:funlen,gocognit // legacy workflow
	ctx context.Context,
	req *serviceports.PostCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*customerpayment.Payment, error) {
	plan, err := s.planPost(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	control, entity, invoices, period := plan.control, plan.entity, plan.invoices, plan.period

	batchNumber, err := s.generator.GenerateJournalBatchNumber(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.generator.GenerateJournalEntryNumber(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt := paymentPostingWorkflow(
		control,
		actor.UserID,
		now,
	)
	originalInvoices := cloneInvoices(invoices)
	var created *customerpayment.Payment
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		created, txErr = s.repo.Create(txCtx, entity)
		if txErr != nil {
			return txErr
		}

		for idx, inv := range invoices {
			applyPostedApplication(inv, entity.Applications[idx])
			inv, txErr = s.invoiceRepo.Update(txCtx, inv)
			if txErr != nil {
				return txErr
			}
			invoices[idx] = inv
		}

		batchID := pulid.MustNew("jb_")
		entryID := pulid.MustNew("je_")
		sourceID := pulid.MustNew("jsrc_")
		entryDescription := fmt.Sprintf("Customer payment %s", created.DocumentLabel())
		appliedCreditAccountID := control.DefaultARAccountID
		if control.AccountingBasis == tenant.AccountingBasisCash ||
			control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt {
			appliedCreditAccountID = control.DefaultRevenueAccountID
			entryDescription = fmt.Sprintf("Customer cash receipt %s", created.DocumentLabel())
		}
		lines := make([]repositories.JournalPostingLine, 0, 2+len(created.Applications))
		lines = append(
			lines,
			repositories.JournalPostingLine{
				ID:          pulid.MustNew("jel_"),
				GLAccountID: control.DefaultCashAccountID,
				LineNumber:  1,
				Description: entryDescription,
				DebitAmount: created.AmountMinor,
				NetAmount:   created.AmountMinor,
				CustomerID:  entity.CustomerID,
			},
		)
		lineNumber := int16(2)
		for _, app := range created.Applications {
			if app == nil || app.AppliedAmountMinor <= 0 {
				if app == nil || app.ShortPayAmountMinor <= 0 {
					continue
				}
			}
			if app.AppliedAmountMinor > 0 {
				lines = append(
					lines,
					repositories.JournalPostingLine{
						ID:           pulid.MustNew("jel_"),
						GLAccountID:  appliedCreditAccountID,
						LineNumber:   lineNumber,
						Description:  entryDescription,
						CreditAmount: app.AppliedAmountMinor,
						NetAmount:    -app.AppliedAmountMinor,
						CustomerID:   entity.CustomerID,
					},
				)
				lineNumber++
			}
			if app.ShortPayAmountMinor > 0 {
				lines = append(
					lines,
					repositories.JournalPostingLine{
						ID:           pulid.MustNew("jel_"),
						GLAccountID:  control.DefaultARAccountID,
						LineNumber:   lineNumber,
						Description:  entryDescription,
						CreditAmount: app.ShortPayAmountMinor,
						NetAmount:    -app.ShortPayAmountMinor,
						CustomerID:   entity.CustomerID,
					},
				)
				lineNumber++
				lines = append(
					lines,
					repositories.JournalPostingLine{
						ID:          pulid.MustNew("jel_"),
						GLAccountID: control.DefaultWriteOffAccountID,
						LineNumber:  lineNumber,
						Description: entryDescription,
						DebitAmount: app.ShortPayAmountMinor,
						NetAmount:   app.ShortPayAmountMinor,
						CustomerID:  entity.CustomerID,
					},
				)
				lineNumber++
			}
		}
		if created.UnappliedAmountMinor > 0 {
			lines = append(
				lines,
				repositories.JournalPostingLine{
					ID:           pulid.MustNew("jel_"),
					GLAccountID:  control.DefaultUnappliedCashAccountID,
					LineNumber:   lineNumber,
					Description:  entryDescription,
					CreditAmount: created.UnappliedAmountMinor,
					NetAmount:    -created.UnappliedAmountMinor,
					CustomerID:   entity.CustomerID,
				},
			)
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
		if s.customerLedgerRepo != nil {
			ledgerEntries := make(
				[]*customerledger.CustomerLedgerEntry,
				0,
				len(created.Applications)*2,
			)
			line := 1
			for _, app := range created.Applications {
				if app == nil {
					continue
				}
				if app.AppliedAmountMinor > 0 {
					ledgerEntries = append(
						ledgerEntries,
						&customerledger.CustomerLedgerEntry{
							ID:               pulid.MustNew("cledg_"),
							OrganizationID:   created.OrganizationID,
							BusinessUnitID:   created.BusinessUnitID,
							CustomerID:       created.CustomerID,
							SourceObjectType: "CustomerPayment",
							SourceObjectID:   created.ID.String(),
							SourceEventType:  tenant.JournalSourceEventCustomerPaymentPosted.String(),
							RelatedInvoiceID: app.InvoiceID,
							DocumentNumber:   created.ReferenceNumber,
							TransactionDate:  created.AccountingDate,
							LineNumber:       line,
							AmountMinor:      -app.AppliedAmountMinor,
							CreatedByID:      actor.UserID,
						},
					)
					line++
				}
				if app.ShortPayAmountMinor > 0 {
					ledgerEntries = append(
						ledgerEntries,
						&customerledger.CustomerLedgerEntry{
							ID:               pulid.MustNew("cledg_"),
							OrganizationID:   created.OrganizationID,
							BusinessUnitID:   created.BusinessUnitID,
							CustomerID:       created.CustomerID,
							SourceObjectType: "CustomerPayment",
							SourceObjectID:   created.ID.String(),
							SourceEventType:  tenant.JournalSourceEventCustomerShortPayRecognized.String(),
							RelatedInvoiceID: app.InvoiceID,
							DocumentNumber:   created.ReferenceNumber,
							TransactionDate:  created.AccountingDate,
							LineNumber:       line,
							AmountMinor:      -app.ShortPayAmountMinor,
							CreatedByID:      actor.UserID,
						},
					)
					line++
				}
			}
			if txErr = s.customerLedgerRepo.AppendEntries(txCtx, ledgerEntries); txErr != nil {
				return txErr
			}
		}

		created.PostedBatchID = batchID
		created.UpdatedByID = actor.UserID
		if created, txErr = s.repo.Update(txCtx, created); txErr != nil {
			return txErr
		}
		return serviceports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			serviceports.PaymentSyncRequest(
				created,
				accountingsync.SyncOperationCreate,
				accountingsync.SyncSourceCustomerPaymentPosted,
			),
		)
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

func (s *Service) ApplyUnapplied( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *serviceports.ApplyCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*customerpayment.Payment, error) {
	plan, err := s.planApplyUnapplied(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	payment, control, applications, invoices, period := plan.payment, plan.control, plan.applications, plan.invoices, plan.period
	entryNumber, err := s.generator.GenerateJournalEntryNumber(
		ctx,
		payment.OrganizationID,
		payment.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	batchNumber, err := s.generator.GenerateJournalBatchNumber(
		ctx,
		payment.OrganizationID,
		payment.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()
	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt := paymentPostingWorkflow(
		control,
		actor.UserID,
		now,
	)
	originalPayment := *payment
	originalInvoices := cloneInvoices(invoices)
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		payment.Applications = append(payment.Applications, applications...)
		payment.UpdatedByID = actor.UserID
		for idx, inv := range invoices {
			applyPostedApplication(inv, applications[idx])
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

		journal := applicationJournal(
			control,
			payment,
			payment.AppliedAmountMinor-originalPayment.AppliedAmountMinor,
			applications,
		)
		if txErr = s.journalRepo.CreatePosting(
			txCtx,
			repositories.CreateJournalPostingParams{
				BatchID:              pulid.MustNew("jb_"),
				OrganizationID:       payment.OrganizationID,
				BusinessUnitID:       payment.BusinessUnitID,
				BatchNumber:          batchNumber,
				BatchType:            "System",
				BatchStatus:          batchStatus,
				BatchDescription:     journal.description,
				FiscalYearID:         period.FiscalYearID,
				FiscalPeriodID:       period.ID,
				AccountingDate:       req.AccountingDate,
				PostedAt:             postedAt,
				PostedByID:           postedByID,
				CreatedByID:          actor.UserID,
				UpdatedByID:          actor.UserID,
				EntryID:              pulid.MustNew("je_"),
				EntryNumber:          entryNumber,
				EntryType:            "Standard",
				EntryStatus:          entryStatus,
				ReferenceNumber:      payment.ReferenceNumber,
				ReferenceType:        "CustomerPaymentApplied",
				ReferenceID:          payment.ID.String(),
				EntryDescription:     journal.description,
				TotalDebit:           journal.total,
				TotalCredit:          journal.total,
				IsPosted:             postedAt != nil,
				IsAutoGenerated:      false,
				RequiresApproval:     requiresApproval,
				IsApproved:           isApproved,
				ApprovedByID:         approvedByID,
				ApprovedAt:           approvedAt,
				SourceID:             pulid.MustNew("jsrc_"),
				SourceObjectType:     "CustomerPayment",
				SourceObjectID:       payment.ID.String(),
				SourceEventType:      "CustomerPaymentApplied",
				SourceStatus:         entryStatus,
				SourceDocumentNumber: payment.ReferenceNumber,
				SourceIdempotencyKey: "customer-payment-applied:" + payment.ID.String() + ":" + strconv.FormatInt(
					payment.AppliedAmountMinor,
					10,
				),
				Lines: journal.lines,
			},
		); txErr != nil {
			return txErr
		}
		if s.customerLedgerRepo != nil {
			ledgerEntries := make([]*customerledger.CustomerLedgerEntry, 0, len(applications)*2)
			projectionLine := 1
			for _, app := range applications {
				if app == nil {
					continue
				}
				if app.AppliedAmountMinor > 0 {
					ledgerEntries = append(
						ledgerEntries,
						&customerledger.CustomerLedgerEntry{
							ID:               pulid.MustNew("cledg_"),
							OrganizationID:   payment.OrganizationID,
							BusinessUnitID:   payment.BusinessUnitID,
							CustomerID:       payment.CustomerID,
							SourceObjectType: "CustomerPayment",
							SourceObjectID:   payment.ID.String(),
							SourceEventType:  "CustomerPaymentApplied",
							RelatedInvoiceID: app.InvoiceID,
							DocumentNumber:   payment.ReferenceNumber,
							TransactionDate:  req.AccountingDate,
							LineNumber:       projectionLine,
							AmountMinor:      -app.AppliedAmountMinor,
							CreatedByID:      actor.UserID,
						},
					)
					projectionLine++
				}
				if app.ShortPayAmountMinor > 0 {
					ledgerEntries = append(
						ledgerEntries,
						&customerledger.CustomerLedgerEntry{
							ID:               pulid.MustNew("cledg_"),
							OrganizationID:   payment.OrganizationID,
							BusinessUnitID:   payment.BusinessUnitID,
							CustomerID:       payment.CustomerID,
							SourceObjectType: "CustomerPayment",
							SourceObjectID:   payment.ID.String(),
							SourceEventType:  tenant.JournalSourceEventCustomerShortPayRecognized.String(),
							RelatedInvoiceID: app.InvoiceID,
							DocumentNumber:   payment.ReferenceNumber,
							TransactionDate:  req.AccountingDate,
							LineNumber:       projectionLine,
							AmountMinor:      -app.ShortPayAmountMinor,
							CreatedByID:      actor.UserID,
						},
					)
					projectionLine++
				}
			}
			if txErr = s.customerLedgerRepo.AppendEntries(txCtx, ledgerEntries); txErr != nil {
				return txErr
			}
		}
		return serviceports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			serviceports.PaymentSyncRequest(
				payment,
				accountingsync.SyncOperationUpdate,
				accountingsync.SyncSourceCustomerPaymentApplied,
			),
		)
	})
	if err != nil {
		return nil, err
	}
	s.logAudit(
		payment,
		&originalPayment,
		actor.UserID,
		permission.OpUpdate,
		"Customer payment unapplied amount applied",
	)
	for idx, inv := range invoices {
		s.logInvoiceAudit(originalInvoices[idx], inv, actor.UserID)
	}
	return payment, nil
}

func (s *Service) Reverse( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *serviceports.ReverseCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*customerpayment.Payment, error) {
	plan, err := s.planReverse(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	payment, control, invoices, period := plan.payment, plan.control, plan.invoices, plan.period
	batchNumber, err := s.generator.GenerateJournalBatchNumber(
		ctx,
		payment.OrganizationID,
		payment.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.generator.GenerateJournalEntryNumber(
		ctx,
		payment.OrganizationID,
		payment.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()
	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt := paymentPostingWorkflow(
		control,
		actor.UserID,
		now,
	)
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

		journal := reversalJournal(control, payment)
		batchID := pulid.MustNew("jb_")
		if txErr := s.journalRepo.CreatePosting(
			txCtx,
			repositories.CreateJournalPostingParams{
				BatchID:              batchID,
				OrganizationID:       payment.OrganizationID,
				BusinessUnitID:       payment.BusinessUnitID,
				BatchNumber:          batchNumber,
				BatchType:            "Reversal",
				BatchStatus:          batchStatus,
				BatchDescription:     journal.description,
				FiscalYearID:         period.FiscalYearID,
				FiscalPeriodID:       period.ID,
				AccountingDate:       req.AccountingDate,
				PostedAt:             postedAt,
				PostedByID:           postedByID,
				CreatedByID:          actor.UserID,
				UpdatedByID:          actor.UserID,
				EntryID:              pulid.MustNew("je_"),
				EntryNumber:          entryNumber,
				EntryType:            "Reversal",
				EntryStatus:          entryStatus,
				ReferenceNumber:      payment.ReferenceNumber,
				ReferenceType:        tenant.JournalSourceEventCustomerPaymentReversed.String(),
				ReferenceID:          payment.ID.String(),
				EntryDescription:     journal.description,
				TotalDebit:           journal.total,
				TotalCredit:          journal.total,
				IsPosted:             postedAt != nil,
				IsAutoGenerated:      false,
				ReversalDate:         postedAt,
				ReversalReason:       req.Reason,
				RequiresApproval:     requiresApproval,
				IsApproved:           isApproved,
				ApprovedByID:         approvedByID,
				ApprovedAt:           approvedAt,
				SourceID:             pulid.MustNew("jsrc_"),
				SourceObjectType:     "CustomerPayment",
				SourceObjectID:       payment.ID.String(),
				SourceEventType:      tenant.JournalSourceEventCustomerPaymentReversed.String(),
				SourceStatus:         entryStatus,
				SourceDocumentNumber: payment.ReferenceNumber,
				SourceIdempotencyKey: "customer-payment-reversed:" + payment.ID.String(),
				Lines:                journal.lines,
			},
		); txErr != nil {
			return txErr
		}
		if s.customerLedgerRepo != nil {
			reverseAmount := payment.AppliedAmountMinor
			for _, app := range payment.Applications {
				if app != nil {
					reverseAmount += app.ShortPayAmountMinor
				}
			}
			if txErr := s.customerLedgerRepo.AppendEntries(
				txCtx,
				[]*customerledger.CustomerLedgerEntry{
					{
						ID:               pulid.MustNew("cledg_"),
						OrganizationID:   payment.OrganizationID,
						BusinessUnitID:   payment.BusinessUnitID,
						CustomerID:       payment.CustomerID,
						SourceObjectType: "CustomerPayment",
						SourceObjectID:   payment.ID.String(),
						SourceEventType:  tenant.JournalSourceEventCustomerPaymentReversed.String(),
						DocumentNumber:   payment.ReferenceNumber,
						TransactionDate:  req.AccountingDate,
						LineNumber:       1,
						AmountMinor:      reverseAmount,
						CreatedByID:      actor.UserID,
					},
				},
			); txErr != nil {
				return txErr
			}
		}

		markReversed(payment, req, actor.UserID, now)
		payment.ReversalBatchID = batchID
		updatedPayment, txErr := s.repo.Update(txCtx, payment)
		if txErr != nil {
			return txErr
		}
		payment = updatedPayment
		return serviceports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			serviceports.PaymentSyncRequest(
				payment,
				accountingsync.SyncOperationVoid,
				accountingsync.SyncSourceCustomerPaymentReversed,
			),
		)
	})
	if err != nil {
		return nil, err
	}
	s.logAudit(
		payment,
		&originalPayment,
		actor.UserID,
		permission.OpUpdate,
		"Customer payment reversed",
	)
	for idx, inv := range invoices {
		s.logInvoiceAudit(originalInvoices[idx], inv, actor.UserID)
	}
	return payment, nil
}

func mapApplications(
	inputs []*serviceports.CustomerPaymentApplicationInput,
) []*customerpayment.Application {
	apps := make([]*customerpayment.Application, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			apps = append(apps, nil)
			continue
		}
		apps = append(
			apps,
			&customerpayment.Application{
				LineNumber:         idx + 1,
				InvoiceID:          input.InvoiceID,
				AppliedAmountMinor: input.AppliedAmountMinor,
			},
		)
		apps[len(apps)-1].ShortPayAmountMinor = input.ShortPayAmountMinor
	}
	return apps
}

func paymentPostingWorkflow( //nolint:gocritic // stable API shape
	control *tenant.AccountingControl,
	userID pulid.ID,
	now int64,
) (string, string, *int64, pulid.ID, bool, bool, pulid.ID, *int64) {
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
		invoiceCopy := *inv
		clones = append(clones, &invoiceCopy)
	}
	return clones
}

func (s *Service) logAudit(
	current *customerpayment.Payment,
	previous *customerpayment.Payment,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceCustomerPayment,
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
			"failed to log customer payment audit action",
			zap.Error(err),
			zap.String("paymentId", current.ID.String()),
		)
	}
}

func (s *Service) logInvoiceAudit(
	previous *invoice.Invoice,
	current *invoice.Invoice,
	userID pulid.ID,
) {
	if previous == nil || current == nil {
		return
	}
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceInvoice,
		ResourceID:     current.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		PreviousState:  jsonutils.MustToJSON(previous),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if err := s.auditService.LogAction(
		params,
		auditservice.WithComment("Invoice updated from customer payment application"),
		auditservice.WithDiff(previous, current),
	); err != nil {
		s.l.Error(
			"failed to log invoice payment application audit action",
			zap.Error(err),
			zap.String("invoiceId", current.ID.String()),
		)
	}
}
