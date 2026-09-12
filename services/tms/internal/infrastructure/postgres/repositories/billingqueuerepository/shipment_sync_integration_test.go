//go:build integration

package billingqueuerepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentrepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type syncTenant struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type syncShipmentRow struct {
	BillingTransferStatus  *string `bun:"billing_transfer_status"`
	TransferredToBillingAt *int64  `bun:"transferred_to_billing_at"`
}

type syncFixture struct {
	ctx          context.Context
	db           *bun.DB
	repo         *repository
	shipmentRepo repositories.ShipmentRepository
	tenant       pagination.TenantInfo
	shipmentID   pulid.ID
}

func setupSyncFixture(t *testing.T) *syncFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org syncTenant
	require.NoError(t, db.NewSelect().
		Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))

	var shipmentID pulid.ID
	require.NoError(t, db.NewSelect().
		Table("shipments").Column("id").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("created_at ASC").
		Limit(1).Scan(ctx, &shipmentID))

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()

	return &syncFixture{
		ctx:          ctx,
		db:           db,
		repo:         New(Params{DB: conn, Logger: logger}).(*repository),
		shipmentRepo: shipmentrepository.New(shipmentrepository.Params{DB: conn, Logger: logger}),
		tenant:       pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		shipmentID:   shipmentID,
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
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		ShipmentID:     f.shipmentID,
		Status:         status,
		BillType:       billType,
		Number:         number,
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
	require.NoError(t, f.db.NewSelect().
		Table("shipments").
		Column("billing_transfer_status", "transferred_to_billing_at").
		Where("id = ?", f.shipmentID).
		Scan(f.ctx, &row))
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
		billingqueue.StatusReadyForReview,
		billingqueue.BillTypeInvoice,
	)
	older = f.updateStatus(t, older, billingqueue.StatusSentBackToOps)

	newer := f.createItem(
		t,
		"BQ-SYNC-0102",
		billingqueue.StatusReadyForReview,
		billingqueue.BillTypeInvoice,
	)
	requireTransferStatus(t, f.shipmentRow(t), billingqueue.StatusReadyForReview)

	f.updateStatus(t, older, billingqueue.StatusCanceled)

	row := f.shipmentRow(t)
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

	entity, err := f.shipmentRepo.GetByID(f.ctx, &repositories.GetShipmentByIDRequest{
		ID:              f.shipmentID,
		TenantInfo:      f.tenant,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	require.NoError(t, err)

	entity.BillingTransferStatus = shipment.BillingTransferSentBackToOps
	entity.TransferredToBillingAt = nil
	entity, err = f.shipmentRepo.Update(f.ctx, entity)
	require.NoError(t, err)

	row := f.shipmentRow(t)
	requireTransferStatus(t, row, billingqueue.StatusInReview)
	require.Equal(t, transferredAt, row.TransferredToBillingAt)

	entity.BillingTransferStatus = shipment.BillingTransferNone
	_, err = f.shipmentRepo.UpdateDerivedState(f.ctx, entity)
	require.NoError(t, err)

	requireTransferStatus(t, f.shipmentRow(t), billingqueue.StatusInReview)
}
