//go:build integration

package exceptionsourcerepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/exceptionsourcerepository"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentprojection"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type harness struct {
	ctx     context.Context
	db      *keel.DB
	reader  watchtowerports.ExceptionSourceReader
	tenant  pagination.TenantInfo
	data    *seedtest.TestData
	fixture *testutil.ShipmentIntegrationFixture
	now     int64
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	return &harness{
		ctx: dbscope.WithTenant(ctx, tenant.DBTenant()),
		db:  db,
		reader: exceptionsourcerepository.New(
			exceptionsourcerepository.Params{DB: postgres.NewTestConnection(db)},
		),
		tenant:  tenant,
		data:    data,
		fixture: testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant),
		now:     timeutils.NowUnix(),
	}
}

func (h *harness) shipment(
	t *testing.T,
	code string,
	status shipmentvocab.Status,
) *testutil.ShipmentGraph {
	t.Helper()
	graph := testutil.CreateShipmentGraph(
		t,
		h.ctx,
		h.db,
		h.fixture,
		h.tenant,
		testutil.ShipmentGraphParams{
			BOL:          "BOL-" + code,
			ProNumber:    "PRO-" + code,
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipmentvocab.MoveStatus{shipmentvocab.MoveStatusNew},
		},
	)
	h.exec(t, `UPDATE shipments SET status = $1, owner_id = $2 WHERE id = $3`,
		string(status), h.data.User.ID.String(), graph.Shipment.ID.String())
	return graph
}

func (h *harness) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := keel.Exec(h.ctx, h.db, query, args...)
	require.NoError(t, err)
}

func (h *harness) window(t *testing.T, stop *shipment.Stop, end int64) {
	t.Helper()
	h.exec(
		t,
		`UPDATE stops SET scheduled_window_start = $1, scheduled_window_end = $2 WHERE id = $3`,
		end-1800,
		end,
		stop.ID.String(),
	)
}

func (h *harness) arrive(t *testing.T, stop *shipment.Stop, at int64) {
	t.Helper()
	h.exec(t, `UPDATE stops SET actual_arrival = $1, status = $2 WHERE id = $3`,
		at, string(shipmentvocab.StopStatusCompleted), stop.ID.String())
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, stop.ShipmentMoveID))
}

func TestListMissedStopsFindsClosedWindowsWithoutAnArrival(t *testing.T) {
	h := newHarness(t)

	missed := h.shipment(t, "MS-1", shipmentvocab.StatusAssigned)
	pickup, delivery := missed.Moves[0].Stops[0], missed.Moves[0].Stops[1]
	h.window(t, pickup, h.now-7200)
	h.window(t, delivery, h.now+7200)

	arrived := h.shipment(t, "MS-2", shipmentvocab.StatusInTransit)
	h.window(t, arrived.Moves[0].Stops[0], h.now-7200)
	h.arrive(t, arrived.Moves[0].Stops[0], h.now-7000)

	canceled := h.shipment(t, "MS-3", shipmentvocab.StatusCanceled)
	h.window(t, canceled.Moves[0].Stops[0], h.now-7200)

	stops, err := h.reader.ListMissedStops(h.ctx, &watchtowerports.ListMissedStopsRequest{
		TenantInfo: h.tenant,
		StopTypes: []shipmentvocab.StopType{
			shipmentvocab.StopTypePickup,
			shipmentvocab.StopTypeSplitPickup,
		},
		ClosedBefore: h.now - 1800,
	})
	require.NoError(t, err)
	require.Len(t, stops, 1)
	assert.Equal(t, pickup.ID, stops[0].StopID)
	assert.Equal(t, missed.Shipment.ID, stops[0].ShipmentID)
	assert.Equal(t, "PRO-MS-1", stops[0].ProNumber)
	assert.Equal(t, h.data.User.ID, stops[0].OwnerID)
	assert.Equal(t, h.now-7200, stops[0].Cutoff)
	assert.NotEmpty(t, stops[0].City)

	deliveries, err := h.reader.ListMissedStops(h.ctx, &watchtowerports.ListMissedStopsRequest{
		TenantInfo:   h.tenant,
		StopTypes:    []shipmentvocab.StopType{shipmentvocab.StopTypeDelivery},
		ClosedBefore: h.now - 1800,
	})
	require.NoError(t, err)
	assert.Empty(t, deliveries)

	scoped, err := h.reader.ListMissedStops(h.ctx, &watchtowerports.ListMissedStopsRequest{
		TenantInfo:   h.tenant,
		StopTypes:    []shipmentvocab.StopType{shipmentvocab.StopTypePickup},
		ClosedBefore: h.now - 1800,
		ShipmentIDs:  []pulid.ID{arrived.Shipment.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, scoped)
}

func (h *harness) podType(t *testing.T) *documenttype.DocumentType {
	t.Helper()
	docType := &documenttype.DocumentType{
		ID:             pulid.MustNew("dt_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Code:           documenttype.CodePOD,
		Name:           "Proof of delivery",
	}
	err := documenttypedb.DocumentTypes.Create(h.ctx, h.db, docType)
	require.NoError(t, err)
	return docType
}

func (h *harness) attach(
	t *testing.T,
	shipmentID pulid.ID,
	docType *documenttype.DocumentType,
	status document.Status,
) {
	t.Helper()
	typeID := docType.ID
	entity := &document.Document{
		ID:               pulid.MustNew("doc_"),
		OrganizationID:   h.tenant.OrgID,
		BusinessUnitID:   h.tenant.BuID,
		LineageID:        pulid.MustNew("dln_"),
		IsCurrentVersion: true,
		FileName:         "pod.pdf",
		OriginalName:     "pod.pdf",
		FileSize:         128,
		FileType:         "application/pdf",
		StoragePath:      "test/pod.pdf",
		Status:           status,
		ResourceID:       shipmentID.String(),
		ResourceType:     "shipment",
		UploadedByID:     h.data.User.ID,
		DocumentTypeID:   &typeID,
	}
	if status == document.StatusRejected {
		rejectedAt := h.now
		entity.RejectedByID = h.data.User.ID
		entity.RejectedAt = &rejectedAt
		entity.RejectionReason = "Illegible"
	}
	err := documentdb.Documents.Create(h.ctx, h.db, entity)
	require.NoError(t, err)
}

func TestListMissingPODsSkipsShipmentsWithAPodOnFile(t *testing.T) {
	h := newHarness(t)
	podType := h.podType(t)

	missing := h.shipment(t, "POD-1", shipmentvocab.StatusCompleted)
	h.arrive(t, missing.Moves[0].Stops[1], h.now-30*3600)

	filed := h.shipment(t, "POD-2", shipmentvocab.StatusCompleted)
	h.arrive(t, filed.Moves[0].Stops[1], h.now-30*3600)
	h.attach(t, filed.Shipment.ID, podType, document.StatusActive)

	rejected := h.shipment(t, "POD-3", shipmentvocab.StatusReadyToInvoice)
	h.arrive(t, rejected.Moves[0].Stops[1], h.now-30*3600)
	h.attach(t, rejected.Shipment.ID, podType, document.StatusRejected)

	recent := h.shipment(t, "POD-4", shipmentvocab.StatusCompleted)
	h.arrive(t, recent.Moves[0].Stops[1], h.now-3600)

	rows, err := h.reader.ListMissingPODs(h.ctx, &watchtowerports.ListMissingPODsRequest{
		TenantInfo:      h.tenant,
		DeliveredBefore: h.now - 24*3600,
		DeliveredAfter:  h.now - 90*24*3600,
	})
	require.NoError(t, err)
	ids := make([]pulid.ID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ShipmentID)
	}
	assert.ElementsMatch(t, []pulid.ID{missing.Shipment.ID, rejected.Shipment.ID}, ids)
	for _, row := range rows {
		assert.Equal(t, h.now-30*3600, row.DeliveredAt)
		assert.NotEmpty(t, row.CustomerName)
	}
}

func (h *harness) tender(
	t *testing.T,
	graph *testutil.ShipmentGraph,
	status tender.Status,
	createdAt int64,
) *tender.Tender {
	t.Helper()
	entity := &tender.Tender{
		ID:             pulid.MustNew("tnd_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		ShipmentID:     graph.Shipment.ID,
		ShipmentMoveID: graph.Moves[0].ID,
		Mode:           tender.ModeWaterfall,
		Status:         status,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	if status == tender.StatusExhausted {
		entity.ExhaustedAt = &createdAt
	}
	err := tenderdb.Tenders.Create(h.ctx, h.db, entity)
	require.NoError(t, err)
	h.exec(t, `UPDATE tenders SET created_at = $1 WHERE id = $2`, createdAt, entity.ID.String())
	return entity
}

func TestListExhaustedTendersOnlyForStillUncoveredMoves(t *testing.T) {
	h := newHarness(t)

	open := h.shipment(t, "TND-1", shipmentvocab.StatusNew)
	exhausted := h.tender(t, open, tender.StatusExhausted, h.now-600)

	retendered := h.shipment(t, "TND-2", shipmentvocab.StatusNew)
	h.tender(t, retendered, tender.StatusExhausted, h.now-900)
	h.tender(t, retendered, tender.StatusActive, h.now-300)

	assigned := h.shipment(t, "TND-3", shipmentvocab.StatusAssigned)
	h.tender(t, assigned, tender.StatusExhausted, h.now-600)
	h.exec(t, `UPDATE shipment_moves SET status = $1 WHERE id = $2`,
		string(shipmentvocab.MoveStatusAssigned), assigned.Moves[0].ID.String())

	rows, err := h.reader.ListExhaustedTenders(h.ctx, &watchtowerports.ListExhaustedTendersRequest{
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, exhausted.ID, rows[0].TenderID)
	assert.Equal(t, open.Moves[0].ID, rows[0].MoveID)
	assert.Equal(t, h.now-600, rows[0].ExhaustedAt)
	assert.Equal(t, h.data.User.ID, rows[0].OwnerID)
}
