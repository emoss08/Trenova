package invoiceadjustmentservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/temporaljobs/invoiceadjustmentjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/billingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/billingqueuerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/customerrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/documentrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoiceadjustmentcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoiceadjustmentrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoicerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalpostingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/m2msync"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentrepository"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type integrationHarness struct {
	ctx                context.Context
	db                 *bun.DB
	conn               *postgres.Connection
	service            servicesports.InvoiceAdjustmentService
	customerRepo       repositories.CustomerRepository
	invoiceRepo        repositories.InvoiceRepository
	adjustmentRepo     repositories.InvoiceAdjustmentRepository
	billingQueueRepo   repositories.BillingQueueRepository
	adjustmentCtrlRepo repositories.InvoiceAdjustmentControlRepository
	accountingRepo     repositories.AccountingControlRepository
	orgID              pulid.ID
	buID               pulid.ID
	userID             pulid.ID
	customerID         pulid.ID
	customerName       string
	customerCode       string
	customerAddress1   string
	customerCity       string
	customerState      string
	customerPostal     string
	customerCountry    string
	shipmentID         pulid.ID
	shipmentPro        string
	shipmentBOL        string
	nextNumber         int
}

type seededOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type seededUser struct {
	ID pulid.ID `bun:"id"`
}

type seededCustomer struct {
	ID           pulid.ID `bun:"id"`
	Name         string   `bun:"name"`
	Code         string   `bun:"code"`
	AddressLine1 string   `bun:"address_line_1"`
	City         string   `bun:"city"`
	PostalCode   string   `bun:"postal_code"`
}

type seededShipment struct {
	ID        pulid.ID `bun:"id"`
	ProNumber string   `bun:"pro_number"`
	BOL       string   `bun:"bol"`
}

type glAccountRow struct {
	ID pulid.ID `bun:"id"`
}

type journalEntryRow struct {
	ID          pulid.ID `bun:"id"`
	Status      string   `bun:"status"`
	TotalDebit  int64    `bun:"total_debit"`
	TotalCredit int64    `bun:"total_credit"`
}

type journalEntryLineRow struct {
	GLAccountID  pulid.ID `bun:"gl_account_id"`
	DebitAmount  int64    `bun:"debit_amount"`
	CreditAmount int64    `bun:"credit_amount"`
}

type fakeGenerator struct {
	next int
}

func (g *fakeGenerator) GenerateInvoiceNumber(context.Context, pulid.ID, pulid.ID, string, string) (string, error) {
	g.next++
	return fmt.Sprintf("REBILL-%03d", g.next), nil
}

func (g *fakeGenerator) GenerateCreditMemoNumber(context.Context, pulid.ID, pulid.ID, string, string) (string, error) {
	g.next++
	return fmt.Sprintf("CM-%03d", g.next), nil
}

type fakeFormulaCalculator struct {
	amount decimal.Decimal
}

func (f *fakeFormulaCalculator) Calculate(_ context.Context, req *formulatemplatetypes.CalculateRequest) (*formulatemplatetypes.CalculateResponse, error) {
	return &formulatemplatetypes.CalculateResponse{
		Amount:              f.amount,
		Variables:           map[string]any{},
		FormulaTemplateID:   req.TemplateID.String(),
		FormulaTemplateName: "test-formula",
		Expression:          "flat",
	}, nil
}

type fakeAccessorialRepo struct{}

func (fakeAccessorialRepo) List(context.Context, *repositories.ListAccessorialChargeRequest) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	return nil, nil
}
func (fakeAccessorialRepo) SelectOptions(context.Context, *pagination.SelectQueryRequest) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	return nil, nil
}
func (fakeAccessorialRepo) GetByID(context.Context, repositories.GetAccessorialChargeByIDRequest) (*accessorialcharge.AccessorialCharge, error) {
	return nil, nil
}
func (fakeAccessorialRepo) Create(context.Context, *accessorialcharge.AccessorialCharge) (*accessorialcharge.AccessorialCharge, error) {
	return nil, nil
}
func (fakeAccessorialRepo) Update(context.Context, *accessorialcharge.AccessorialCharge) (*accessorialcharge.AccessorialCharge, error) {
	return nil, nil
}

type noopAuditService struct{}

func (noopAuditService) List(context.Context, *repositories.ListAuditEntriesRequest) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}
func (noopAuditService) ListByResourceID(context.Context, *repositories.ListByResourceIDRequest) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}
func (noopAuditService) GetByID(context.Context, repositories.GetAuditEntryByIDOptions) (*audit.Entry, error) {
	return nil, nil
}
func (noopAuditService) LogAction(*servicesports.LogActionParams, ...servicesports.LogOption) error {
	return nil
}
func (noopAuditService) LogActions([]servicesports.BulkLogEntry) error { return nil }
func (noopAuditService) RegisterSensitiveFields(permission.Resource, []servicesports.SensitiveField) error {
	return nil
}

type fakeWorkflowRun struct {
	id    string
	runID string
}

func (f fakeWorkflowRun) GetID() string                  { return f.id }
func (f fakeWorkflowRun) GetRunID() string               { return f.runID }
func (f fakeWorkflowRun) Get(context.Context, any) error { return nil }
func (f fakeWorkflowRun) GetWithOptions(context.Context, any, client.WorkflowRunGetOptions) error {
	return nil
}

type fakeWorkflowStarter struct {
	enabled bool
	calls   []workflowCall
}

type workflowCall struct {
	options  client.StartWorkflowOptions
	workflow any
	args     []any
}

func (f *fakeWorkflowStarter) StartWorkflow(_ context.Context, options client.StartWorkflowOptions, workflow any, args ...any) (client.WorkflowRun, error) {
	f.calls = append(f.calls, workflowCall{options: options, workflow: workflow, args: args})
	return fakeWorkflowRun{id: options.ID, runID: "run-1"}, nil
}

func (f *fakeWorkflowStarter) Enabled() bool { return f.enabled }

func TestInvoiceAdjustmentService_EngineScenarios(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{
		System: config.SystemConfig{
			SystemUserPassword: "test-system-password",
		},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	h := newIntegrationHarness(t, ctx, db, &fakeWorkflowStarter{enabled: false}, decimal.NewFromInt(100))
	h.ensureOpenFiscalPeriod(t)
	h.ensureAccountingDefaults(t)
	h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
		control.AdjustmentAttachmentRequirement = tenant.AdjustmentAttachmentPolicyOptional
		control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
	})

	t.Run("full invoice credit only executes credit memo", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = tenant.ApprovalPolicyAmountThreshold
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Base freight", 1, 100),
			makeInvoiceLine(2, invoice.LineTypeAccessorial, "Fuel", 1, 25),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		adjustment, submitErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "credit-only-" + entity.ID.String(),
			Reason:         "Customer concession",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusExecuted, adjustment.Status)
		require.True(t, adjustment.ReplacementInvoiceID.IsNil())
		require.True(t, adjustment.RebillQueueItemID.IsNil())
	})

	t.Run("credit and rebill creates draft replacement invoice and lineage", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
			control.ReplacementInvoiceReviewPolicy = tenant.ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange
		})

		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Base freight", 1, 100),
			makeInvoiceLine(2, invoice.LineTypeAccessorial, "Fuel", 1, 50),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		adjustment, submitErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditRebill,
			RebillStrategy: invoiceadjustment.RebillStrategyManual,
			IdempotencyKey: "credit-rebill-" + entity.ID.String(),
			Reason:         "Rate correction",
			TenantInfo:     h.tenantInfo(),
			Lines: []*servicesports.InvoiceAdjustmentLineInput{
				{
					OriginalLineID: entity.Lines[0].ID,
					CreditQuantity: decimal.NewFromInt(1),
					CreditAmount:   decimal.NewFromInt(100),
					RebillQuantity: decimal.NewFromInt(1),
					RebillAmount:   decimal.NewFromInt(110),
				},
				{
					OriginalLineID: entity.Lines[1].ID,
					CreditQuantity: decimal.NewFromInt(1),
					CreditAmount:   decimal.NewFromInt(50),
					RebillQuantity: decimal.NewFromInt(1),
					RebillAmount:   decimal.NewFromInt(60),
				},
			},
		}, h.actor())
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusExecuted, adjustment.Status)
		require.True(t, adjustment.CreditMemoInvoiceID.IsNotNil())
		require.True(t, adjustment.ReplacementInvoiceID.IsNotNil())
		require.True(t, adjustment.RebillQueueItemID.IsNotNil())

		replacement, getErr := h.invoiceRepo.GetByID(h.ctx, repositories.GetInvoiceByIDRequest{
			ID:         adjustment.ReplacementInvoiceID,
			TenantInfo: h.tenantInfo(),
		})
		require.NoError(t, getErr)
		assert.Equal(t, invoice.StatusDraft, replacement.Status)
		assert.Equal(t, entity.ID, replacement.SupersedesInvoiceID)

		queueItem, queueErr := h.billingQueueRepo.GetByID(h.ctx, &repositories.GetBillingQueueItemByIDRequest{
			ItemID:     adjustment.RebillQueueItemID,
			TenantInfo: h.tenantInfo(),
		})
		require.NoError(t, queueErr)
		assert.Equal(t, billingqueue.StatusReadyForReview, queueItem.Status)
		require.NotNil(t, queueItem.SourceCreditMemoInvoiceID)
		assert.Equal(t, adjustment.CreditMemoInvoiceID, *queueItem.SourceCreditMemoInvoiceID)
	})

	t.Run("write-off uses write-off approval policy and creates journal entry", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.WriteOffApprovalPolicy = tenant.WriteOffApprovalPolicyRequireApprovalAboveThreshold
			control.WriteOffApprovalThreshold = decimal.NewFromInt(50)
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Open balance", 1, 80),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		pending, submitErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindWriteOff,
			IdempotencyKey: "writeoff-" + entity.ID.String(),
			Reason:         "Bad debt",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusPendingApproval, pending.Status)

		approved, approveErr := h.service.Approve(h.ctx, &servicesports.ApproveInvoiceAdjustmentRequest{
			AdjustmentID: pending.ID,
			TenantInfo:   h.tenantInfo(),
		}, h.actor())
		require.NoError(t, approveErr)
		require.Equal(t, invoiceadjustment.StatusExecuted, approved.Status)
		assert.Contains(t, approved.Metadata, "writeOffJournalEntryId")

		var entry journalEntryRow
		require.NoError(t, h.db.NewSelect().
			Table("journal_entries").
			Column("id", "status", "total_debit", "total_credit").
			Where("reference_id = ?", approved.ID.String()).
			Limit(1).
			Scan(h.ctx, &entry))
		assert.Equal(t, "Posted", entry.Status)
		assert.Equal(t, int64(8000), entry.TotalDebit)
		assert.Equal(t, int64(8000), entry.TotalCredit)

		lines := make([]journalEntryLineRow, 0, 2)
		require.NoError(t, h.db.NewSelect().
			Table("journal_entry_lines").
			Column("gl_account_id", "debit_amount", "credit_amount").
			Where("journal_entry_id = ?", entry.ID).
			OrderExpr("line_number ASC").
			Scan(h.ctx, &lines))
		require.Len(t, lines, 2)
		assert.Equal(t, h.lookupGLAccount(t, "6940"), lines[0].GLAccountID)
		assert.Equal(t, int64(8000), lines[0].DebitAmount)
		assert.Equal(t, h.lookupGLAccount(t, "1110"), lines[1].GLAccountID)
		assert.Equal(t, int64(8000), lines[1].CreditAmount)

		var source struct {
			SourceEventType string `bun:"source_event_type"`
			Status          string `bun:"status"`
		}
		require.NoError(t, h.db.NewSelect().
			Table("journal_sources").
			Column("source_event_type", "status").
			Where("source_object_id = ?", approved.ID.String()).
			Limit(1).
			Scan(h.ctx, &source))
		assert.Equal(t, "InvoiceWriteOffCreated", source.SourceEventType)
		assert.Equal(t, "Posted", source.Status)

		var balance struct {
			PeriodDebitMinor  int64 `bun:"period_debit_minor"`
			PeriodCreditMinor int64 `bun:"period_credit_minor"`
		}
		require.NoError(t, h.db.NewSelect().
			Table("gl_account_balances_by_period").
			Column("period_debit_minor", "period_credit_minor").
			Where("gl_account_id = ?", h.lookupGLAccount(t, "6940")).
			Limit(1).
			Scan(h.ctx, &balance))
		assert.Equal(t, int64(8000), balance.PeriodDebitMinor)
		assert.Equal(t, int64(0), balance.PeriodCreditMinor)
	})

	t.Run("approve revalidates against current state", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = tenant.ApprovalPolicyAlways
		})

		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Base freight", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		pending, submitErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "pending-" + entity.ID.String(),
			Reason:         "Pending review",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusPendingApproval, pending.Status)

		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = tenant.ApprovalPolicyAmountThreshold
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})
		_, executeErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "consume-" + entity.ID.String(),
			Reason:         "Later credit",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, executeErr)

		failed, approveErr := h.service.Approve(h.ctx, &servicesports.ApproveInvoiceAdjustmentRequest{
			AdjustmentID: pending.ID,
			TenantInfo:   h.tenantInfo(),
		}, h.actor())
		require.NoError(t, approveErr)
		assert.Equal(t, invoiceadjustment.StatusExecutionFailed, failed.Status)
		assert.Contains(t, failed.ExecutionError, "remaining eligible amount")
	})

	t.Run("reject marks pending approval adjustments rejected", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = tenant.ApprovalPolicyAlways
		})

		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Base freight", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		pending, submitErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "reject-" + entity.ID.String(),
			Reason:         "Awaiting approval",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusPendingApproval, pending.Status)

		rejected, rejectErr := h.service.Reject(h.ctx, &servicesports.RejectInvoiceAdjustmentRequest{
			AdjustmentID: pending.ID,
			TenantInfo:   h.tenantInfo(),
			Reason:       "Denied by finance",
		}, h.actor())
		require.NoError(t, rejectErr)
		assert.Equal(t, invoiceadjustment.StatusRejected, rejected.Status)
		assert.Equal(t, invoiceadjustment.ApprovalStatusRejected, rejected.ApprovalStatus)
		assert.Equal(t, "Denied by finance", rejected.RejectionReason)
	})

	t.Run("idempotency and partial credits exhaust eligibility", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Base freight", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		firstReq := &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "idem-" + entity.ID.String(),
			Reason:         "First partial",
			TenantInfo:     h.tenantInfo(),
			Lines: []*servicesports.InvoiceAdjustmentLineInput{{
				OriginalLineID: entity.Lines[0].ID,
				CreditQuantity: decimal.NewFromInt(1),
				CreditAmount:   decimal.NewFromInt(40),
			}},
		}
		first, submitErr := h.service.Submit(h.ctx, firstReq, h.actor())
		require.NoError(t, submitErr)
		second, retryErr := h.service.Submit(h.ctx, firstReq, h.actor())
		require.NoError(t, retryErr)
		assert.Equal(t, first.ID, second.ID)

		_, submitSecondErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "idem-second-" + entity.ID.String(),
			Reason:         "Second partial",
			TenantInfo:     h.tenantInfo(),
			Lines: []*servicesports.InvoiceAdjustmentLineInput{{
				OriginalLineID: entity.Lines[0].ID,
				CreditQuantity: decimal.NewFromInt(1),
				CreditAmount:   decimal.NewFromInt(60),
			}},
		}, h.actor())
		require.NoError(t, submitSecondErr)

	})

	t.Run("paid invoice policy can block or require approval", func(t *testing.T) {
		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Settled freight", 1, 100),
		}, invoice.SettlementStatusPaid, decimal.NewFromInt(100))

		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.PaidInvoiceAdjustmentPolicy = tenant.AdjustmentEligibilityDisallow
		})
		blocked, blockedErr := h.service.Preview(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "paid-blocked-" + entity.ID.String(),
			Reason:         "Blocked case",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, blockedErr)
		assert.Contains(t, blocked.Errors, "invoiceId")

		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.PaidInvoiceAdjustmentPolicy = tenant.AdjustmentEligibilityAllowWithApproval
		})
		allowed, allowedErr := h.service.Preview(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "paid-approval-" + entity.ID.String(),
			Reason:         "Approval case",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, allowedErr)
		assert.True(t, allowed.RequiresApproval)
		assert.True(t, allowed.RequiresReconciliationException)
	})

	t.Run("reason and attachment policies are enforced", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.AdjustmentReasonRequirement = tenant.RequirementPolicyRequired
			control.AdjustmentAttachmentRequirement = tenant.AdjustmentAttachmentPolicyRequiredForAll
			control.StandardAdjustmentApprovalPolicy = tenant.ApprovalPolicyAmountThreshold
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})
		h.setCustomerSupportingDocumentPolicy(
			t,
			customer.InvoiceAdjustmentSupportingDocumentPolicyRequired,
		)
		t.Cleanup(func() {
			h.setCustomerSupportingDocumentPolicy(
				t,
				customer.InvoiceAdjustmentSupportingDocumentPolicyInherit,
			)
		})

		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Documented adjustment", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		_, missingErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "policy-missing-" + entity.ID.String(),
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.Error(t, missingErr)
		assert.Contains(t, missingErr.Error(), "reason")
		assert.Contains(t, missingErr.Error(), "Supporting documents")

		doc := h.createDocument(t, document.StatusActive)
		valid, validErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "policy-valid-" + entity.ID.String(),
			Reason:         "Has support",
			AttachmentIDs:  []pulid.ID{doc.ID},
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, validErr)
		assert.Equal(t, invoiceadjustment.StatusExecuted, valid.Status)

		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.AdjustmentAttachmentRequirement = tenant.AdjustmentAttachmentPolicyOptional
		})
	})

	t.Run("customer supporting document policy overrides organization default", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.AdjustmentAttachmentRequirement = tenant.AdjustmentAttachmentPolicyRequiredForAll
			control.StandardAdjustmentApprovalPolicy = tenant.ApprovalPolicyAmountThreshold
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})
		h.setCustomerSupportingDocumentPolicy(
			t,
			customer.InvoiceAdjustmentSupportingDocumentPolicyOptional,
		)
		profile, profileErr := h.customerRepo.GetByID(h.ctx, repositories.GetCustomerByIDRequest{
			ID:         h.customerID,
			TenantInfo: h.tenantInfo(),
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeBillingProfile: true,
			},
		})
		require.NoError(t, profileErr)
		require.NotNil(t, profile.BillingProfile)
		assert.Equal(
			t,
			customer.InvoiceAdjustmentSupportingDocumentPolicyOptional,
			profile.BillingProfile.InvoiceAdjustmentSupportingDocumentPolicy,
		)
		t.Cleanup(func() {
			h.setCustomerSupportingDocumentPolicy(
				t,
				customer.InvoiceAdjustmentSupportingDocumentPolicyInherit,
			)
		})

		entity := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Customer override optional", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		draft, draftErr := h.service.CreateDraft(h.ctx, &servicesports.CreateDraftInvoiceAdjustmentRequest{
			InvoiceID:  entity.ID,
			TenantInfo: h.tenantInfo(),
		}, h.actor())
		require.NoError(t, draftErr)
		assert.False(t, draft.SupportingDocumentsRequired)
		assert.Equal(
			t,
			customer.InvoiceAdjustmentSupportingDocumentPolicyOptional,
			draft.CustomerSupportingDocumentPolicy,
		)
		assert.Equal(
			t,
			string(invoiceadjustment.SupportingDocumentPolicySourceCustomerBillingProfile),
			draft.SupportingDocumentPolicySource,
		)

		optionalPreview, optionalErr := h.service.Preview(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "customer-optional-" + entity.ID.String(),
			Reason:         "No docs required",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, optionalErr)
		assert.False(t, optionalPreview.SupportingDocumentsRequired)
		assert.NotContains(t, optionalPreview.Errors, "attachmentIds")

		h.setCustomerSupportingDocumentPolicy(
			t,
			customer.InvoiceAdjustmentSupportingDocumentPolicyRequired,
		)
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.AdjustmentAttachmentRequirement = tenant.AdjustmentAttachmentPolicyOptional
		})

		requiredPreview, requiredErr := h.service.Preview(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "customer-required-" + entity.ID.String(),
			Reason:         "Docs required",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, requiredErr)
		assert.True(t, requiredPreview.SupportingDocumentsRequired)
		assert.Equal(
			t,
			customer.InvoiceAdjustmentSupportingDocumentPolicyRequired,
			requiredPreview.CustomerSupportingDocumentPolicy,
		)
		assert.Equal(
			t,
			string(invoiceadjustment.SupportingDocumentPolicySourceCustomerBillingProfile),
			requiredPreview.SupportingDocumentPolicySource,
		)

		_, requiredSubmitErr := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "customer-required-submit-" + entity.ID.String(),
			Reason:         "Docs required",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.Error(t, requiredSubmitErr)
		assert.Contains(
			t,
			requiredSubmitErr.Error(),
			"Supporting documents are required for this adjustment by policy",
		)
	})

	t.Run("bulk inline tracks partial success and large batches route to Temporal", func(t *testing.T) {
		h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		first := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Base freight", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)
		second := h.createPostedInvoice(t, []invoice.Line{
			makeInvoiceLine(1, invoice.LineTypeFreight, "Base freight", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		batch, batchErr := h.service.BulkSubmit(h.ctx, &servicesports.InvoiceAdjustmentBulkRequest{
			IdempotencyKey: "bulk-inline",
			TenantInfo:     h.tenantInfo(),
			Items: []*servicesports.InvoiceAdjustmentRequest{
				{
					InvoiceID:      first.ID,
					Kind:           invoiceadjustment.KindCreditOnly,
					IdempotencyKey: "bulk-inline-1",
					Reason:         "Valid",
				},
				{
					InvoiceID:      second.ID,
					Kind:           invoiceadjustment.KindCreditOnly,
					IdempotencyKey: "bulk-inline-2",
					Reason:         "Invalid",
					Lines: []*servicesports.InvoiceAdjustmentLineInput{{
						OriginalLineID: second.Lines[0].ID,
						CreditQuantity: decimal.NewFromInt(1),
						CreditAmount:   decimal.NewFromInt(150),
					}},
				},
			},
		}, h.actor())
		require.NoError(t, batchErr)
		assert.Equal(t, invoiceadjustment.BatchStatusPartial, batch.Status)
		assert.Equal(t, 1, batch.SucceededCount)
		assert.Equal(t, 1, batch.FailedCount)

		temporalStarter := &fakeWorkflowStarter{enabled: true}
		h.service = h.buildService(temporalStarter, decimal.NewFromInt(100))
		items := make([]*servicesports.InvoiceAdjustmentRequest, 0, batchInlineThreshold+1)
		for i := 0; i < batchInlineThreshold+1; i++ {
			items = append(items, &servicesports.InvoiceAdjustmentRequest{
				InvoiceID:      first.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: fmt.Sprintf("bulk-temporal-%d", i),
				Reason:         "Queued",
			})
		}

		queued, queuedErr := h.service.BulkSubmit(h.ctx, &servicesports.InvoiceAdjustmentBulkRequest{
			IdempotencyKey: "bulk-temporal",
			TenantInfo:     h.tenantInfo(),
			Items:          items,
		}, h.actor())
		require.NoError(t, queuedErr)
		assert.Equal(t, invoiceadjustment.BatchStatusSubmitted, queued.Status)
		require.Len(t, temporalStarter.calls, 1)
		assert.Equal(t, invoiceadjustmentjobs.InvoiceAdjustmentBatchWorkflowName, temporalStarter.calls[0].workflow)
	})
}

func newIntegrationHarness(t *testing.T, ctx context.Context, db *bun.DB, starter servicesports.WorkflowStarter, formulaAmount decimal.Decimal) *integrationHarness {
	t.Helper()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	adjustmentRepo := invoiceadjustmentrepository.New(invoiceadjustmentrepository.Params{DB: conn, Logger: logger})
	billingQueueRepo := billingqueuerepository.New(billingqueuerepository.Params{DB: conn, Logger: logger})
	adjustmentCtrlRepo := invoiceadjustmentcontrolrepository.New(invoiceadjustmentcontrolrepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	billingCtrlRepo := billingcontrolrepository.New(billingcontrolrepository.Params{DB: conn, Logger: logger})
	customerRepo := customerrepository.New(customerrepository.Params{
		DB:      conn,
		Logger:  logger,
		M2MSync: m2msync.NewSyncer(m2msync.SyncerParams{Logger: logger}),
	})
	shipmentRepo := shipmentrepository.New(shipmentrepository.Params{DB: conn, Logger: logger})
	shipmentCtrlRepo := shipmentcontrolrepository.New(shipmentcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	documentRepo := documentrepository.New(documentrepository.Params{DB: conn, Logger: logger})

	var org seededOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	var customer seededCustomer
	require.NoError(t, db.NewSelect().Table("customers").Column("id", "name", "code", "address_line_1", "city", "postal_code").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &customer))
	var shp seededShipment
	require.NoError(t, db.NewSelect().Table("shipments").Column("id", "pro_number", "bol").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &shp))

	h := &integrationHarness{
		ctx:                ctx,
		db:                 db,
		conn:               conn,
		customerRepo:       customerRepo,
		invoiceRepo:        invoiceRepo,
		adjustmentRepo:     adjustmentRepo,
		billingQueueRepo:   billingQueueRepo,
		adjustmentCtrlRepo: adjustmentCtrlRepo,
		accountingRepo:     accountingRepo,
		orgID:              org.ID,
		buID:               org.BusinessUnitID,
		userID:             user.ID,
		customerID:         customer.ID,
		customerName:       customer.Name,
		customerCode:       customer.Code,
		customerAddress1:   customer.AddressLine1,
		customerCity:       customer.City,
		customerPostal:     customer.PostalCode,
		customerCountry:    "US",
		customerState:      "CA",
		shipmentID:         shp.ID,
		shipmentPro:        shp.ProNumber,
		shipmentBOL:        shp.BOL,
		nextNumber:         0,
	}

	h.service = New(Params{
		Logger:             logger,
		DB:                 conn,
		Repo:               adjustmentRepo,
		InvoiceRepo:        invoiceRepo,
		CustomerRepo:       customerRepo,
		BillingQueueRepo:   billingQueueRepo,
		ShipmentRepo:       shipmentRepo,
		ShipmentCtrlRepo:   shipmentCtrlRepo,
		BillingCtrlRepo:    billingCtrlRepo,
		AdjustmentCtrlRepo: adjustmentCtrlRepo,
		AccountingRepo:     accountingRepo,
		JournalRepo:        journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger}),
		FiscalPeriodRepo:   fiscalPeriodRepo,
		DocumentRepo:       documentRepo,
		Validator:          NewValidator(ValidatorParams{}),
		AuditService:       noopAuditService{},
		WorkflowStarter:    starter,
		Commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         &fakeFormulaCalculator{amount: formulaAmount},
			AccessorialRepo: fakeAccessorialRepo{},
		}),
		Generator:         &fakeGenerator{},
		SequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "ACC-SEQ"},
	})

	return h
}

func (h *integrationHarness) buildService(starter servicesports.WorkflowStarter, formulaAmount decimal.Decimal) servicesports.InvoiceAdjustmentService {
	return New(Params{
		Logger:      zap.NewNop(),
		DB:          h.conn,
		Repo:        h.adjustmentRepo,
		InvoiceRepo: h.invoiceRepo,
		CustomerRepo: customerrepository.New(customerrepository.Params{
			DB:      h.conn,
			Logger:  zap.NewNop(),
			M2MSync: m2msync.NewSyncer(m2msync.SyncerParams{Logger: zap.NewNop()}),
		}),
		BillingQueueRepo:   h.billingQueueRepo,
		ShipmentRepo:       shipmentrepository.New(shipmentrepository.Params{DB: h.conn, Logger: zap.NewNop()}),
		ShipmentCtrlRepo:   shipmentcontrolrepository.New(shipmentcontrolrepository.Params{DB: h.conn, Logger: zap.NewNop()}),
		BillingCtrlRepo:    billingcontrolrepository.New(billingcontrolrepository.Params{DB: h.conn, Logger: zap.NewNop()}),
		AdjustmentCtrlRepo: h.adjustmentCtrlRepo,
		AccountingRepo:     h.accountingRepo,
		JournalRepo:        journalpostingrepository.New(journalpostingrepository.Params{DB: h.conn, Logger: zap.NewNop()}),
		FiscalPeriodRepo:   fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: h.conn, Logger: zap.NewNop()}),
		DocumentRepo:       documentrepository.New(documentrepository.Params{DB: h.conn, Logger: zap.NewNop()}),
		Validator:          NewValidator(ValidatorParams{}),
		AuditService:       noopAuditService{},
		WorkflowStarter:    starter,
		Commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         &fakeFormulaCalculator{amount: formulaAmount},
			AccessorialRepo: fakeAccessorialRepo{},
		}),
		Generator:         &fakeGenerator{},
		SequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "ACC-SEQ"},
	})
}

func (h *integrationHarness) ensureOpenFiscalPeriod(t *testing.T) {
	t.Helper()

	count, err := h.db.NewSelect().Table("fiscal_periods").Where("organization_id = ?", h.orgID).Where("business_unit_id = ?", h.buID).Count(h.ctx)
	require.NoError(t, err)
	if count > 0 {
		return
	}

	start := timeutils.NowUnix() - 86_400
	end := timeutils.NowUnix() + 86_400
	year := &fiscalyear.FiscalYear{
		ID:             pulid.MustNew("fy_"),
		OrganizationID: h.orgID,
		BusinessUnitID: h.buID,
		Status:         fiscalyear.StatusOpen,
		Year:           2026,
		Name:           "FY2026",
		StartDate:      start,
		EndDate:        end,
		IsCurrent:      true,
	}
	period := &fiscalperiod.FiscalPeriod{
		ID:                    pulid.MustNew("fp_"),
		OrganizationID:        h.orgID,
		BusinessUnitID:        h.buID,
		FiscalYearID:          year.ID,
		PeriodNumber:          1,
		PeriodType:            fiscalperiod.PeriodTypeMonth,
		Status:                fiscalperiod.StatusOpen,
		Name:                  "Current",
		StartDate:             start,
		EndDate:               end,
		AllowAdjustingEntries: true,
	}
	_, err = h.db.NewInsert().Model(year).Exec(h.ctx)
	require.NoError(t, err)
	_, err = h.db.NewInsert().Model(period).Exec(h.ctx)
	require.NoError(t, err)
}

func (h *integrationHarness) ensureAccountingDefaults(t *testing.T) {
	t.Helper()

	control, err := h.accountingRepo.GetByOrgID(h.ctx, h.orgID)
	require.NoError(t, err)
	control.DefaultARAccountID = h.lookupGLAccount(t, "1110")
	control.DefaultWriteOffAccountID = h.lookupGLAccount(t, "6940")
	control.JournalPostingMode = tenant.JournalPostingModeAutomatic
	control.ManualJournalEntryPolicy = tenant.ManualJournalEntryPolicyAdjustmentOnly
	control.RequireManualJEApproval = true
	_, err = h.accountingRepo.Update(h.ctx, control)
	require.NoError(t, err)
}

func (h *integrationHarness) lookupGLAccount(t *testing.T, code string) pulid.ID {
	t.Helper()

	var row glAccountRow
	require.NoError(t, h.db.NewSelect().
		Table("gl_accounts").
		Column("id").
		Where("organization_id = ?", h.orgID).
		Where("business_unit_id = ?", h.buID).
		Where("account_code = ?", code).
		Limit(1).
		Scan(h.ctx, &row))
	return row.ID
}

func (h *integrationHarness) setControls(t *testing.T, mutate func(*tenant.InvoiceAdjustmentControl)) {
	t.Helper()

	control, err := h.adjustmentCtrlRepo.GetByOrgID(h.ctx, h.orgID)
	require.NoError(t, err)
	mutate(control)
	_, err = h.adjustmentCtrlRepo.Update(h.ctx, control)
	require.NoError(t, err)
}

func (h *integrationHarness) setCustomerSupportingDocumentPolicy(
	t *testing.T,
	policy customer.InvoiceAdjustmentSupportingDocumentPolicy,
) {
	t.Helper()

	result, err := h.db.NewUpdate().
		Table("customer_billing_profiles").
		Set("invoice_adjustment_supporting_document_policy = ?", policy).
		Where("customer_id = ?", h.customerID).
		Where("organization_id = ?", h.orgID).
		Where("business_unit_id = ?", h.buID).
		Exec(h.ctx)
	require.NoError(t, err)
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
}

func (h *integrationHarness) createPostedInvoice(t *testing.T, lines []invoice.Line, settlementStatus invoice.SettlementStatus, applied decimal.Decimal) *invoice.Invoice {
	t.Helper()

	h.nextNumber++
	item, err := h.billingQueueRepo.Create(h.ctx, &billingqueue.BillingQueueItem{
		OrganizationID:        h.orgID,
		BusinessUnitID:        h.buID,
		ShipmentID:            h.shipmentID,
		Number:                fmt.Sprintf("INV-%03d", h.nextNumber),
		Status:                billingqueue.StatusPosted,
		BillType:              billingqueue.BillTypeInvoice,
		RerateVariancePercent: decimal.Zero,
	})
	require.NoError(t, err)

	lineCopies := make([]*invoice.Line, 0, len(lines))
	subtotal := decimal.Zero
	other := decimal.Zero
	total := decimal.Zero
	for idx := range lines {
		line := lines[idx]
		if line.Type == invoice.LineTypeFreight {
			subtotal = subtotal.Add(line.Amount)
		} else {
			other = other.Add(line.Amount)
		}
		total = total.Add(line.Amount)
		lineCopies = append(lineCopies, &line)
	}

	entity, err := h.invoiceRepo.Create(h.ctx, &invoice.Invoice{
		OrganizationID:     h.orgID,
		BusinessUnitID:     h.buID,
		BillingQueueItemID: item.ID,
		ShipmentID:         h.shipmentID,
		CustomerID:         h.customerID,
		Number:             item.Number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusPosted,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        timeutils.NowUnix(),
		DueDate:            invoice.DueDateFromPaymentTerm(timeutils.NowUnix(), invoice.PaymentTermNet30),
		PostedAt:           ptrInt64(timeutils.NowUnix()),
		ShipmentProNumber:  h.shipmentPro,
		ShipmentBOL:        h.shipmentBOL,
		BillToName:         h.customerName,
		BillToCode:         h.customerCode,
		BillToAddressLine1: h.customerAddress1,
		BillToCity:         h.customerCity,
		BillToState:        h.customerState,
		BillToPostalCode:   h.customerPostal,
		BillToCountry:      h.customerCountry,
		SubtotalAmount:     subtotal,
		OtherAmount:        other,
		TotalAmount:        total,
		AppliedAmount:      applied,
		SettlementStatus:   settlementStatus,
		DisputeStatus:      invoice.DisputeStatusNone,
		Lines:              lineCopies,
	})
	require.NoError(t, err)
	return entity
}

func (h *integrationHarness) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: h.orgID, BuID: h.buID, UserID: h.userID}
}

func (h *integrationHarness) actor() *servicesports.RequestActor {
	return &servicesports.RequestActor{
		PrincipalType:  servicesports.PrincipalTypeUser,
		PrincipalID:    h.userID,
		UserID:         h.userID,
		OrganizationID: h.orgID,
		BusinessUnitID: h.buID,
	}
}

func makeInvoiceLine(lineNumber int, lineType invoice.LineType, description string, quantity, amount int64) invoice.Line {
	qty := decimal.NewFromInt(quantity)
	total := decimal.NewFromInt(amount)
	return invoice.Line{
		LineNumber:  lineNumber,
		Type:        lineType,
		Description: description,
		Quantity:    qty,
		UnitPrice:   total.Div(qty),
		Amount:      total,
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}

func (h *integrationHarness) createDocument(t *testing.T, status document.Status) *document.Document {
	t.Helper()

	entity := &document.Document{
		OrganizationID: h.orgID,
		BusinessUnitID: h.buID,
		FileName:       "adjustment-support.pdf",
		OriginalName:   "adjustment-support.pdf",
		FileSize:       128,
		FileType:       "application/pdf",
		StoragePath:    "test/adjustment-support.pdf",
		Status:         status,
		ResourceID:     h.shipmentID.String(),
		ResourceType:   "Shipment",
		UploadedByID:   h.userID,
	}
	_, err := h.db.NewInsert().Model(entity).Exec(h.ctx)
	require.NoError(t, err)
	return entity
}

func TestInvoiceAdjustmentSchemaAndSeeds(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{
		System: config.SystemConfig{
			SystemUserPassword: "test-system-password",
		},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	type tableCheck struct {
		Name string `bun:"to_regclass"`
	}
	for _, tableName := range []string{
		"invoice_adjustments",
		"invoice_adjustment_lines",
		"invoice_adjustment_batches",
		"invoice_adjustment_batch_items",
		"invoice_correction_groups",
		"invoice_reconciliation_exceptions",
	} {
		var row tableCheck
		require.NoError(t, db.NewSelect().ColumnExpr("to_regclass(?)", tableName).Scan(ctx, &row.Name))
		assert.Equal(t, tableName, row.Name)
	}

	type columnCheck struct {
		IsNullable    string `bun:"is_nullable"`
		ColumnDefault string `bun:"column_default"`
	}
	var rerateColumn columnCheck
	require.NoError(t, db.NewSelect().
		Table("information_schema.columns").
		Column("is_nullable", "column_default").
		Where("table_name = ?", "billing_queue_items").
		Where("column_name = ?", "rerate_variance_percent").
		Scan(ctx, &rerateColumn))
	assert.Equal(t, "NO", rerateColumn.IsNullable)
	assert.Contains(t, rerateColumn.ColumnDefault, "0")

	type enumLabel struct {
		Label string `bun:"enumlabel"`
	}
	enumRows := make([]enumLabel, 0, 3)
	require.NoError(t, db.NewSelect().
		TableExpr("pg_enum e").
		Column("e.enumlabel").
		Join("JOIN pg_type t ON t.oid = e.enumtypid").
		Where("t.typname = ?", "write_off_approval_policy_enum").
		OrderExpr("e.enumsortorder ASC").
		Scan(ctx, &enumRows))
	assert.Equal(t, []string{
		"Disallow",
		"AlwaysRequireApproval",
		"RequireApprovalAboveThreshold",
	}, []string{enumRows[0].Label, enumRows[1].Label, enumRows[2].Label})

	type seedCheck struct {
		AdjustmentControls int `bun:"adjustment_controls"`
		AccountingControls int `bun:"accounting_controls"`
		WriteOffConfigured int `bun:"write_off_configured"`
		ARConfigured       int `bun:"ar_configured"`
	}
	var counts seedCheck
	require.NoError(t, db.NewSelect().
		TableExpr("invoice_adjustment_controls iac").
		ColumnExpr("(SELECT COUNT(*) FROM invoice_adjustment_controls) AS adjustment_controls").
		ColumnExpr("(SELECT COUNT(*) FROM accounting_controls) AS accounting_controls").
		ColumnExpr("(SELECT COUNT(*) FROM accounting_controls WHERE default_write_off_account_id IS NOT NULL) AS write_off_configured").
		ColumnExpr("(SELECT COUNT(*) FROM accounting_controls WHERE default_ar_account_id IS NOT NULL) AS ar_configured").
		Limit(1).
		Scan(ctx, &counts))
	assert.GreaterOrEqual(t, counts.AdjustmentControls, 1)
	assert.GreaterOrEqual(t, counts.AccountingControls, 1)
	assert.Equal(t, counts.AccountingControls, counts.WriteOffConfigured)
	assert.Equal(t, counts.AccountingControls, counts.ARConfigured)
}
