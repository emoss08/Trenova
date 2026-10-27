//go:build integration

package billingtransferrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer/billingtransferdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingtransferports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordedItem struct {
	ShipmentID         pulid.ID
	Sequence           int
	Status             string
	FailureCode        *string
	ErrorMessage       *string
	BillingQueueItemID *string
	BillingQueueNumber *string
	BillingQueueStatus *string
	ProcessedAt        *int64
}

func TestRecordItemOutcomesWritesEveryOutcomeOfAFlush(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID, userID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	require.NoError(t, db.Pool().QueryRow(ctx,
		`SELECT id FROM users WHERE current_organization_id = $1 AND business_unit_id = $2
		ORDER BY id LIMIT 1`,
		orgID, buID,
	).Scan(&userID))
	shipmentRows, err := db.Pool().Query(ctx,
		`SELECT id FROM shipments WHERE organization_id = $1 AND business_unit_id = $2
		ORDER BY id LIMIT 3`,
		orgID, buID,
	)
	require.NoError(t, err)
	shipments, err := pgx.CollectRows(shipmentRows, pgx.RowTo[pulid.ID])
	require.NoError(t, err)
	require.Len(t, shipments, 3)

	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	run, err := repo.Create(ctx, &billingtransfer.BillingTransferRun{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		RequestedByID:  userID,
		Status:         billingtransfer.RunStatusRunning,
		Scope:          billingtransfer.RunScopeSelected,
		BillType:       billingqueue.BillTypeInvoice,
		TotalCount:     len(shipments),
		QueuedAt:       timeutils.NowUnix(),
	})
	require.NoError(t, err)

	seeded, err := repo.SeedItems(ctx, &billingtransferports.SeedBillingTransferRunItemsRequest{
		TenantInfo:  tenant,
		RunID:       run.ID,
		ShipmentIDs: shipments,
	})
	require.NoError(t, err)
	require.Equal(t, len(shipments), seeded)

	queueItemID := pulid.MustNew("bqi_")
	progress, err := repo.RecordItemOutcomes(
		ctx,
		&billingtransferports.RecordBillingTransferOutcomesRequest{
			TenantInfo: tenant,
			RunID:      run.ID,
			Outcomes: []billingtransferports.BillingTransferItemOutcome{
				{
					ShipmentID: shipments[0],
					Status:     billingtransfer.ItemStatusTransferred,
				},
				{
					ShipmentID:   shipments[1],
					ProNumber:    "PRO-1",
					Status:       billingtransfer.ItemStatusNotTransferred,
					FailureCode:  billingtransfer.FailureRequirementsUnmet,
					ErrorMessage: "Billing requirements must be resolved first",
					MissingRequirements: []billingtransfer.MissingRequirement{
						{DocumentTypeID: "dt_1", DocumentTypeName: "POD"},
					},
					ProcessedAt: 42,
				},
				{
					ShipmentID:           shipments[2],
					ProNumber:            "PRO-2",
					Status:               billingtransfer.ItemStatusTransferred,
					MarkedReadyToInvoice: true,
					BillingQueueItemID:   queueItemID,
					BillingQueueNumber:   "INV-1",
					BillingQueueStatus:   billingqueue.StatusReadyForReview,
				},
			},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 3, progress.ProcessedCount)
	assert.Equal(t, 2, progress.TransferredCount)
	assert.Equal(t, 1, progress.NotTransferredCount)

	rows, err := db.Pool().Query(ctx,
		`SELECT shipment_id, sequence, status, failure_code, error_message,
			billing_queue_item_id, billing_queue_number, billing_queue_status::text, processed_at
		FROM billing_transfer_run_items
		WHERE organization_id = $1 AND business_unit_id = $2 AND run_id = $3
		ORDER BY sequence`,
		orgID, buID, run.ID,
	)
	require.NoError(t, err)
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[recordedItem])
	require.NoError(t, err)
	require.Len(t, items, 3)

	clean := items[0]
	assert.Equal(t, shipments[0], clean.ShipmentID)
	assert.Equal(t, 0, clean.Sequence)
	assert.Equal(t, string(billingtransfer.ItemStatusTransferred), clean.Status)
	assert.Nil(t, clean.FailureCode)
	assert.Nil(t, clean.ErrorMessage)
	assert.Nil(t, clean.BillingQueueItemID)
	assert.Nil(t, clean.BillingQueueNumber)
	assert.Nil(t, clean.BillingQueueStatus)
	require.NotNil(t, clean.ProcessedAt)
	assert.Positive(t, *clean.ProcessedAt)

	failed := items[1]
	assert.Equal(t, 1, failed.Sequence)
	require.NotNil(t, failed.FailureCode)
	assert.Equal(t, string(billingtransfer.FailureRequirementsUnmet), *failed.FailureCode)
	require.NotNil(t, failed.ErrorMessage)
	assert.Equal(t, "Billing requirements must be resolved first", *failed.ErrorMessage)
	assert.Nil(t, failed.BillingQueueStatus)
	require.NotNil(t, failed.ProcessedAt)
	assert.Equal(t, int64(42), *failed.ProcessedAt)

	transferred := items[2]
	require.NotNil(t, transferred.BillingQueueItemID)
	assert.Equal(t, queueItemID.String(), *transferred.BillingQueueItemID)
	require.NotNil(t, transferred.BillingQueueNumber)
	assert.Equal(t, "INV-1", *transferred.BillingQueueNumber)
	require.NotNil(t, transferred.BillingQueueStatus)
	assert.Equal(t, string(billingqueue.StatusReadyForReview), *transferred.BillingQueueStatus)

	item := billingtransferdb.BillingTransferRunItems
	reread, err := item.Where(item.RunID.Eq(run.ID), item.ShipmentID.Eq(shipments[1])).
		One(ctx, db)
	require.NoError(t, err)
	require.Len(t, reread.MissingRequirements, 1)
	assert.Equal(t, "POD", reread.MissingRequirements[0].DocumentTypeName)
	assert.Empty(t, reread.ValidationFailures)
	assert.Equal(t, "PRO-1", reread.ProNumber)
	assert.Equal(t, string(billingtransfer.ItemStatusTransferred), transferred.Status)
}
