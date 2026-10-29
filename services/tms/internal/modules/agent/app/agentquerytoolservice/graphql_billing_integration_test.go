//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/billingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerpaymentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoiceadjustmentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicedisputerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentengine"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicequeryservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// invoiceWithheldAtInternal is what DeskInvoiceDetail selects that the
// invoice's sensitivities keep Restricted: the money, the memo and the words
// about a failed send or a void.
var invoiceWithheldAtInternal = []string{
	"subtotalAmount", "otherAmount", "totalAmount", "appliedAmount", "openBalance", "memo",
	"remittanceInstructions", "voidReason", "lastSendError", "lastEdiError", "lines.quantity",
	"lines.unitPrice", "lines.amount",
}

// adjustmentWithheldAtInternal is what DeskInvoiceAdjustmentCard selects that
// stays Restricted: the reasons, the amounts and the quantities.
var adjustmentWithheldAtInternal = []string{
	"reason", "policyReason", "rejectionReason", "creditTotalAmount", "rebillTotalAmount",
	"lines.creditQuantity", "lines.creditAmount", "lines.rebillQuantity", "lines.rebillAmount",
	"lines.remainingEligibleAmount",
}

// billingSeed writes invoices, payments, disputes and adjustments for a
// tenant: each invoice bills a shipment of its own, as a posted one does.
type billingSeed struct {
	t        *testing.T
	admin    *keel.DB
	invoices invoiceports.InvoiceRepository
	fixtures map[pulid.ID]*testutil.ShipmentIntegrationFixture
}

func (b *billingSeed) fixture(tenant *agentreadtest.Tenant) *testutil.ShipmentIntegrationFixture {
	b.t.Helper()

	if fixture, ok := b.fixtures[tenant.Actor.OrganizationID]; ok {
		return fixture
	}
	fixture := testutil.SeedShipmentIntegrationFixture(b.t, b.t.Context(), b.admin,
		&seedtest.TestData{State: &usstate.UsState{ID: tenant.StateID}}, tenantInfoOf(tenant))
	b.fixtures[tenant.Actor.OrganizationID] = fixture

	return fixture
}

// invoice writes a posted invoice of two lines for its own shipment, owned
// by owner when one is named.
func (b *billingSeed) invoice(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	settlement invoice.SettlementStatus,
	owner pulid.ID,
) (*invoice.Invoice, error) {
	b.t.Helper()

	fixture := b.fixture(tenant)
	info := tenantInfoOf(tenant)
	graph := testutil.CreateShipmentGraph(b.t, b.t.Context(), b.admin, fixture, info,
		testutil.ShipmentGraphParams{
			BOL: "BOL" + label, ProNumber: "PRO" + label, ShipmentID: pulid.MustNew("shp_"),
		})
	if owner.IsNotNil() {
		shipments := shipmentdb.Shipments
		_, err := shipments.Where(shipments.ID.Eq(graph.Shipment.ID)).
			Update(tenant.Ctx(b.t.Context()), b.admin, shipments.OwnerID.To(owner))
		require.NoError(b.t, err)
	}
	queue := &billingqueue.BillingQueueItem{
		OrganizationID: info.OrgID, BusinessUnitID: info.BuID, ShipmentID: graph.Shipment.ID,
		BillToCustomerID: fixture.Customer.ID, Number: "INV-" + label,
		Status: billingqueue.StatusPosted, BillType: billingqueue.BillTypeInvoice,
	}
	require.NoError(
		b.t,
		billingqueuedb.BillingQueueItems.Create(tenant.Ctx(b.t.Context()), b.admin, queue),
	)

	posted := timeutils.NowUnix()
	due := posted + 30*86_400
	line := func(number int, amount int64) *invoice.InvoiceLine {
		return &invoice.InvoiceLine{
			LineNumber: number, Type: invoice.InvoiceLineTypeFreight, Description: "Linehaul",
			Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(amount),
			Amount: decimal.NewFromInt(amount),
		}
	}

	return b.invoices.Create(ctx, &invoice.Invoice{
		OrganizationID: info.OrgID, BusinessUnitID: info.BuID, BillingQueueItemID: queue.ID,
		ShipmentID: graph.Shipment.ID, CustomerID: fixture.Customer.ID, Number: queue.Number,
		BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted, PostedAt: &posted,
		PaymentTerm: invoice.PaymentTermNet30, CurrencyCode: "USD", InvoiceDate: posted,
		DueDate: &due, ShipmentProNumber: graph.Shipment.ProNumber,
		ShipmentBOL: graph.Shipment.BOL, BillToName: "Customer " + label,
		SubtotalAmount: decimal.NewFromInt(150), SubtotalAmountMinor: 15_000,
		TotalAmount: decimal.NewFromInt(150), TotalAmountMinor: 15_000,
		SettlementStatus: settlement, DisputeStatus: invoice.DisputeStatusNone,
		Memo:  "Rate agreed by phone",
		Lines: []*invoice.InvoiceLine{line(1, 100), line(2, 50)},
	})
}

// adjustment writes a credit drafted against a new invoice's first line.
func (b *billingSeed) adjustment(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
) (pulid.ID, error) {
	b.t.Helper()

	original, err := b.invoice(ctx, tenant, label, invoice.SettlementStatusUnpaid, pulid.Nil)
	if err != nil {
		return pulid.Nil, err
	}
	info := tenantInfoOf(tenant)
	group := &invoiceadjustment.InvoiceAdjustmentCorrectionGroup{
		OrganizationID: info.OrgID, BusinessUnitID: info.BuID, RootInvoiceID: original.ID,
	}
	tenantCtx := tenant.Ctx(b.t.Context())
	require.NoError(
		b.t,
		invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups.Create(tenantCtx, b.admin, group),
	)
	adjustment := &invoiceadjustment.InvoiceAdjustment{
		OrganizationID: info.OrgID, BusinessUnitID: info.BuID, CorrectionGroupID: group.ID,
		OriginalInvoiceID: original.ID, Kind: invoiceadjustment.KindCreditOnly,
		Status: invoiceadjustment.StatusDraft, IdempotencyKey: "adj-" + label,
		AccountingDate: timeutils.NowUnix(), Reason: "Detention billed twice",
		CreditTotalAmount: decimal.NewFromInt(-25), CreditTotalAmountMinor: -2_500,
		NetDeltaAmount: decimal.NewFromInt(-25), NetDeltaAmountMinor: -2_500,
	}
	require.NoError(
		b.t,
		invoiceadjustmentdb.InvoiceAdjustments.Create(tenantCtx, b.admin, adjustment),
	)
	require.NoError(
		b.t,
		invoiceadjustmentdb.InvoiceAdjustmentLines.Create(
			tenantCtx,
			b.admin,
			&invoiceadjustment.InvoiceAdjustmentLine{
				OrganizationID: info.OrgID, BusinessUnitID: info.BuID, AdjustmentID: adjustment.ID,
				OriginalInvoiceID: original.ID, OriginalLineID: original.Lines[0].ID, LineNumber: 1,
				Description: "Linehaul", CreditQuantity: decimal.NewFromInt(1),
				CreditAmount: decimal.NewFromInt(-25), CreditAmountMinor: -2_500,
			},
		),
	)

	return adjustment.ID, nil
}

func (b *billingSeed) dispute(
	tenant *agentreadtest.Tenant,
	disputed *invoice.Invoice,
	status invoicedispute.DisputeCaseStatus,
) pulid.ID {
	b.t.Helper()

	info := tenantInfoOf(tenant)
	row := &invoicedispute.InvoiceDispute{
		OrganizationID: info.OrgID, BusinessUnitID: info.BuID, InvoiceID: disputed.ID,
		CustomerID: disputed.CustomerID, Status: status,
		ReasonCode:     invoicedispute.DisputeReasonRateDiscrepancy,
		DisputedAmount: decimal.NewFromInt(40), DisputedAmountMinor: 4_000,
		Notes: "Customer says the rate was 110", OpenedByID: info.UserID,
		OpenedAt: timeutils.NowUnix(),
	}
	require.NoError(b.t, invoicedb.InvoiceDisputes.Create(tenant.Ctx(b.t.Context()), b.admin, row))

	return row.ID
}

func (b *billingSeed) payment(tenant *agentreadtest.Tenant, label string, reversed bool) pulid.ID {
	b.t.Helper()

	info := tenantInfoOf(tenant)
	status := customerpayment.StatusPosted
	if reversed {
		status = customerpayment.StatusReversed
	}
	row := &customerpayment.Payment{
		OrganizationID: info.OrgID, BusinessUnitID: info.BuID,
		CustomerID: b.fixture(tenant).Customer.ID, PaymentDate: timeutils.NowUnix(),
		AccountingDate: timeutils.NowUnix(), AmountMinor: 15_000, UnappliedAmountMinor: 15_000,
		Status: status, PaymentMethod: customerpayment.MethodACH,
		ReferenceNumber: "ACH" + label, Memo: "Remittance " + label, CurrencyCode: "USD",
		CreatedByID: info.UserID,
	}
	require.NoError(b.t, customerpaymentdb.Payments.Create(tenant.Ctx(b.t.Context()), b.admin, row))

	return row.ID
}

// billingServices are the repositories and services the billing reads
// resolve through, over the RLS connection.
func billingServices(
	t *testing.T,
	conn *postgres.Connection,
	admin *keel.DB,
) (*billingSeed, agentread.ResolverParams, invoiceports.InvoiceRepository) {
	t.Helper()

	logger := zap.NewNop()
	invoices := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	_, services := convertedListCases(conn, admin)
	services.InvoiceQueryService = invoicequeryservice.New(invoicequeryservice.Params{
		Repo: invoices,
	})
	services.CustomerPaymentRepo = customerpaymentrepository.New(
		customerpaymentrepository.Params{DB: conn, Logger: logger})
	services.InvoiceDisputeRepo = invoicedisputerepository.New(
		invoicedisputerepository.Params{DB: conn, Logger: logger})
	services.InvoiceAdjustmentService = invoiceadjustmentservice.New(
		invoiceadjustmentservice.Params{
			Repo: invoiceadjustmentrepository.New(
				invoiceadjustmentrepository.Params{DB: conn, Logger: logger},
			),
			InvoiceRepo: invoices,
			Engine: invoiceadjustmentengine.New(invoiceadjustmentengine.Params{
				Logger: logger,
				Repo: invoiceadjustmentrepository.New(
					invoiceadjustmentrepository.Params{DB: conn, Logger: logger},
				),
				DocumentRepo: documentrepository.New(
					documentrepository.Params{DB: conn, Logger: logger},
				),
				InvoiceRepo: invoices,
			}),
		})

	seed := &billingSeed{
		t: t, admin: admin, invoices: invoices,
		fixtures: make(map[pulid.ID]*testutil.ShipmentIntegrationFixture),
	}

	return seed, services, invoices
}

func TestConvertedLists_BillingReadsReadWhatTheRepositoryToolsReadUnderRowLevelSecurity(
	t *testing.T,
) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	seed, services, invoices := billingServices(t, conn, admin)
	engine := &mocks.DenyResourcesPermissionEngine{}
	readers := &convertedListEnv{
		reader: agentreadtest.NewReader(
			t,
			agentreadtest.Options{Engine: engine, Resolvers: services},
		),
		public: agentreadtest.NewReader(t, agentreadtest.Options{
			Engine: &publicRole{}, Resolvers: services,
		}),
		engine: engine,
		mine:   agentreadtest.SeedTenant(t, admin, "A"),
		theirs: agentreadtest.SeedTenant(t, admin, "B"),
	}

	cases := []referenceCase{
		{
			tool:   "invoice_list",
			marked: map[string]any{"settlementStatus": []any{"Paid"}},
			oldMarked: map[string]any{"filters": []any{map[string]any{
				"field": "settlementStatus", "operator": "in", "values": []any{"Paid"},
			}}},
			old: billingquerytools.NewListInvoicesTool(invoices, engine),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				settlement := invoice.SettlementStatusUnpaid
				if marked {
					settlement = invoice.SettlementStatusPaid
				}
				row, err := seed.invoice(ctx, tenant, label, settlement, pulid.Nil)
				return created(row, err)
			},
		},
		{
			tool:                 "receivable_list_customer_payments",
			readsActiveByDefault: true,
			marked:               map[string]any{"status": []any{"Reversed"}},
			write: func(_ context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				return seed.payment(tenant, label, marked), nil
			},
		},
	}
	for idx := range cases {
		c := &cases[idx]
		t.Run(c.tool, func(t *testing.T) {
			checkConvertedList(t, c, readers)
		})
	}
}

// The total an invoice list withheld at the default agent data access, it
// still withholds, and the amounts and memo of a payment, which the
// repository tool returned to every reader, are withheld now.
func TestConvertedLists_BillingReadsWithholdTheMoneyAtInternal(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	seed, services, invoices := billingServices(t, conn, admin)
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	tenant := agentreadtest.SeedTenant(t, admin, "A")
	ctx := readscope.With(dbscope.WithTenant(t.Context(), tenant.Scope()), nil)
	_, err := seed.invoice(ctx, tenant, "A1", invoice.SettlementStatusUnpaid, pulid.Nil)
	require.NoError(t, err)
	seed.payment(tenant, "A1", false)

	read := func(tool agenttoolports.AgentQueryTool) map[string]any {
		data, readErr := tool.Query(ctx, &agenttoolports.QueryToolParams{
			OrganizationID:    tenant.Actor.OrganizationID,
			BusinessUnitID:    tenant.Actor.BusinessUnitID,
			Actor:             tenant.Actor,
			Timezone:          "America/Chicago",
			Params:            map[string]any{"limit": 10},
			DataAccessCeiling: permission.SensitivityInternal,
			RunID:             pulid.MustNew("ar_"),
		})
		require.NoError(t, readErr)
		return asMap(t, data)
	}

	invoiceList, err := agentquerytoolservice.ConvertedListTool("invoice_list", reader, engine)
	require.NoError(t, err)
	converted := read(invoiceList)
	before := read(billingquerytools.NewListInvoicesTool(invoices, engine))
	for _, field := range before["withheldByAccess"].([]any) {
		assert.Contains(t, converted["withheldByAccess"], field,
			"the converted list withholds at least what the repository tool did")
	}
	assert.ElementsMatch(t, []any{"totalAmount"}, converted["withheldByAccess"])

	payments, err := agentquerytoolservice.ConvertedListTool(
		"receivable_list_customer_payments", reader, engine)
	require.NoError(t, err)
	assert.ElementsMatch(t, []any{
		"amountMinor", "appliedAmountMinor", "unappliedAmountMinor", "memo",
	}, read(payments)["withheldByAccess"])
}

// An invoice a person may read only because they own its shipment is read
// alike by the repository list and the GraphQL list.
func TestConvertedLists_InvoiceListIsNarrowedToOwnShipmentsOnBothPaths(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	seed, services, invoices := billingServices(t, conn, admin)
	tenant := agentreadtest.SeedTenant(t, admin, "A")
	tenantCtx := dbscope.WithTenant(t.Context(), tenant.Scope())

	own := make([]string, 0, 2)
	for idx := range 5 {
		owner := pulid.Nil
		if idx < 2 {
			owner = tenant.Actor.UserID
		}
		row, err := seed.invoice(readscope.With(tenantCtx, nil), tenant, fmt.Sprintf("O%d", idx),
			invoice.SettlementStatusUnpaid, owner)
		require.NoError(t, err)
		if idx < 2 {
			own = append(own, row.ID.String())
		}
	}

	scoped := readscope.With(tenantCtx, func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{"invoice": {Scope: readscope.ScopeOwn}}, nil
	})
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	converted, err := agentquerytoolservice.ConvertedListTool("invoice_list", reader, engine)
	require.NoError(t, err)
	old := billingquerytools.NewListInvoicesTool(invoices, engine)

	before, _ := walk(t, scoped, old, tenant.Actor, nil)
	after, _ := walk(t, scoped, converted, tenant.Actor, nil)
	assert.ElementsMatch(t, own, before, "the repository list reads only the owner's invoices")
	assert.ElementsMatch(t, before, after, "the GraphQL list reads what the repository list read")

	everyone, _ := walk(t, readscope.With(tenantCtx, nil), converted, tenant.Actor, nil)
	assert.Len(t, everyone, 5, "without a narrow grant every invoice is read")
}

// The dispute cases on the invoices named are read, every case of ours once
// and none of another tenant's, with the amount and notes withheld at the
// default agent data access and every withheld field absent at Public.
func TestConvertedLists_InvoiceDisputesReadTheNamedInvoicesUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	seed, services, _ := billingServices(t, conn, admin)
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	public := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &publicRole{}, Resolvers: services,
	})
	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	mineCtx := readscope.With(dbscope.WithTenant(t.Context(), mine.Scope()), nil)
	theirsCtx := readscope.With(dbscope.WithTenant(t.Context(), theirs.Scope()), nil)

	var named, ours []string
	for idx := range 2 {
		row, err := seed.invoice(mineCtx, mine, fmt.Sprintf("D%d", idx),
			invoice.SettlementStatusUnpaid, pulid.Nil)
		require.NoError(t, err)
		named = append(named, row.ID.String())
		ours = append(ours,
			seed.dispute(mine, row, invoicedispute.DisputeCaseStatusOpen).String(),
			seed.dispute(mine, row, invoicedispute.DisputeCaseStatusWithdrawn).String())
	}
	other, err := seed.invoice(theirsCtx, theirs, "X1", invoice.SettlementStatusUnpaid, pulid.Nil)
	require.NoError(t, err)
	seed.dispute(theirs, other, invoicedispute.DisputeCaseStatusOpen)
	call := map[string]any{"invoiceIds": []any{named[0], named[1], other.ID.String()}}

	tool, err := agentquerytoolservice.ConvertedListTool("invoice_list_disputes", reader, engine)
	require.NoError(t, err)
	data, err := tool.Query(mineCtx, &agenttoolports.QueryToolParams{
		OrganizationID:    mine.Actor.OrganizationID,
		BusinessUnitID:    mine.Actor.BusinessUnitID,
		Actor:             mine.Actor,
		Timezone:          "America/Chicago",
		Params:            call,
		DataAccessCeiling: permission.SensitivityInternal,
		RunID:             pulid.MustNew("ar_"),
	})
	require.NoError(t, err)
	page := asMap(t, data)
	assert.ElementsMatch(t, ours, itemIDs(page), "every case of ours once, none of theirs")
	assert.ElementsMatch(t, []any{"disputedAmount", "notes", "resolutionNotes"},
		page["withheldByAccess"])

	hidden, err := agentquerytoolservice.ConvertedListTool(
		"invoice_list_disputes",
		public,
		&publicRole{},
	)
	require.NoError(t, err)
	page = callTool(t, mineCtx, hidden, mine.Actor, call)
	withheld, _ := page["withheldByAccess"].([]any)
	require.NotEmpty(t, withheld)
	for _, item := range page["items"].([]any) {
		row, _ := item.(map[string]any)
		assert.Contains(t, row, "id")
		for _, field := range withheld {
			assert.NotContains(t, row, field)
		}
	}
}

func TestConvertedGets_InvoicesAndAdjustmentsReadOneRecordUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	seed, services, _ := billingServices(t, conn, admin)
	engine := &mocks.DenyResourcesPermissionEngine{}
	readers := &convertedListEnv{
		reader: agentreadtest.NewReader(
			t,
			agentreadtest.Options{Engine: engine, Resolvers: services},
		),
		public: agentreadtest.NewReader(t, agentreadtest.Options{
			Engine: &publicRole{}, Resolvers: services,
		}),
		engine: engine,
		mine:   agentreadtest.SeedTenant(t, admin, "A"),
		theirs: agentreadtest.SeedTenant(t, admin, "B"),
	}

	t.Run("invoice_get", func(t *testing.T) {
		checkConvertedGet(t, "invoice_get", &referenceCase{
			tool:               "invoice_get",
			withheldAtInternal: invoiceWithheldAtInternal,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, _ bool) (pulid.ID, error) {
				row, err := seed.invoice(
					ctx,
					tenant,
					label,
					invoice.SettlementStatusUnpaid,
					pulid.Nil,
				)
				return created(row, err)
			},
		}, readers, recordID)
	})
	t.Run("invoice_get_adjustment", func(t *testing.T) {
		checkConvertedGet(t, "invoice_get_adjustment", &referenceCase{
			tool:               "invoice_get_adjustment",
			withheldAtInternal: adjustmentWithheldAtInternal,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, _ bool) (pulid.ID, error) {
				return seed.adjustment(ctx, tenant, "J"+label)
			},
		}, readers, recordID)
	})
}
