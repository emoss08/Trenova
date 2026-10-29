//go:build integration

package billingqueuerepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/chargeallocationrepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentadditionalchargerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommodityrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type syncShipmentRow struct {
	BillingTransferStatus  *string
	TransferredToBillingAt *int64
}

type syncFixture struct {
	ctx          context.Context
	db           *keel.DB
	repo         *repository
	shipmentRepo shipmentports.ShipmentRepository
	tenant       pagination.TenantInfo
	shipmentID   pulid.ID
	payerID      pulid.ID
}

func setupSyncFixture(t *testing.T) *syncFixture {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID, shipmentID, payerID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT id, COALESCE(bill_to_customer_id, customer_id) FROM shipments
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY created_at ASC LIMIT 1`,
		orgID, buID,
	).Scan(&shipmentID, &payerID))

	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()

	return &syncFixture{
		ctx:  ctx,
		db:   db,
		repo: New(Params{DB: conn, Logger: logger}).(*repository),
		shipmentRepo: shipmentrepository.New(shipmentrepository.Params{
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
		}),
		tenant:     tenant,
		shipmentID: shipmentID,
		payerID:    payerID,
	}
}

func (f *syncFixture) createItem(
	t *testing.T,
	number string,
	status billingqueue.Status,
	billType billingqueue.BillType,
) *billingqueue.BillingQueueItem {
	t.Helper()

	item, err := f.repo.Create(f.ctx, &billingqueue.BillingQueueItem{
		OrganizationID:   f.tenant.OrgID,
		BusinessUnitID:   f.tenant.BuID,
		ShipmentID:       f.shipmentID,
		BillToCustomerID: f.payerID,
		Status:           status,
		BillType:         billType,
		Number:           number,
	})
	require.NoError(t, err)
	return item
}

func (f *syncFixture) updateStatus(
	t *testing.T,
	item *billingqueue.BillingQueueItem,
	status billingqueue.Status,
) *billingqueue.BillingQueueItem {
	t.Helper()

	item.Status = status
	updated, err := f.repo.Update(f.ctx, item)
	require.NoError(t, err)
	return updated
}

func (f *syncFixture) shipmentRow(t *testing.T) syncShipmentRow {
	t.Helper()

	var row syncShipmentRow
	require.NoError(t, f.db.Pool().QueryRow(f.ctx,
		"SELECT billing_transfer_status, transferred_to_billing_at FROM shipments WHERE id = $1",
		f.shipmentID,
	).Scan(&row.BillingTransferStatus, &row.TransferredToBillingAt))
	return row
}

func requireTransferStatus(t *testing.T, row syncShipmentRow, want billingqueue.Status) {
	t.Helper()

	require.NotNil(t, row.BillingTransferStatus)
	require.Equal(t, string(want), *row.BillingTransferStatus)
}

func TestShipmentMirrorsItsBillingQueueItem(t *testing.T) {
	f := setupSyncFixture(t)

	item := f.createItem(
		t,
		"BQ-SYNC-0001",
		billingqueue.StatusReadyForReview,
		billingqueue.BillTypeInvoice,
	)

	row := f.shipmentRow(t)
	requireTransferStatus(t, row, billingqueue.StatusReadyForReview)
	require.NotNil(t, row.TransferredToBillingAt)
	require.Equal(t, item.CreatedAt, *row.TransferredToBillingAt)

	item = f.updateStatus(t, item, billingqueue.StatusInReview)
	requireTransferStatus(t, f.shipmentRow(t), billingqueue.StatusInReview)

	item = f.updateStatus(t, item, billingqueue.StatusApproved)
	requireTransferStatus(t, f.shipmentRow(t), billingqueue.StatusApproved)

	f.updateStatus(t, item, billingqueue.StatusPosted)
	requireTransferStatus(t, f.shipmentRow(t), billingqueue.StatusPosted)
}

func TestShipmentFollowsItsLatestInvoiceItem(t *testing.T) {
	f := setupSyncFixture(t)

	older := f.createItem(
		t,
		"BQ-SYNC-0101",
		billingqueue.StatusPosted,
		billingqueue.BillTypeInvoice,
	)
	requireTransferStatus(t, f.shipmentRow(t), billingqueue.StatusPosted)

	newer := f.createItem(
		t,
		"BQ-SYNC-0102",
		billingqueue.StatusReadyForReview,
		billingqueue.BillTypeInvoice,
	)
	row := f.shipmentRow(t)
	requireTransferStatus(t, row, billingqueue.StatusReadyForReview)
	require.Equal(t, newer.CreatedAt, *row.TransferredToBillingAt)

	f.updateStatus(t, older, billingqueue.StatusCanceled)

	row = f.shipmentRow(t)
	requireTransferStatus(t, row, billingqueue.StatusReadyForReview)
	require.Equal(t, newer.CreatedAt, *row.TransferredToBillingAt)
}

func TestCreditMemoItemsDoNotMoveTheShipment(t *testing.T) {
	f := setupSyncFixture(t)

	f.createItem(t, "BQ-SYNC-0201", billingqueue.StatusInReview, billingqueue.BillTypeInvoice)
	f.createItem(t, "BQ-SYNC-0202", billingqueue.StatusPosted, billingqueue.BillTypeCreditMemo)

	requireTransferStatus(t, f.shipmentRow(t), billingqueue.StatusInReview)
}

func TestShipmentWritesCannotOverwriteTheMirror(t *testing.T) {
	f := setupSyncFixture(t)

	f.createItem(t, "BQ-SYNC-0301", billingqueue.StatusInReview, billingqueue.BillTypeInvoice)
	transferredAt := f.shipmentRow(t).TransferredToBillingAt

	entity, err := f.shipmentRepo.GetByID(f.ctx, &shipmentports.GetShipmentByIDRequest{
		ID:              f.shipmentID,
		TenantInfo:      f.tenant,
		ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
	})
	require.NoError(t, err)

	entity.BillingTransferStatus = shipmentvocab.BillingTransferSentBackToOps
	entity.TransferredToBillingAt = nil
	entity, err = f.shipmentRepo.Update(f.ctx, entity)
	require.NoError(t, err)

	row := f.shipmentRow(t)
	requireTransferStatus(t, row, billingqueue.StatusInReview)
	require.Equal(t, transferredAt, row.TransferredToBillingAt)

	entity.BillingTransferStatus = shipmentvocab.BillingTransferNone
	_, err = f.shipmentRepo.UpdateDerivedState(f.ctx, entity)
	require.NoError(t, err)

	requireTransferStatus(t, f.shipmentRow(t), billingqueue.StatusInReview)
}
