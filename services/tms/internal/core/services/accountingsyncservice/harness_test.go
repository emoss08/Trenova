package accountingsyncservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/agenteventstest"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testAccessToken = "secret-access-token-value"

var (
	syncStartDate = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	syncEnabledAt = time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC).Unix()
	aprilTenth    = time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC).Unix()
	mayTenth      = time.Date(2026, time.May, 10, 0, 0, 0, 0, time.UTC).Unix()
)

type harness struct {
	svc         *Service
	enqueuer    *Enqueuer
	connections *fakeConnections
	records     *fakeRecords
	backfills   *fakeBackfills
	mappings    *fakeMappingStore
	connService *fakeConnService
	writer      *fakeWriter
	invoices    *fakeInvoices
	adjustments *fakeAdjustments
	payments    *fakePayments
	audit       *fakeAudit
	watchtower  *fakeWatchtower
	dispatcher  *fakeDispatcher
	events      *agenteventstest.Recorder
	tenant      pagination.TenantInfo
	userID      pulid.ID
	conn        *accountingsync.AccountingConnection
}

type harnessOption func(*harnessConfig)

type harnessConfig struct {
	withoutDispatcher bool
}

func withoutDispatcher() harnessOption {
	return func(c *harnessConfig) { c.withoutDispatcher = true }
}

func newHarness(t *testing.T, opts ...harnessOption) *harness {
	t.Helper()
	cfg := harnessConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	h := &harness{
		connections: newFakeConnections(),
		records:     newFakeRecords(),
		backfills:   newFakeBackfills(),
		mappings:    newFakeMappingStore(),
		writer:      newFakeWriter(),
		invoices:    &fakeInvoices{rows: map[pulid.ID]*invoice.Invoice{}},
		adjustments: &fakeAdjustments{rows: map[pulid.ID]*invoiceadjustment.InvoiceAdjustment{}},
		payments: &fakePayments{
			rows:         map[pulid.ID]*customerpayment.Payment{},
			applications: map[pulid.ID]*customerpayment.CreditMemoApplication{},
		},
		audit:      &fakeAudit{},
		watchtower: newFakeWatchtower(),
		dispatcher: &fakeDispatcher{},
		events:     &agenteventstest.Recorder{},
		tenant:     pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		userID:     pulid.MustNew("usr_"),
	}
	h.connService = &fakeConnService{
		connections: h.connections,
		writer:      h.writer,
		accessToken: testAccessToken,
	}
	h.conn = h.addConnection(syncingConnection)

	var dispatcher services.AccountingSyncDispatcher = h.dispatcher
	if cfg.withoutDispatcher {
		dispatcher = nil
	}
	h.enqueuer = NewEnqueuer(EnqueuerParams{
		Logger:      zap.NewNop(),
		Connections: h.connections,
		Records:     h.records,
		Mappings:    fakeMappingRepo{store: h.mappings},
		Dispatcher:  dispatcher,
	})
	h.svc = New(Params{
		Logger:            zap.NewNop(),
		DB:                dbtest.NopConnection{},
		Connections:       h.connections,
		Records:           h.records,
		Backfills:         h.backfills,
		Mappings:          fakeMappingRepo{store: h.mappings},
		MappingService:    fakeMappingService{store: h.mappings},
		ConnectionService: h.connService,
		Invoices:          h.invoices,
		Adjustments:       h.adjustments,
		Payments:          h.payments,
		Organizations:     fakeOrganizations{timezone: "UTC"},
		AuditService:      h.audit,
		Enqueuer:          h.enqueuer,
		Dispatcher:        dispatcher,
		Publisher:         h.events,
		Watchtower:        h.watchtower,
	})
	return h
}

func syncingConnection(conn *accountingsync.AccountingConnection) {
	start := syncStartDate
	enabled := syncEnabledAt
	conn.SetupStep = accountingsync.SetupStepComplete
	conn.SyncStartDate = &start
	conn.SyncEnabledAt = &enabled
	conn.AutoSync = true
}

func (h *harness) addConnection(
	configure func(*accountingsync.AccountingConnection),
) *accountingsync.AccountingConnection {
	conn := &accountingsync.AccountingConnection{
		ID:                   pulid.MustNew("acctc_"),
		OrganizationID:       h.tenant.OrgID,
		BusinessUnitID:       h.tenant.BuID,
		IntegrationType:      integration.TypeQuickBooksOnline,
		Status:               accountingsync.ConnectionStatusConnected,
		ExternalRealmID:      "9341452431742015",
		AppSource:            accountingsync.AppSourceInstance,
		ExternalHomeCurrency: "USD",
		SetupStep:            accountingsync.SetupStepMappings,
		ConnectedByID:        h.userID,
		ConnectedAt:          syncStartDate,
	}
	if configure != nil {
		configure(conn)
	}
	h.connections.put(conn)
	return cloneConnection(conn)
}

func (h *harness) updateConnection(fn func(*accountingsync.AccountingConnection)) {
	h.connections.mu.Lock()
	defer h.connections.mu.Unlock()
	fn(h.connections.rows[h.conn.ID])
}

func (h *harness) confirm(target mappingTarget, externalID string) *accountingsync.AccountingMapping {
	return h.mappings.set(h.conn, target, accountingsync.MappingStateConfirmed, externalID)
}

func customerTarget(customerID pulid.ID, name string) mappingTarget {
	return mappingTarget{TargetType: accountingsync.TargetCustomer, ObjectID: customerID, Label: name}
}

func freightTarget() mappingTarget {
	return mappingTarget{
		TargetType: accountingsync.TargetLineType,
		Key:        string(invoice.InvoiceLineTypeFreight),
		Label:      "Freight",
	}
}

func chargeTarget(chargeID pulid.ID) mappingTarget {
	return mappingTarget{TargetType: accountingsync.TargetAccessorialCharge, ObjectID: chargeID, Label: "DET"}
}

type invoiceSpec struct {
	number   string
	billType billingqueue.BillType
	date     int64
	currency string
	lines    []*invoice.InvoiceLine
}

func (h *harness) postedInvoice(customerID pulid.ID, spec invoiceSpec) *invoice.Invoice {
	if spec.billType == "" {
		spec.billType = billingqueue.BillTypeInvoice
	}
	if spec.date == 0 {
		spec.date = aprilTenth
	}
	if spec.currency == "" {
		spec.currency = "USD"
	}
	if spec.number == "" {
		spec.number = "INV-" + pulid.MustNew("n_").String()[2:8]
	}
	if spec.lines == nil {
		spec.lines = []*invoice.InvoiceLine{freightLine("1500.00")}
	}
	total := decimal.Zero
	for _, line := range spec.lines {
		total = total.Add(line.Amount)
	}
	due := spec.date + 30*24*60*60
	posted := spec.date + 3600
	inv := &invoice.Invoice{
		ID:                pulid.MustNew("inv_"),
		OrganizationID:    h.tenant.OrgID,
		BusinessUnitID:    h.tenant.BuID,
		CustomerID:        customerID,
		Number:            spec.number,
		BillType:          spec.billType,
		Status:            invoice.StatusPosted,
		PaymentTerm:       invoice.PaymentTermNet30,
		CurrencyCode:      spec.currency,
		InvoiceDate:       spec.date,
		DueDate:           &due,
		PostedAt:          &posted,
		ShipmentProNumber: "PRO-1001",
		ShipmentBOL:       "BOL-77",
		TotalAmount:       total,
		Lines:             spec.lines,
	}
	h.invoices.put(inv)
	return inv
}

func freightLine(amount string) *invoice.InvoiceLine {
	value := decimal.RequireFromString(amount)
	return &invoice.InvoiceLine{
		ID:          pulid.MustNew("invl_"),
		Type:        invoice.InvoiceLineTypeFreight,
		Description: "Linehaul",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   value,
		Amount:      value,
	}
}

func chargeLine(chargeID pulid.ID, amount string) *invoice.InvoiceLine {
	value := decimal.RequireFromString(amount)
	return &invoice.InvoiceLine{
		ID:                  pulid.MustNew("invl_"),
		Type:                invoice.InvoiceLineTypeAccessorial,
		Description:         "Detention",
		Quantity:            decimal.NewFromInt(2),
		UnitPrice:           value.Div(decimal.NewFromInt(2)),
		Amount:              value,
		AccessorialChargeID: chargeID,
		ChargeCode:          "DET",
	}
}

func (h *harness) enqueue(t *testing.T, req *services.AccountingSyncEnqueueRequest) {
	t.Helper()
	req.TenantInfo = h.tenant
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), req))
}

func (h *harness) enqueueInvoice(t *testing.T, inv *invoice.Invoice) *accountingsync.AccountingSyncRecord {
	t.Helper()
	require.NoError(t, h.enqueuer.Enqueue(
		t.Context(),
		services.InvoiceSyncRequest(inv, accountingsync.PostedSourceEvent(inv.BillType)),
	))
	return h.only(t, accountingsync.SyncObjectTypeForBill(inv.BillType), inv.ID, accountingsync.SyncOperationCreate)
}

func (h *harness) only(
	t *testing.T,
	objectType accountingsync.SyncObjectType,
	objectID pulid.ID,
	operation accountingsync.SyncOperation,
) *accountingsync.AccountingSyncRecord {
	t.Helper()
	found := h.records.find(objectType, objectID, operation)
	require.Len(t, found, 1, "expected one %s %s record", objectType, operation)
	return found[0]
}

func (h *harness) drain(t *testing.T) *services.AccountingSyncDrainResult {
	t.Helper()
	result, err := h.svc.Drain(t.Context(), &services.DrainAccountingSyncRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Lease:        time.Minute,
	})
	require.NoError(t, err)
	return result
}

func (h *harness) syncedInvoiceRecord(inv *invoice.Invoice, externalID string) *accountingsync.AccountingSyncRecord {
	req := services.InvoiceSyncRequest(inv, accountingsync.PostedSourceEvent(inv.BillType))
	record := NewRecordFor(h.conn, req, inv.InvoiceDate)
	record.MarkSynced(&accountingsync.SyncResult{ExternalID: externalID}, inv.InvoiceDate)
	h.records.put(record)
	return record
}

func (h *harness) eventKinds() []string {
	out := []string{}
	for _, event := range h.events.Published() {
		out = append(out, string(event.Kind))
	}
	return out
}

func nearNow(t *testing.T, want time.Duration, got *int64) {
	t.Helper()
	require.NotNil(t, got)
	expected := time.Now().Add(want).Unix()
	require.InDelta(t, expected, *got, 3, "expected about now+%s", want)
}
