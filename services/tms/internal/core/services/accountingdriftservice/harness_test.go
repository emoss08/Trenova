package accountingdriftservice

import (
	"slices"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
	tenant   pagination.TenantInfo
	conn     *accountingsync.AccountingConnection
	now      time.Time
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	enabled := checkTime.Add(-90 * 24 * time.Hour).Unix()
	start := enabled
	conn := &accountingsync.AccountingConnection{
		ID:                   pulid.MustNew("acctc_"),
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		IntegrationType:      integration.TypeQuickBooksOnline,
		Status:               accountingsync.ConnectionStatusConnected,
		ExternalRealmID:      "123",
		ExternalHomeCurrency: "USD",
		SetupStep:            accountingsync.SetupStepComplete,
		SyncStartDate:        &start,
		SyncEnabledAt:        &enabled,
		AutoSync:             true,
	}
	h := &harness{
		conns:    &fakeConnections{conn: conn},
		reader:   &fakeReader{maxRead: 200, docs: map[string]*services.AccountingDocumentState{}, omit: map[string]bool{}},
		records:  &fakeRecords{},
		findings: &fakeFindings{},
		source: &fakeSource{
			states: map[pulid.ID]*repositories.AccountingDriftState{},
		},
		events: &agenteventstest.Recorder{},
		tenant: tenantInfo,
		conn:   conn,
		now:    checkTime,
	}
	h.connSvc = &fakeConnService{connections: h.conns, reader: h.reader}
	h.svc = New(Params{
		Logger:            zap.NewNop(),
		DB:                dbtest.NopConnection{},
		Connections:       h.conns,
		ConnectionService: h.connSvc,
		Records:           h.records,
		Findings:          h.findings,
		Source:            h.source,
		Publisher:         h.events,
	})
	h.svc.now = func() time.Time { return h.now }
	return h
}

type sent struct {
	record *accountingsync.AccountingSyncRecord
	state  *repositories.AccountingDriftState
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
	state := &repositories.AccountingDriftState{
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
) *services.AccountingDocumentState {
	state := &services.AccountingDocumentState{
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

func (h *harness) reconcile(t *testing.T) *services.AccountingDriftBatchResult {
	t.Helper()
	result, err := h.svc.ReconcileBatch(t.Context(), &services.ReconcileAccountingDriftRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		EventBudget:  20,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	return result
}
