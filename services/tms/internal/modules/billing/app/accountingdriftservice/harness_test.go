package accountingdriftservice

import (
	"slices"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/agenteventstest"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var checkTime = time.Date(2026, time.March, 9, 6, 13, 0, 0, time.UTC)

type harness struct {
	svc      *Service
	conns    *fakeConnections
	connSvc  *fakeConnService
	reader   *fakeReader
	records  *fakeRecords
	findings *fakeFindings
	source   *fakeSource
	events   *agenteventstest.Recorder
	perms    *fakePermissions
	invRepo  *fakeInvoiceRepo
	invoices *fakeInvoices
	payments *fakePayments
	controls *fakeControls
	audit    *fakeAudit
	kicks    *fakeDispatcher
	other    *accountingconnection.AccountingConnection
	tenant   pagination.TenantInfo
	conn     *accountingconnection.AccountingConnection
	now      time.Time
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	enabled := checkTime.Add(-90 * 24 * time.Hour).Unix()
	start := enabled
	conn := &accountingconnection.AccountingConnection{
		ID:                   pulid.MustNew("acctc_"),
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		IntegrationType:      integration.TypeQuickBooksOnline,
		Status:               accountingconnection.ConnectionStatusConnected,
		ExternalRealmID:      "123",
		ExternalHomeCurrency: "USD",
		SetupStep:            accountingconnection.SetupStepComplete,
		SyncStartDate:        &start,
		SyncEnabledAt:        &enabled,
		AutoSync:             true,
	}
	h := &harness{
		conns: &fakeConnections{conn: conn},
		reader: &fakeReader{
			maxRead: 200,
			docs:    map[string]*accountingconnectorports.AccountingDocumentState{},
			omit:    map[string]bool{},
		},
		records:  &fakeRecords{},
		findings: &fakeFindings{},
		source: &fakeSource{
			states: map[pulid.ID]*accountingdriftports.AccountingDriftState{},
		},
		events: &agenteventstest.Recorder{},
		tenant: tenantInfo,
		conn:   conn,
		now:    checkTime,
	}
	h.connSvc = &fakeConnService{connections: h.conns, reader: h.reader}
	h.other = &accountingconnection.AccountingConnection{
		ID:             pulid.MustNew("acctc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		AutoSync:       true,
	}
	book := &bookEnqueuer{
		records: h.records,
		conns:   []*accountingconnection.AccountingConnection{conn, h.other},
		at:      checkTime.Unix(),
	}
	h.perms = &fakePermissions{denied: map[string]bool{}}
	h.invRepo = &fakeInvoiceRepo{rows: map[pulid.ID]*invoice.Invoice{}}
	h.invoices = &fakeInvoices{book: book}
	h.payments = &fakePayments{book: book, rows: map[pulid.ID]*customerpayment.Payment{}}
	h.controls = &fakeControls{tolerance: "5.00"}
	h.audit = &fakeAudit{}
	h.kicks = &fakeDispatcher{}
	h.svc = New(Params{
		Logger:       zap.NewNop(),
		DB:           dbtest.NopConnection{},
		Connections:  h.conns,
		Sessions:     h.connSvc,
		Records:      h.records,
		Findings:     h.findings,
		Source:       h.source,
		InvoiceRepo:  h.invRepo,
		Controls:     h.controls,
		Memos:        h.invoices,
		Voids:        h.invoices,
		Payments:     h.payments,
		Permissions:  h.perms,
		AuditService: h.audit,
		Dispatcher:   h.kicks,
		Publisher:    h.events,
	})
	h.svc.now = func() time.Time { return h.now }
	return h
}

type sent struct {
	record *accountingsync.AccountingSyncRecord
	state  *accountingdriftports.AccountingDriftState
}

func (h *harness) synced(
	objectType accountingsync.SyncObjectType,
	externalID string,
	amountMinor int64,
) *sent {
	objectID := pulid.MustNew("obj_")
	record := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: objectType,
			ObjectID:   objectID,
			Operation:  accountingsync.SyncOperationCreate,
			Revision:   1,
		},
		ObjectNumber: "DOC-" + externalID,
		SourceEvent:  accountingsync.SyncSourceInvoicePosted,
		At:           h.now.Add(-time.Hour).Unix(),
	})
	record.Status = accountingsync.SyncStatusSynced
	record.ExternalID = externalID
	h.records.add(record)
	h.source.records = append(h.source.records, record)
	slices.SortFunc(h.source.records, func(a, b *accountingsync.AccountingSyncRecord) int {
		switch {
		case a.ID.String() < b.ID.String():
			return -1
		case a.ID.String() > b.ID.String():
			return 1
		default:
			return 0
		}
	})
	state := &accountingdriftports.AccountingDriftState{
		ObjectID:     objectID,
		Number:       "DOC-" + externalID,
		PartyID:      pulid.MustNew("cus_"),
		PartyName:    "Acme Freight",
		CurrencyCode: "USD",
		AmountMinor:  amountMinor,
		OpenMinor:    amountMinor,
		State:        "Posted",
	}
	h.source.states[objectID] = state
	return &sent{record: record, state: state}
}

func (h *harness) provider(
	objectType accountingsync.SyncObjectType,
	externalID, total string,
) *accountingconnectorports.AccountingDocumentState {
	state := &accountingconnectorports.AccountingDocumentState{
		ExternalID:   externalID,
		Found:        true,
		Total:        decimalOf(total),
		CurrencyCode: "USD",
		ModifiedAt:   h.now.Add(-2 * time.Hour).Unix(),
		ModifiedBy:   "Pat Bookkeeper",
	}
	h.reader.set(objectType, state)
	return state
}

func (h *harness) reconcile(t *testing.T) *accountingdriftports.AccountingDriftBatchResult {
	t.Helper()
	result, err := h.svc.ReconcileBatch(
		t.Context(),
		&accountingdriftports.ReconcileAccountingDriftRequest{
			TenantInfo:   h.tenant,
			ConnectionID: h.conn.ID,
			EventBudget:  20,
		},
	)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	return result
}
