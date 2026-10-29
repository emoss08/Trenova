//go:build integration

package invoiceadjustmentservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueuerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/chargeallocationrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerledgerrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoiceadjustmentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentengine"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/invoiceadjustmentjobs/invoiceadjustmentcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/invoiceadjustmentcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/shipmentcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentadditionalchargerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommodityrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/customerrepository"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type integrationHarness struct {
	ctx                context.Context
	db                 *keel.DB
	conn               *postgres.Connection
	service            invoiceadjustmentports.InvoiceAdjustmentService
	customerRepo       customerports.CustomerRepository
	invoiceRepo        invoiceports.InvoiceRepository
	adjustmentRepo     invoiceadjustmentports.InvoiceAdjustmentRepository
	billingQueueRepo   billingqueueports.BillingQueueRepository
	adjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
	accountingRepo     tenantports.AccountingControlRepository
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
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

type seededUser struct {
	ID pulid.ID
}

type seededCustomer struct {
	ID           pulid.ID
	Name         string
	Code         string
	AddressLine1 string
	City         string
	PostalCode   string
}

type seededShipment struct {
	ID        pulid.ID
	ProNumber string
	BOL       string
}

type journalEntryRow struct {
	ID          pulid.ID
	Status      string
	TotalDebit  int64
	TotalCredit int64
}

type journalEntryLineRow struct {
	GLAccountID  pulid.ID
	DebitAmount  int64
	CreditAmount int64
}

type fakeGenerator struct {
	next int
}

func (g *fakeGenerator) GenerateInvoiceNumber(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
	string,
) (string, error) {
	g.next++
	return fmt.Sprintf("REBILL-%03d", g.next), nil
}

func (g *fakeGenerator) GenerateCreditMemoNumber(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
	string,
) (string, error) {
	g.next++
	return fmt.Sprintf("CM-%03d", g.next), nil
}

type fakeAccessorialRepo struct{}

func (fakeAccessorialRepo) List(
	context.Context,
	*accessorialchargeports.ListAccessorialChargeRequest,
) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	return nil, nil
}

func (fakeAccessorialRepo) ListConnection(
	context.Context,
	*accessorialchargeports.ListAccessorialChargeConnectionRequest,
) (*pagination.CursorListResult[*accessorialcharge.AccessorialCharge], error) {
	return nil, nil
}

func (fakeAccessorialRepo) SelectOptions(
	context.Context,
	*pagination.SelectQueryRequest,
) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	return nil, nil
}

func (fakeAccessorialRepo) GetByID(
	context.Context,
	accessorialchargeports.GetAccessorialChargeByIDRequest,
) (*accessorialcharge.AccessorialCharge, error) {
	return nil, nil
}

func (fakeAccessorialRepo) ListCatalog(
	context.Context,
	pagination.TenantInfo,
) ([]*accessorialcharge.AccessorialCharge, error) {
	return nil, nil
}

func (fakeAccessorialRepo) Create(
	context.Context,
	*accessorialcharge.AccessorialCharge,
) (*accessorialcharge.AccessorialCharge, error) {
	return nil, nil
}

func (fakeAccessorialRepo) Update(
	context.Context,
	*accessorialcharge.AccessorialCharge,
) (*accessorialcharge.AccessorialCharge, error) {
	return nil, nil
}

type noopAuditService struct {
	*auditportsmock.MockAuditService
}

func (noopAuditService) List(
	context.Context,
	*auditports.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (noopAuditService) ListByResourceID(
	context.Context,
	*auditports.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (noopAuditService) GetByID(
	context.Context,
	auditports.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	return nil, nil
}

func (noopAuditService) LogAction(
	*auditports.LogActionParams,
	...auditports.LogOption,
) error {
	return nil
}
func (noopAuditService) LogActions([]auditports.BulkLogEntry) error { return nil }

func (noopAuditService) RegisterSensitiveFields(
	permission.Resource,
	[]auditports.SensitiveField,
) error {
	return nil
}

type noopRealtimeService struct{}

func (noopRealtimeService) PublishResourceInvalidation(
	context.Context,
	*realtimeports.PublishResourceInvalidationRequest,
) error {
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

// GetFirstExecutionRunID identifies the run a continue-as-new chain started
// from. Nothing here continues as new, so the first run is this one.
func (f fakeWorkflowRun) GetFirstExecutionRunID() string { return f.runID }

type fakeWorkflowStarter struct {
	enabled bool
	calls   []workflowCall
}

func (f *fakeWorkflowStarter) CancelWorkflow(context.Context, string, string) error {
	return nil
}

func (f *fakeWorkflowStarter) SignalWorkflow(context.Context, string, string, string, any) error {
	return nil
}

type workflowCall struct {
	options  client.StartWorkflowOptions
	workflow any
	args     []any
}

func (f *fakeWorkflowStarter) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	f.calls = append(f.calls, workflowCall{options: options, workflow: workflow, args: args})
	return fakeWorkflowRun{id: options.ID, runID: "run-1"}, nil
}

func (f *fakeWorkflowStarter) Enabled() bool { return f.enabled }

func TestInvoiceAdjustmentService_EngineScenarios(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	h := newIntegrationHarness(
		t,
		ctx,
		db,
		&fakeWorkflowStarter{enabled: false},
		decimal.NewFromInt(100),
	)
	h.ensureOpenFiscalPeriod(t)
	h.ensureAccountingDefaults(t)
	h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
		control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
	})

	t.Run("full invoice credit only executes credit memo", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = invoiceadjustmentcontrol.ApprovalPolicyAmountThreshold
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
			makeInvoiceLine(2, invoice.InvoiceLineTypeAccessorial, "Fuel", 1, 25),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		adjustment, submitErr := h.service.Submit(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "credit-only-" + entity.ID.String(),
				Reason:         "Customer concession",
				TenantInfo:     h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusExecuted, adjustment.Status)
		require.True(t, adjustment.ReplacementInvoiceID.IsNil())
		require.True(t, adjustment.RebillQueueItemID.IsNil())
		require.Contains(t, adjustment.Metadata, "creditMemoJournalEntryId")

		journal := h.creditMemoJournal(t, adjustment.CreditMemoInvoiceID)
		require.NotNil(t, journal)
		assert.Equal(t, "CreditMemoPosted", journal.SourceEventType)
		assert.Equal(t, "Posted", journal.Status)
		require.Len(t, journal.Lines, 2)
		assert.Equal(t, h.lookupGLAccount(t, "4000"), journal.Lines[0].GLAccountID)
		assert.Equal(t, int64(12500), journal.Lines[0].DebitAmount)
		assert.Equal(t, h.lookupGLAccount(t, "1110"), journal.Lines[1].GLAccountID)
		assert.Equal(t, int64(12500), journal.Lines[1].CreditAmount)

		ledger := h.customerLedgerLines(t, adjustment.CreditMemoInvoiceID)
		require.Len(t, ledger, 1)
		assert.Equal(t, int64(-12500), ledger[0].AmountMinor)
		assert.Equal(t, "CreditMemoPosted", ledger[0].SourceEventType)
		assert.Equal(t, entity.ID, ledger[0].RelatedInvoiceID)
	})

	t.Run("credit and rebill creates draft replacement invoice and lineage", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
			control.ReplacementInvoiceReviewPolicy = invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
			makeInvoiceLine(2, invoice.InvoiceLineTypeAccessorial, "Fuel", 1, 50),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		adjustment, submitErr := h.service.Submit(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditRebill,
				RebillStrategy: invoiceadjustment.RebillStrategyManual,
				IdempotencyKey: "credit-rebill-" + entity.ID.String(),
				Reason:         "Rate correction",
				TenantInfo:     h.tenantInfo(),
				Lines: []*invoiceadjustmentports.InvoiceAdjustmentLineInput{
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
			},
			h.actor(),
		)
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusExecuted, adjustment.Status)
		require.True(t, adjustment.CreditMemoInvoiceID.IsNotNil())
		require.True(t, adjustment.ReplacementInvoiceID.IsNotNil())
		require.True(t, adjustment.RebillQueueItemID.IsNotNil())
		require.Equal(
			t,
			invoiceadjustment.ReplacementReviewStatusRequired,
			adjustment.ReplacementReviewStatus,
		)

		replacement, getErr := h.invoiceRepo.GetByID(h.ctx, invoiceports.GetInvoiceByIDRequest{
			ID:         adjustment.ReplacementInvoiceID,
			TenantInfo: h.tenantInfo(),
		})
		require.NoError(t, getErr)
		assert.Equal(t, invoice.StatusDraft, replacement.Status)
		assert.Equal(t, entity.ID, replacement.SupersedesInvoiceID)

		queueItem, queueErr := h.billingQueueRepo.GetByID(
			h.ctx,
			&billingqueueports.GetBillingQueueItemByIDRequest{
				ItemID:     adjustment.RebillQueueItemID,
				TenantInfo: h.tenantInfo(),
			},
		)
		require.NoError(t, queueErr)
		assert.Equal(t, billingqueue.StatusReadyForReview, queueItem.Status)
		require.NotNil(t, queueItem.SourceCreditMemoInvoiceID)
		assert.Equal(t, adjustment.CreditMemoInvoiceID, *queueItem.SourceCreditMemoInvoiceID)
		require.NotNil(t, queueItem.SourceInvoiceAdjustmentID)
		assert.Equal(t, adjustment.ID, *queueItem.SourceInvoiceAdjustmentID)
		assert.True(t, queueItem.RequiresReplacementReview)

		billerID := h.userID
		queueItem.Status = billingqueue.StatusInReview
		queueItem.AssignedBillerID = &billerID
		_, updateErr := h.billingQueueRepo.Update(h.ctx, queueItem)
		require.NoError(t, updateErr)

		_, statusErr := h.buildBillingQueueService().UpdateStatus(
			h.ctx,
			&billingqueueports.UpdateBillingQueueStatusRequest{
				ItemID:     queueItem.ID,
				NewStatus:  billingqueue.StatusApproved,
				TenantInfo: h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, statusErr)

		completed, completedErr := h.adjustmentRepo.GetByID(
			h.ctx,
			invoiceadjustmentports.GetInvoiceAdjustmentRequest{
				ID:         adjustment.ID,
				TenantInfo: h.tenantInfo(),
			},
		)
		require.NoError(t, completedErr)
		assert.Equal(
			t,
			invoiceadjustment.ReplacementReviewStatusCompleted,
			completed.ReplacementReviewStatus,
		)
	})

	t.Run("full reversal ignores partial line input and does not rebill", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
			control.ReplacementInvoiceReviewPolicy = invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicyNoAdditionalReview
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
			makeInvoiceLine(2, invoice.InvoiceLineTypeAccessorial, "Fuel", 1, 50),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		preview, previewErr := h.service.Preview(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindFullReversal,
				IdempotencyKey: "full-reversal-preview-" + entity.ID.String(),
				Reason:         "Reverse the invoice",
				TenantInfo:     h.tenantInfo(),
				Lines: []*invoiceadjustmentports.InvoiceAdjustmentLineInput{{
					OriginalLineID: entity.Lines[0].ID,
					CreditQuantity: decimal.NewFromInt(1),
					CreditAmount:   decimal.NewFromInt(25),
					RebillQuantity: decimal.NewFromInt(1),
					RebillAmount:   decimal.NewFromInt(25),
				}},
			},
			h.actor(),
		)
		require.NoError(t, previewErr)
		require.Empty(t, preview.Errors)
		require.Len(t, preview.Lines, 2)
		assert.True(t, decimal.NewFromInt(150).Equal(preview.CreditTotalAmount))
		assert.True(t, preview.RebillTotalAmount.IsZero())

		adjustment, submitErr := h.service.Submit(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindFullReversal,
				IdempotencyKey: "full-reversal-submit-" + entity.ID.String(),
				Reason:         "Reverse the invoice",
				TenantInfo:     h.tenantInfo(),
				Lines: []*invoiceadjustmentports.InvoiceAdjustmentLineInput{{
					OriginalLineID: entity.Lines[0].ID,
					CreditQuantity: decimal.NewFromInt(1),
					CreditAmount:   decimal.NewFromInt(25),
					RebillQuantity: decimal.NewFromInt(1),
					RebillAmount:   decimal.NewFromInt(25),
				}},
			},
			h.actor(),
		)
		require.NoError(t, submitErr)
		assert.True(t, decimal.NewFromInt(150).Equal(adjustment.CreditTotalAmount))
		assert.True(t, adjustment.RebillTotalAmount.IsZero())
		assert.True(t, adjustment.ReplacementInvoiceID.IsNil())
		assert.True(t, adjustment.RebillQueueItemID.IsNil())
	})

	t.Run("credit only partial line behavior is unchanged", func(t *testing.T) {
		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
			makeInvoiceLine(2, invoice.InvoiceLineTypeAccessorial, "Fuel", 1, 50),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		preview, previewErr := h.service.Preview(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "credit-only-partial-preview-" + entity.ID.String(),
				Reason:         "Partial concession",
				TenantInfo:     h.tenantInfo(),
				Lines: []*invoiceadjustmentports.InvoiceAdjustmentLineInput{{
					OriginalLineID: entity.Lines[0].ID,
					CreditQuantity: decimal.NewFromInt(1),
					CreditAmount:   decimal.NewFromInt(40),
				}},
			},
			h.actor(),
		)
		require.NoError(t, previewErr)
		require.Empty(t, preview.Errors)
		require.Len(t, preview.Lines, 1)
		assert.True(t, decimal.NewFromInt(40).Equal(preview.CreditTotalAmount))
	})

	t.Run("write-off uses write-off approval policy and creates journal entry", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.WriteOffApprovalPolicy = invoiceadjustmentcontrol.WriteOffApprovalPolicyRequireApprovalAboveThreshold
			control.WriteOffApprovalThreshold = decimal.NewFromInt(50)
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Open balance", 1, 80),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		pending, submitErr := h.service.Submit(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindWriteOff,
				IdempotencyKey: "writeoff-" + entity.ID.String(),
				Reason:         "Bad debt",
				TenantInfo:     h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusPendingApproval, pending.Status)

		approved, approveErr := h.service.Approve(
			h.ctx,
			&invoiceadjustmentports.ApproveInvoiceAdjustmentRequest{
				AdjustmentID: pending.ID,
				TenantInfo:   h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, approveErr)
		require.Equal(t, invoiceadjustment.StatusExecuted, approved.Status)
		assert.Contains(t, approved.Metadata, "writeOffJournalEntryId")

		var entry journalEntryRow
		require.NoError(t, h.db.Pool().QueryRow(h.ctx,
			`SELECT id, status, total_debit, total_credit FROM journal_entries
			WHERE organization_id = $1 AND business_unit_id = $2 AND reference_id = $3 LIMIT 1`,
			h.orgID, h.buID, approved.ID.String(),
		).Scan(&entry.ID, &entry.Status, &entry.TotalDebit, &entry.TotalCredit))
		assert.Equal(t, "Posted", entry.Status)
		assert.Equal(t, int64(8000), entry.TotalDebit)
		assert.Equal(t, int64(8000), entry.TotalCredit)

		lines := h.journalEntryLines(t, entry.ID)
		require.Len(t, lines, 2)
		assert.Equal(t, h.lookupGLAccount(t, "6940"), lines[0].GLAccountID)
		assert.Equal(t, int64(8000), lines[0].DebitAmount)
		assert.Equal(t, h.lookupGLAccount(t, "1110"), lines[1].GLAccountID)
		assert.Equal(t, int64(8000), lines[1].CreditAmount)

		var source struct {
			SourceEventType string
			Status          string
		}
		require.NoError(t, h.db.Pool().QueryRow(h.ctx,
			`SELECT source_event_type, status FROM journal_sources
			WHERE organization_id = $1 AND business_unit_id = $2 AND source_object_id = $3 LIMIT 1`,
			h.orgID, h.buID, approved.ID.String(),
		).Scan(&source.SourceEventType, &source.Status))
		assert.Equal(t, "InvoiceWriteOffCreated", source.SourceEventType)
		assert.Equal(t, "Posted", source.Status)

		var balance struct {
			PeriodDebitMinor  int64
			PeriodCreditMinor int64
		}
		require.NoError(t, h.db.Pool().QueryRow(h.ctx,
			`SELECT period_debit_minor, period_credit_minor FROM gl_account_balances_by_period
			WHERE organization_id = $1 AND business_unit_id = $2 AND gl_account_id = $3 LIMIT 1`,
			h.orgID, h.buID, h.lookupGLAccount(t, "6940"),
		).Scan(&balance.PeriodDebitMinor, &balance.PeriodCreditMinor))
		assert.Equal(t, int64(8000), balance.PeriodDebitMinor)
		assert.Equal(t, int64(0), balance.PeriodCreditMinor)

		assert.Nil(t, h.creditMemoJournal(t, approved.CreditMemoInvoiceID))
		ledger := h.customerLedgerLines(t, approved.CreditMemoInvoiceID)
		require.Len(t, ledger, 1)
		assert.Equal(t, int64(-8000), ledger[0].AmountMinor)
		assert.Equal(t, entity.ID, ledger[0].RelatedInvoiceID)
	})

	t.Run("approve revalidates against current state", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = invoiceadjustmentcontrol.ApprovalPolicyAlways
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		pending, submitErr := h.service.Submit(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "pending-" + entity.ID.String(),
				Reason:         "Pending review",
				TenantInfo:     h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusPendingApproval, pending.Status)

		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = invoiceadjustmentcontrol.ApprovalPolicyAmountThreshold
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})
		_, executeErr := h.service.Submit(h.ctx, &invoiceadjustmentports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "consume-" + entity.ID.String(),
			Reason:         "Later credit",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, executeErr)

		failed, approveErr := h.service.Approve(
			h.ctx,
			&invoiceadjustmentports.ApproveInvoiceAdjustmentRequest{
				AdjustmentID: pending.ID,
				TenantInfo:   h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, approveErr)
		assert.Equal(t, invoiceadjustment.StatusExecutionFailed, failed.Status)
		assert.Contains(t, failed.ExecutionError, "remaining eligible amount")
	})

	t.Run("reject marks pending approval adjustments rejected", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = invoiceadjustmentcontrol.ApprovalPolicyAlways
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		pending, submitErr := h.service.Submit(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "reject-" + entity.ID.String(),
				Reason:         "Awaiting approval",
				TenantInfo:     h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, submitErr)
		require.Equal(t, invoiceadjustment.StatusPendingApproval, pending.Status)

		rejected, rejectErr := h.service.Reject(
			h.ctx,
			&invoiceadjustmentports.RejectInvoiceAdjustmentRequest{
				AdjustmentID: pending.ID,
				TenantInfo:   h.tenantInfo(),
				Reason:       "Denied by finance",
			},
			h.actor(),
		)
		require.NoError(t, rejectErr)
		assert.Equal(t, invoiceadjustment.StatusRejected, rejected.Status)
		assert.Equal(t, invoiceadjustment.ApprovalStatusRejected, rejected.ApprovalStatus)
		assert.Equal(t, "Denied by finance", rejected.RejectionReason)
	})

	t.Run("idempotency and partial credits exhaust eligibility", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		firstReq := &invoiceadjustmentports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "idem-" + entity.ID.String(),
			Reason:         "First partial",
			TenantInfo:     h.tenantInfo(),
			Lines: []*invoiceadjustmentports.InvoiceAdjustmentLineInput{{
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

		_, submitSecondErr := h.service.Submit(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "idem-second-" + entity.ID.String(),
				Reason:         "Second partial",
				TenantInfo:     h.tenantInfo(),
				Lines: []*invoiceadjustmentports.InvoiceAdjustmentLineInput{{
					OriginalLineID: entity.Lines[0].ID,
					CreditQuantity: decimal.NewFromInt(1),
					CreditAmount:   decimal.NewFromInt(60),
				}},
			},
			h.actor(),
		)
		require.NoError(t, submitSecondErr)
	})

	t.Run("paid invoice policy can block or require approval", func(t *testing.T) {
		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Settled freight", 1, 100),
		}, invoice.SettlementStatusPaid, decimal.NewFromInt(100))

		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.PaidInvoiceAdjustmentPolicy = invoiceadjustmentcontrol.AdjustmentEligibilityDisallow
		})
		blocked, blockedErr := h.service.Preview(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "paid-blocked-" + entity.ID.String(),
				Reason:         "Blocked case",
				TenantInfo:     h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, blockedErr)
		assert.Contains(t, blocked.Errors, "invoiceId")

		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.PaidInvoiceAdjustmentPolicy = invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithApproval
		})
		allowed, allowedErr := h.service.Preview(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "paid-approval-" + entity.ID.String(),
				Reason:         "Approval case",
				TenantInfo:     h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, allowedErr)
		assert.True(t, allowed.RequiresApproval)
		assert.True(t, allowed.RequiresReconciliationException)
	})

	t.Run("reason policy is enforced and supporting documents are optional", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.AdjustmentReasonRequirement = invoiceadjustmentcontrol.RequirementPolicyRequired
			control.StandardAdjustmentApprovalPolicy = invoiceadjustmentcontrol.ApprovalPolicyAmountThreshold
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Undocumented adjustment", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		_, missingErr := h.service.Submit(h.ctx, &invoiceadjustmentports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "policy-missing-" + entity.ID.String(),
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.Error(t, missingErr)
		assert.Contains(t, missingErr.Error(), "reason")
		assert.NotContains(t, missingErr.Error(), "Supporting documents")

		valid, validErr := h.service.Submit(h.ctx, &invoiceadjustmentports.InvoiceAdjustmentRequest{
			InvoiceID:      entity.ID,
			Kind:           invoiceadjustment.KindCreditOnly,
			IdempotencyKey: "policy-valid-" + entity.ID.String(),
			Reason:         "No documents needed",
			TenantInfo:     h.tenantInfo(),
		}, h.actor())
		require.NoError(t, validErr)
		assert.Equal(t, invoiceadjustment.StatusExecuted, valid.Status)
	})

	t.Run("no adjustment kind requires supporting documents", func(t *testing.T) {
		h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
			control.StandardAdjustmentApprovalPolicy = invoiceadjustmentcontrol.ApprovalPolicyAmountThreshold
			control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		})

		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Any kind", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		kinds := []invoiceadjustment.Kind{
			invoiceadjustment.KindCreditOnly,
			invoiceadjustment.KindCreditRebill,
			invoiceadjustment.KindFullReversal,
			invoiceadjustment.KindWriteOff,
		}
		for _, kind := range kinds {
			t.Run(string(kind), func(t *testing.T) {
				preview, previewErr := h.service.Preview(
					h.ctx,
					&invoiceadjustmentports.InvoiceAdjustmentRequest{
						InvoiceID:      entity.ID,
						Kind:           kind,
						RebillStrategy: invoiceadjustment.RebillStrategyManual,
						IdempotencyKey: "no-docs-" + string(kind),
						Reason:         "No documents attached",
						TenantInfo:     h.tenantInfo(),
						Lines: []*invoiceadjustmentports.InvoiceAdjustmentLineInput{{
							OriginalLineID: entity.Lines[0].ID,
							CreditQuantity: decimal.NewFromInt(1),
							CreditAmount:   decimal.NewFromInt(100),
							RebillQuantity: decimal.NewFromInt(1),
							RebillAmount:   decimal.NewFromInt(100),
						}},
					},
					h.actor(),
				)
				require.NoError(t, previewErr)
				assert.NotContains(t, preview.Errors, "attachmentIds")
			})
		}
	})

	t.Run("attached supporting documents must be usable", func(t *testing.T) {
		entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
			makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Archived evidence", 1, 100),
		}, invoice.SettlementStatusUnpaid, decimal.Zero)

		archived := h.createDocument(t, document.StatusArchived)
		preview, previewErr := h.service.Preview(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "archived-doc-" + entity.ID.String(),
				Reason:         "Archived attachment",
				AttachmentIDs:  []pulid.ID{archived.ID},
				TenantInfo:     h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, previewErr)
		assert.Contains(t, preview.Errors, "attachmentIds")

		active := h.createDocument(t, document.StatusActive)
		executed, submitErr := h.service.Submit(
			h.ctx,
			&invoiceadjustmentports.InvoiceAdjustmentRequest{
				InvoiceID:      entity.ID,
				Kind:           invoiceadjustment.KindCreditOnly,
				IdempotencyKey: "active-doc-" + entity.ID.String(),
				Reason:         "Active attachment",
				AttachmentIDs:  []pulid.ID{active.ID},
				TenantInfo:     h.tenantInfo(),
			},
			h.actor(),
		)
		require.NoError(t, submitErr)
		assert.Equal(t, invoiceadjustment.StatusExecuted, executed.Status)
	})

	t.Run(
		"bulk inline tracks partial success and large batches route to Temporal",
		func(t *testing.T) {
			h.setControls(t, func(control *invoiceadjustmentcontrol.InvoiceAdjustmentControl) {
				control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
			})

			first := h.createPostedInvoice(t, []invoice.InvoiceLine{
				makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
			}, invoice.SettlementStatusUnpaid, decimal.Zero)
			second := h.createPostedInvoice(t, []invoice.InvoiceLine{
				makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
			}, invoice.SettlementStatusUnpaid, decimal.Zero)

			batch, batchErr := h.service.BulkSubmit(
				h.ctx,
				&invoiceadjustmentports.InvoiceAdjustmentBulkRequest{
					IdempotencyKey: "bulk-inline",
					TenantInfo:     h.tenantInfo(),
					Items: []*invoiceadjustmentports.InvoiceAdjustmentRequest{
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
							Lines: []*invoiceadjustmentports.InvoiceAdjustmentLineInput{{
								OriginalLineID: second.Lines[0].ID,
								CreditQuantity: decimal.NewFromInt(1),
								CreditAmount:   decimal.NewFromInt(150),
							}},
						},
					},
				},
				h.actor(),
			)
			require.NoError(t, batchErr)
			assert.Equal(t, invoiceadjustment.BatchStatusPartial, batch.Status)
			assert.Equal(t, 1, batch.SucceededCount)
			assert.Equal(t, 1, batch.FailedCount)

			temporalStarter := &fakeWorkflowStarter{enabled: true}
			h.service = h.buildService(t, temporalStarter, decimal.NewFromInt(100))
			items := make(
				[]*invoiceadjustmentports.InvoiceAdjustmentRequest,
				0,
				batchInlineThreshold+1,
			)
			for i := range batchInlineThreshold + 1 {
				items = append(items, &invoiceadjustmentports.InvoiceAdjustmentRequest{
					InvoiceID:      first.ID,
					Kind:           invoiceadjustment.KindCreditOnly,
					IdempotencyKey: fmt.Sprintf("bulk-temporal-%d", i),
					Reason:         "Queued",
				})
			}

			queued, queuedErr := h.service.BulkSubmit(
				h.ctx,
				&invoiceadjustmentports.InvoiceAdjustmentBulkRequest{
					IdempotencyKey: "bulk-temporal",
					TenantInfo:     h.tenantInfo(),
					Items:          items,
				},
				h.actor(),
			)
			require.NoError(t, queuedErr)
			assert.Equal(t, invoiceadjustment.BatchStatusSubmitted, queued.Status)
			require.Len(t, temporalStarter.calls, 1)
			assert.Equal(
				t,
				invoiceadjustmentcontract.InvoiceAdjustmentBatchWorkflowName,
				temporalStarter.calls[0].workflow,
			)
		},
	)
}

func newIntegrationHarness(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	starter kernelports.WorkflowStarter,
	formulaAmount decimal.Decimal,
) *integrationHarness {
	t.Helper()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	invoiceRepo := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	adjustmentRepo := invoiceadjustmentrepository.New(
		invoiceadjustmentrepository.Params{DB: conn, Logger: logger},
	)
	billingQueueRepo := billingqueuerepository.New(
		billingqueuerepository.Params{DB: conn, Logger: logger},
	)
	adjustmentCtrlRepo := invoiceadjustmentcontrolrepository.New(
		invoiceadjustmentcontrolrepository.Params{DB: conn, Logger: logger},
	)
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	customerRepo := customerrepository.New(customerrepository.Params{
		DB:     conn,
		Logger: logger,
	})
	shipmentRepo := shipmentrepository.New(shipmentrepository.Params{
		DB:     conn,
		Logger: logger,
		MoveRepository: shipmentmoverepository.New(
			shipmentmoverepository.Params{DB: conn, Logger: logger},
		),
		AdditionalChargeRepository: shipmentadditionalchargerepository.New(
			shipmentadditionalchargerepository.Params{DB: conn, Logger: logger},
		),
		ChargeAllocationRepository: chargeallocationrepository.New(
			chargeallocationrepository.Params{DB: conn, Logger: logger},
		),
		CommodityRepository: shipmentcommodityrepository.New(
			shipmentcommodityrepository.Params{DB: conn, Logger: logger},
		),
	})
	shipmentCtrlRepo := shipmentcontrolrepository.New(
		shipmentcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	documentRepo := documentrepository.New(documentrepository.Params{DB: conn, Logger: logger})

	var org seededOrg
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&org.ID, &org.BusinessUnitID))
	var user seededUser
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT id FROM users WHERE current_organization_id = $1 AND business_unit_id = $2
		ORDER BY id LIMIT 1`,
		org.ID, org.BusinessUnitID,
	).Scan(&user.ID))
	var customer seededCustomer
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT id, name, code, address_line_1, city, postal_code FROM customers
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1`,
		org.ID, org.BusinessUnitID,
	).Scan(
		&customer.ID,
		&customer.Name,
		&customer.Code,
		&customer.AddressLine1,
		&customer.City,
		&customer.PostalCode,
	))
	var shp seededShipment
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT id, COALESCE(pro_number, ''), COALESCE(bol, '') FROM shipments
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1`,
		org.ID, org.BusinessUnitID,
	).Scan(&shp.ID, &shp.ProNumber, &shp.BOL))

	ctx = dbscope.WithTenant(ctx, pagination.TenantInfo{
		OrgID:  org.ID,
		BuID:   org.BusinessUnitID,
		UserID: user.ID,
	}.DBTenant())

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
		DB:                 conn,
		Repo:               adjustmentRepo,
		InvoiceRepo:        invoiceRepo,
		AdjustmentCtrlRepo: adjustmentCtrlRepo,
		Validator: invoiceadjustmentengine.NewValidator(
			invoiceadjustmentengine.ValidatorParams{},
		),
		WorkflowStarter: starter,
		Engine: invoiceadjustmentengine.New(invoiceadjustmentengine.Params{
			Logger:           logger,
			Repo:             adjustmentRepo,
			InvoiceRepo:      invoiceRepo,
			CustomerRepo:     customerRepo,
			BillingQueueRepo: billingQueueRepo,
			ShipmentRepo:     shipmentRepo,
			ShipmentCtrlRepo: shipmentCtrlRepo,
			AccountingRepo:   accountingRepo,
			JournalRepo: journalpostingrepository.New(
				journalpostingrepository.Params{DB: conn, Logger: logger},
			),
			FiscalPeriodRepo: fiscalPeriodRepo,
			DocumentRepo:     documentRepo,
			CustomerLedgerRepo: customerledgerrepository.New(
				customerledgerrepository.Params{DB: conn, Logger: logger},
			),
			Validator: invoiceadjustmentengine.NewValidator(
				invoiceadjustmentengine.ValidatorParams{},
			),
			AuditService: noopAuditService{},
			Commercial: shipmentcommercial.New(shipmentcommercial.Params{
				Logger:          zap.NewNop(),
				RateEngine:      ratingtest.NewFlatEngine(t, formulaAmount),
				AccessorialRepo: fakeAccessorialRepo{},
			}),
			Generator: &fakeGenerator{},
			SequenceGenerator: testutil.UniqueJournalSequenceGenerator{
				TestSequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "ACC-SEQ"},
			},
		}),
	})

	return h
}

func (h *integrationHarness) buildService(
	t *testing.T,
	starter kernelports.WorkflowStarter,
	formulaAmount decimal.Decimal,
) invoiceadjustmentports.InvoiceAdjustmentService {
	t.Helper()

	return New(Params{
		DB:                 h.conn,
		Repo:               h.adjustmentRepo,
		InvoiceRepo:        h.invoiceRepo,
		AdjustmentCtrlRepo: h.adjustmentCtrlRepo,
		Validator: invoiceadjustmentengine.NewValidator(
			invoiceadjustmentengine.ValidatorParams{},
		),
		WorkflowStarter: starter,
		Engine: invoiceadjustmentengine.New(invoiceadjustmentengine.Params{
			Logger:      zap.NewNop(),
			Repo:        h.adjustmentRepo,
			InvoiceRepo: h.invoiceRepo,
			CustomerRepo: customerrepository.New(customerrepository.Params{
				DB:     h.conn,
				Logger: zap.NewNop(),
			}),
			BillingQueueRepo: h.billingQueueRepo,
			ShipmentRepo: shipmentrepository.New(
				shipmentrepository.Params{DB: h.conn, Logger: zap.NewNop()},
			),
			ShipmentCtrlRepo: shipmentcontrolrepository.New(
				shipmentcontrolrepository.Params{DB: h.conn, Logger: zap.NewNop()},
			),
			AccountingRepo: h.accountingRepo,
			JournalRepo: journalpostingrepository.New(
				journalpostingrepository.Params{DB: h.conn, Logger: zap.NewNop()},
			),
			FiscalPeriodRepo: fiscalperiodrepository.New(
				fiscalperiodrepository.Params{DB: h.conn, Logger: zap.NewNop()},
			),
			CustomerLedgerRepo: customerledgerrepository.New(
				customerledgerrepository.Params{DB: h.conn, Logger: zap.NewNop()},
			),
			DocumentRepo: documentrepository.New(
				documentrepository.Params{DB: h.conn, Logger: zap.NewNop()},
			),
			Validator: invoiceadjustmentengine.NewValidator(
				invoiceadjustmentengine.ValidatorParams{},
			),
			AuditService: noopAuditService{},
			Commercial: shipmentcommercial.New(shipmentcommercial.Params{
				Logger:          zap.NewNop(),
				RateEngine:      ratingtest.NewFlatEngine(t, formulaAmount),
				AccessorialRepo: fakeAccessorialRepo{},
			}),
			Generator: &fakeGenerator{},
			SequenceGenerator: testutil.UniqueJournalSequenceGenerator{
				TestSequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "ACC-SEQ"},
			},
		}),
	})
}

func (h *integrationHarness) buildBillingQueueService() billingqueueports.BillingQueueService {
	logger := zap.NewNop()
	shipmentRepo := shipmentrepository.New(shipmentrepository.Params{
		DB: h.conn,
		MoveRepository: shipmentmoverepository.New(
			shipmentmoverepository.Params{DB: h.conn, Logger: logger},
		),
		AdditionalChargeRepository: shipmentadditionalchargerepository.New(
			shipmentadditionalchargerepository.Params{DB: h.conn, Logger: logger},
		),
		CommodityRepository: shipmentcommodityrepository.New(
			shipmentcommodityrepository.Params{DB: h.conn, Logger: logger},
		),
		ChargeAllocationRepository: chargeallocationrepository.New(
			chargeallocationrepository.Params{DB: h.conn, Logger: logger},
		),
		Logger: logger,
	})

	return billingqueueservice.New(billingqueueservice.Params{
		Logger:       logger,
		DB:           h.conn,
		Repo:         h.billingQueueRepo,
		ShipmentRepo: shipmentRepo,
		CommentRepo: shipmentcommentrepository.New(
			shipmentcommentrepository.Params{DB: h.conn, Logger: logger},
		),
		AdjustmentRepo: h.adjustmentRepo,
		AuditService:   noopAuditService{},
		Realtime:       noopRealtimeService{},
		Validator: billingqueueservice.NewValidator(
			billingqueueservice.ValidatorParams{DB: h.conn},
		),
	})
}

func (h *integrationHarness) ensureOpenFiscalPeriod(t *testing.T) {
	t.Helper()

	count, err := fiscalperioddb.FiscalPeriods.Query().Count(h.ctx, h.db)
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
	require.NoError(t, fiscalyeardb.FiscalYears.Create(h.ctx, h.db, year))
	require.NoError(t, fiscalperioddb.FiscalPeriods.Create(h.ctx, h.db, period))
}

func (h *integrationHarness) ensureAccountingDefaults(t *testing.T) {
	t.Helper()

	control, err := h.accountingRepo.GetByOrgID(h.ctx, h.orgID)
	require.NoError(t, err)
	control.DefaultARAccountID = h.lookupGLAccount(t, "1110")
	control.DefaultWriteOffAccountID = h.lookupGLAccount(t, "6940")
	control.DefaultRevenueAccountID = h.lookupGLAccount(t, "4000")
	control.AccountingBasis = accountingcontrol.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnInvoicePost
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	control.ManualJournalEntryPolicy = accountingcontrol.ManualJournalEntryPolicyAdjustmentOnly
	control.RequireManualJEApproval = true
	_, err = h.accountingRepo.Update(h.ctx, control)
	require.NoError(t, err)
}

type creditMemoJournalRow struct {
	SourceEventType string
	Status          string
	Lines           []journalEntryLineRow
}

func (h *integrationHarness) creditMemoJournal(
	t *testing.T,
	creditMemoID pulid.ID,
) *creditMemoJournalRow {
	t.Helper()

	var row creditMemoJournalRow
	var journalEntryID pulid.ID
	err := h.db.Pool().QueryRow(h.ctx,
		`SELECT source_event_type, status, journal_entry_id FROM journal_sources
		WHERE organization_id = $1 AND business_unit_id = $2
			AND source_object_type = 'Invoice' AND source_object_id = $3`,
		h.orgID, h.buID, creditMemoID.String(),
	).Scan(&row.SourceEventType, &row.Status, &journalEntryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	require.NoError(t, err)

	row.Lines = h.journalEntryLines(t, journalEntryID)
	return &row
}

func (h *integrationHarness) journalEntryLines(
	t *testing.T,
	journalEntryID pulid.ID,
) []journalEntryLineRow {
	t.Helper()

	rows, err := h.db.Pool().Query(h.ctx,
		`SELECT gl_account_id, debit_amount, credit_amount FROM journal_entry_lines
		WHERE organization_id = $1 AND business_unit_id = $2 AND journal_entry_id = $3
		ORDER BY line_number ASC`,
		h.orgID, h.buID, journalEntryID,
	)
	require.NoError(t, err)
	lines, err := pgx.CollectRows(rows, pgx.RowToStructByPos[journalEntryLineRow])
	require.NoError(t, err)
	return lines
}

type customerLedgerRow struct {
	AmountMinor      int64
	SourceEventType  string
	RelatedInvoiceID pulid.ID
}

func (h *integrationHarness) customerLedgerLines(
	t *testing.T,
	invoiceID pulid.ID,
) []customerLedgerRow {
	t.Helper()

	rows, err := h.db.Pool().Query(h.ctx,
		`SELECT amount_minor, source_event_type, related_invoice_id FROM customer_ledger_entries
		WHERE organization_id = $1 AND business_unit_id = $2 AND source_object_id = $3`,
		h.orgID, h.buID, invoiceID.String(),
	)
	require.NoError(t, err)
	lines, err := pgx.CollectRows(rows, pgx.RowToStructByPos[customerLedgerRow])
	require.NoError(t, err)
	return lines
}

func (h *integrationHarness) lookupGLAccount(t *testing.T, code string) pulid.ID {
	t.Helper()

	var id pulid.ID
	require.NoError(t, h.db.Pool().QueryRow(h.ctx,
		`SELECT id FROM gl_accounts
		WHERE organization_id = $1 AND business_unit_id = $2 AND account_code = $3 LIMIT 1`,
		h.orgID, h.buID, code,
	).Scan(&id))
	return id
}

func (h *integrationHarness) setControls(
	t *testing.T,
	mutate func(*invoiceadjustmentcontrol.InvoiceAdjustmentControl),
) {
	t.Helper()

	control, err := h.adjustmentCtrlRepo.GetByOrgID(h.ctx, h.orgID)
	require.NoError(t, err)
	mutate(control)
	_, err = h.adjustmentCtrlRepo.Update(h.ctx, control)
	require.NoError(t, err)
}

func (h *integrationHarness) createPostedInvoice(
	t *testing.T,
	lines []invoice.InvoiceLine,
	settlementStatus invoice.SettlementStatus,
	applied decimal.Decimal,
) *invoice.Invoice {
	t.Helper()

	h.nextNumber++
	item, err := h.billingQueueRepo.Create(h.ctx, &billingqueue.BillingQueueItem{
		OrganizationID:        h.orgID,
		BusinessUnitID:        h.buID,
		ShipmentID:            h.shipmentID,
		BillToCustomerID:      h.customerID,
		Number:                fmt.Sprintf("INV-%03d", h.nextNumber),
		Status:                billingqueue.StatusPosted,
		BillType:              billingqueue.BillTypeInvoice,
		RerateVariancePercent: decimal.Zero,
	})
	require.NoError(t, err)

	lineCopies := make([]*invoice.InvoiceLine, 0, len(lines))
	subtotal := decimal.Zero
	other := decimal.Zero
	total := decimal.Zero
	for idx := range lines {
		line := lines[idx]
		if line.Type == invoice.InvoiceLineTypeFreight {
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
		DueDate: invoice.DueDateFromPaymentTerm(
			timeutils.NowUnix(),
			invoice.PaymentTermNet30,
		),
		PostedAt:           new(timeutils.NowUnix()),
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

func (h *integrationHarness) actor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    h.userID,
		UserID:         h.userID,
		OrganizationID: h.orgID,
		BusinessUnitID: h.buID,
	}
}

func makeInvoiceLine(
	lineNumber int,
	lineType invoice.InvoiceLineType,
	description string,
	quantity, amount int64,
) invoice.InvoiceLine {
	qty := decimal.NewFromInt(quantity)
	total := decimal.NewFromInt(amount)
	return invoice.InvoiceLine{
		LineNumber:  lineNumber,
		Type:        lineType,
		Description: description,
		Quantity:    qty,
		UnitPrice:   total.Div(qty),
		Amount:      total,
	}
}

//go:fix inline
func ptrInt64(value int64) *int64 {
	return new(value)
}

func (h *integrationHarness) createDocument(
	t *testing.T,
	status document.Status,
) *document.Document {
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
	require.NoError(t, documentdb.Documents.Create(h.ctx, h.db, entity))
	return entity
}

func TestInvoiceAdjustmentSchemaAndSeeds(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	for _, tableName := range []string{
		"invoice_adjustments",
		"invoice_adjustment_lines",
		"invoice_adjustment_batches",
		"invoice_adjustment_batch_items",
		"invoice_correction_groups",
		"invoice_reconciliation_exceptions",
	} {
		var name string
		require.NoError(t, db.Pool().QueryRow(ctx,
			"SELECT to_regclass($1)::text", tableName,
		).Scan(&name))
		assert.Equal(t, tableName, name)
	}

	var isNullable, columnDefault string
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT is_nullable, COALESCE(column_default, '') FROM information_schema.columns
		WHERE table_name = $1 AND column_name = $2`,
		"billing_queue_items", "rerate_variance_percent",
	).Scan(&isNullable, &columnDefault))
	assert.Equal(t, "NO", isNullable)
	assert.Contains(t, columnDefault, "0")

	labelRows, err := db.Pool().Query(ctx,
		`SELECT e.enumlabel::text FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
		WHERE t.typname = $1 ORDER BY e.enumsortorder ASC`,
		"write_off_approval_policy_enum",
	)
	require.NoError(t, err)
	labels, err := pgx.CollectRows(labelRows, pgx.RowTo[string])
	require.NoError(t, err)
	assert.Equal(t, []string{
		"Disallow",
		"AlwaysRequireApproval",
		"RequireApprovalAboveThreshold",
	}, labels)

	var adjustmentControls, accountingControls, writeOffConfigured, arConfigured int
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT
			(SELECT COUNT(*) FROM invoice_adjustment_controls),
			(SELECT COUNT(*) FROM accounting_controls),
			(SELECT COUNT(*) FROM accounting_controls WHERE default_write_off_account_id IS NOT NULL),
			(SELECT COUNT(*) FROM accounting_controls WHERE default_ar_account_id IS NOT NULL)`,
	).Scan(&adjustmentControls, &accountingControls, &writeOffConfigured, &arConfigured))
	assert.GreaterOrEqual(t, adjustmentControls, 1)
	assert.GreaterOrEqual(t, accountingControls, 1)
	assert.Equal(t, accountingControls, writeOffConfigured)
	assert.Equal(t, accountingControls, arConfigured)
}
