//nolint:cyclop // existing legacy workflow/API shape is intentionally kept stable
package customerpaymentservice

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	DB                 kernelports.Transactor
	Repo               customerpaymentports.CustomerPaymentRepository
	InvoiceRepo        invoiceports.InvoiceRepository
	CustomerLedgerRepo customerledgerports.CustomerLedgerProjectionRepository
	AccountingRepo     tenantports.AccountingControlRepository
	Stamper            *exchangeratestamp.Stamper
	JournalRepo        journalentryports.JournalPostingRepository
	Generator          seqgen.Generator
	Validator          *Validator
	AuditService       auditports.AuditService
	AccountingSync     accountingsyncports.AccountingSyncEnqueuer `optional:"true"`
	Realtime           agentports.RealtimeService                 `optional:"true"`
	Triage             inboundmessageports.TriageEventSink        `optional:"true"`
	ReceiptMatches     bankreceiptports.BankReceiptService        `optional:"true"`
}

type Service struct {
	l                  *zap.Logger
	db                 kernelports.Transactor
	repo               customerpaymentports.CustomerPaymentRepository
	invoiceRepo        invoiceports.InvoiceRepository
	customerLedgerRepo customerledgerports.CustomerLedgerProjectionRepository
	accountingRepo     tenantports.AccountingControlRepository
	stamper            *exchangeratestamp.Stamper
	journalRepo        journalentryports.JournalPostingRepository
	generator          seqgen.Generator
	validator          *Validator
	auditService       auditports.AuditService
	accountingSync     accountingsyncports.AccountingSyncEnqueuer
	realtime           agentports.RealtimeService
	triage             inboundmessageports.TriageEventSink
	receiptMatches     receiptMatchReleaser
}

// receiptMatchReleaser hands back the deposits a reversed payment was
// reconciled against.
type receiptMatchReleaser interface {
	ReleasePaymentMatch(
		ctx context.Context,
		req *bankreceiptports.ReleasePaymentMatchRequest,
		actor *permissionports.RequestActor,
	) ([]*bankreceipt.BankReceipt, error)
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	svc := &Service{
		l:                  p.Logger.Named("service.customer-payment"),
		db:                 p.DB,
		repo:               p.Repo,
		invoiceRepo:        p.InvoiceRepo,
		customerLedgerRepo: p.CustomerLedgerRepo,
		accountingRepo:     p.AccountingRepo,
		stamper:            p.Stamper,
		journalRepo:        p.JournalRepo,
		generator:          p.Generator,
		validator:          p.Validator,
		auditService:       p.AuditService,
		accountingSync:     p.AccountingSync,
		realtime:           p.Realtime,
		triage:             p.Triage,
	}
	if p.ReceiptMatches != nil {
		svc.receiptMatches = p.ReceiptMatches
	}

	return svc
}

func (s *Service) List(
	ctx context.Context,
	req *customerpaymentports.ListCustomerPaymentsRequest,
) (*pagination.ListResult[*customerpayment.Payment], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *customerpaymentports.GetCustomerPaymentRequest,
) (*customerpayment.Payment, error) {
	return s.repo.GetByID(
		ctx,
		customerpaymentports.GetCustomerPaymentByIDRequest{
			ID:         req.PaymentID,
			TenantInfo: req.TenantInfo,
		},
	)
}

func (s *Service) PostAndApply( //nolint:funlen,gocognit // legacy workflow
	ctx context.Context,
	req *customerpaymentports.PostCustomerPaymentRequest,
	actor *permissionports.RequestActor,
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
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
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
		lines := make([]journalentryports.JournalPostingLine, 0, 2+len(created.Applications))
		lines = append(
			lines,
			journalentryports.JournalPostingLine{
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
					journalentryports.JournalPostingLine{
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
					journalentryports.JournalPostingLine{
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
					journalentryports.JournalPostingLine{
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
				journalentryports.JournalPostingLine{
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

		if txErr = s.journalRepo.CreatePosting(txCtx, journalentryports.CreateJournalPostingParams{
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
			TotalDebit:           created.AmountMinor + created.ShortPayAmountMinor(),
			TotalCredit:          created.AmountMinor + created.ShortPayAmountMinor(),
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
			if txErr = s.customerLedgerRepo.AppendEntries(
				txCtx,
				applicationLedgerEntries(&applicationLedgerParams{
					payment:         created,
					applications:    created.Applications,
					appliedEvent:    tenant.JournalSourceEventCustomerPaymentPosted.String(),
					transactionDate: created.AccountingDate,
					actorID:         actor.UserID,
				}),
			); txErr != nil {
				return txErr
			}
		}

		created.PostedBatchID = batchID
		created.UpdatedByID = actor.UserID
		if created, txErr = s.repo.Update(txCtx, created); txErr != nil {
			return txErr
		}
		return accountingsyncports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			accountingsyncports.PaymentSyncRequest(
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
		s.announceInvoice(ctx, inv, actor)
	}
	return created, nil
}

func (s *Service) ApplyUnapplied( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *customerpaymentports.ApplyCustomerPaymentRequest,
	actor *permissionports.RequestActor,
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
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
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
			journalentryports.CreateJournalPostingParams{
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
				SourceEventType:      customerledger.EventPaymentApplied,
				SourceStatus:         entryStatus,
				SourceDocumentNumber: payment.ReferenceNumber,
				SourceIdempotencyKey: "customer-payment-applied:" + payment.ID.String() + ":" + strconv.FormatInt(
					payment.Version,
					10,
				),
				Lines: journal.lines,
			},
		); txErr != nil {
			return txErr
		}
		if s.customerLedgerRepo != nil {
			if txErr = s.customerLedgerRepo.AppendEntries(
				txCtx,
				applicationLedgerEntries(&applicationLedgerParams{
					payment:         payment,
					applications:    applications,
					appliedEvent:    customerledger.EventPaymentApplied,
					transactionDate: req.AccountingDate,
					actorID:         actor.UserID,
				}),
			); txErr != nil {
				return txErr
			}
		}
		return accountingsyncports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			accountingsyncports.PaymentSyncRequest(
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
		s.announceInvoice(ctx, inv, actor)
	}
	return payment, nil
}

func (s *Service) Reverse( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *customerpaymentports.ReverseCustomerPaymentRequest,
	actor *permissionports.RequestActor,
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
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		for idx, app := range payment.Applications {
			if app == nil {
				continue
			}
			invoices[idx].RemovePaymentMinor(app.SettledAmountMinor())
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
			journalentryports.CreateJournalPostingParams{
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
		if s.receiptMatches != nil {
			if _, txErr := s.receiptMatches.ReleasePaymentMatch(
				txCtx,
				&bankreceiptports.ReleasePaymentMatchRequest{
					PaymentID:        payment.ID,
					PaymentReference: payment.DocumentLabel(),
					Reason:           req.Reason,
					TenantInfo:       req.TenantInfo,
				},
				actor,
			); txErr != nil {
				return txErr
			}
		}
		if s.customerLedgerRepo != nil {
			if txErr := s.customerLedgerRepo.AppendEntries(
				txCtx,
				applicationLedgerEntries(&applicationLedgerParams{
					payment:         payment,
					applications:    payment.Applications,
					transactionDate: req.AccountingDate,
					actorID:         actor.UserID,
					reversal:        true,
				}),
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
		return accountingsyncports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			accountingsyncports.PaymentSyncRequest(
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
		s.announceInvoice(ctx, inv, actor)
	}
	return payment, nil
}

func mapApplications(
	inputs []*customerpaymentports.CustomerPaymentApplicationInput,
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
	w := journalposting.ResolveWorkflow(control, userID, now)
	return w.EntryStatus, w.BatchStatus, w.PostedAt, w.PostedByID, w.RequiresApproval,
		w.IsApproved, w.ApprovedByID, w.ApprovedAt
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
	params := &auditports.LogActionParams{
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
	options := []auditports.LogOption{auditservice.WithComment(comment)}
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

// announceInvoice tells every screen showing the invoice that a payment
// moved its balance: the register, the invoice, and a Desk case about it,
// which settles once it is paid.
func (s *Service) announceInvoice(
	ctx context.Context,
	inv *invoice.Invoice,
	actor *permissionports.RequestActor,
) {
	if inv == nil {
		return
	}
	s.wakeTriageOnPaid(ctx, inv)
	if s.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: inv.OrganizationID,
		BusinessUnitID: inv.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceInvoice.String(),
		Action:         "updated",
		RecordID:       inv.ID,
		Fields:         []string{"appliedAmount", "settlementStatus"},
	}); err != nil {
		s.l.Warn("failed to publish invoice invalidation", zap.Error(err))
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
	params := &auditports.LogActionParams{
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

// applicationLedgerParams describes the subledger entries for a payment's
// applications. Each entry names the invoice it settles, so anything that
// reads the ledger per invoice sees the payment against the right invoice.
// A reversal writes the same entries with the sign flipped.
type applicationLedgerParams struct {
	payment         *customerpayment.Payment
	applications    []*customerpayment.Application
	appliedEvent    string
	transactionDate int64
	actorID         pulid.ID
	reversal        bool
}

func applicationLedgerEntries(
	params *applicationLedgerParams,
) []*customerledger.CustomerLedgerEntry {
	entries := make([]*customerledger.CustomerLedgerEntry, 0, len(params.applications)*2)
	appendEntry := func(app *customerpayment.Application, event string, amount int64) {
		amountMinor := -amount
		if params.reversal {
			event = tenant.JournalSourceEventCustomerPaymentReversed.String()
			amountMinor = amount
		}
		entries = append(entries, &customerledger.CustomerLedgerEntry{
			ID:               pulid.MustNew("cledg_"),
			OrganizationID:   params.payment.OrganizationID,
			BusinessUnitID:   params.payment.BusinessUnitID,
			CustomerID:       params.payment.CustomerID,
			SourceObjectType: "CustomerPayment",
			SourceObjectID:   params.payment.ID.String(),
			SourceEventType:  event,
			RelatedInvoiceID: app.InvoiceID,
			DocumentNumber:   params.payment.ReferenceNumber,
			TransactionDate:  params.transactionDate,
			LineNumber:       len(entries) + 1,
			AmountMinor:      amountMinor,
			CreatedByID:      params.actorID,
		})
	}
	for _, app := range params.applications {
		if app == nil {
			continue
		}
		if app.AppliedAmountMinor > 0 {
			appendEntry(app, params.appliedEvent, app.AppliedAmountMinor)
		}
		if app.ShortPayAmountMinor > 0 {
			appendEntry(
				app,
				tenant.JournalSourceEventCustomerShortPayRecognized.String(),
				app.ShortPayAmountMinor,
			)
		}
	}
	return entries
}

func (s *Service) wakeTriageOnPaid(ctx context.Context, inv *invoice.Invoice) {
	if s.triage == nil || inv.SettlementStatus != invoice.SettlementStatusPaid {
		return
	}

	tenantInfo := pagination.TenantInfo{OrgID: inv.OrganizationID, BuID: inv.BusinessUnitID}
	for _, shipmentID := range inv.LegShipmentIDs() {
		s.triage.InvoicePaid(ctx, tenantInfo, shipmentID)
	}
}
