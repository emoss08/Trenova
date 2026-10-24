//go:build integration

package driversettlementrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/driverpayrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/driversettlementrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSettlementCardBatchReads_Integration(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var workerRow struct {
		ID             pulid.ID `bun:"id"`
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("workers").
		ColumnExpr("id, organization_id, business_unit_id").
		Limit(1).
		Scan(ctx, &workerRow))
	tenant := pagination.TenantInfo{OrgID: workerRow.OrganizationID, BuID: workerRow.BusinessUnitID}
	var userID pulid.ID
	require.NoError(t, db.NewSelect().TableExpr("users").Column("id").Limit(1).Scan(ctx, &userID))

	advance := &driverpay.PayAdvance{
		ID:             pulid.MustNew("padv_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		WorkerID:       workerRow.ID,
		Status:         driverpay.AdvanceStatusPartiallyRecovered,
		Source:         driverpay.AdvanceSourceCash,
		IssuedDate:     1_790_000_000,
		AmountMinor:    50000,
		RecoveredMinor: 20000,
		CurrencyCode:   "USD",
	}
	_, err := db.NewInsert().Model(advance).Exec(ctx)
	require.NoError(t, err)
	escrow := &driverpay.EscrowAccount{
		ID:                pulid.MustNew("escr_"),
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		WorkerID:          workerRow.ID,
		Status:            driverpay.EscrowAccountStatusActive,
		TargetAmountMinor: 250000,
		BalanceMinor:      190000,
		OpenedDate:        1_780_000_000,
		CurrencyCode:      "USD",
	}
	_, err = db.NewInsert().Model(escrow).Exec(ctx)
	require.NoError(t, err)

	settlements := make([]*driversettlement.Settlement, 0, 2)
	for _, number := range []string{"DS-CARD-1", "DS-CARD-2"} {
		settlement := &driversettlement.Settlement{
			ID:               pulid.MustNew("dstl_"),
			OrganizationID:   tenant.OrgID,
			BusinessUnitID:   tenant.BuID,
			WorkerID:         workerRow.ID,
			SettlementNumber: number,
			Status:           driversettlement.StatusPendingApproval,
			Classification:   driverpay.PayeeClassificationCompanyDriver,
			PeriodStart:      1_790_000_000,
			PeriodEnd:        1_790_600_000,
			PayDate:          1_790_900_000,
			CurrencyCode:     "USD",
		}
		_, err = db.NewInsert().Model(settlement).Exec(ctx)
		require.NoError(t, err)
		settlements = append(settlements, settlement)
	}
	disputed, quiet := settlements[0], settlements[1]
	disputes := make([]*driversettlement.Dispute, 0, 2)
	for _, created := range []int64{1_790_100_000, 1_790_200_000} {
		dispute := &driversettlement.Dispute{
			ID:                pulid.MustNew("dsd_"),
			OrganizationID:    tenant.OrgID,
			BusinessUnitID:    tenant.BuID,
			SettlementID:      disputed.ID,
			WorkerID:          workerRow.ID,
			Status:            driversettlement.DisputeStatusOpen,
			Category:          driversettlement.DisputeCategoryMissingPay,
			Description:       "Stop pay missing",
			SubmittedByUserID: userID,
		}
		_, err = db.NewInsert().Model(dispute).Exec(ctx)
		require.NoError(t, err)
		_, err = db.NewUpdate().
			TableExpr("settlement_disputes").
			Set("created_at = ?", created).
			Where("id = ?", dispute.ID).
			Exec(ctx)
		require.NoError(t, err)
		disputes = append(disputes, dispute)
	}

	conn := postgres.NewTestConnection(db)
	params := driverpayrepository.Params{DB: conn, Logger: zap.NewNop()}
	stranger := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	advances := driverpayrepository.NewPayAdvance(params)
	gotAdvances, err := advances.GetByIDs(ctx, driverpayports.GetPayAdvancesByIDsRequest{
		TenantInfo: tenant,
		IDs:        []pulid.ID{advance.ID, pulid.MustNew("padv_")},
	})
	require.NoError(t, err)
	require.Len(t, gotAdvances, 1)
	assert.Equal(t, int64(30000), gotAdvances[0].OutstandingMinor())
	foreign, err := advances.GetByIDs(ctx, driverpayports.GetPayAdvancesByIDsRequest{
		TenantInfo: stranger,
		IDs:        []pulid.ID{advance.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, foreign)

	accounts := driverpayrepository.NewEscrowAccount(params)
	gotAccounts, err := accounts.GetByIDs(ctx, driverpayports.GetEscrowAccountsByIDsRequest{
		TenantInfo: tenant,
		IDs:        []pulid.ID{escrow.ID},
	})
	require.NoError(t, err)
	require.Len(t, gotAccounts, 1)
	assert.Equal(t, int64(190000), gotAccounts[0].BalanceMinor)

	disputeRepo := driversettlementrepository.NewDispute(driversettlementrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	grouped, err := disputeRepo.ListBySettlementIDs(
		ctx,
		&driversettlementports.ListSettlementDisputesBySettlementIDsRequest{
			TenantInfo:    tenant,
			SettlementIDs: []pulid.ID{disputed.ID, quiet.ID},
		},
	)
	require.NoError(t, err)
	require.Len(t, grouped[disputed.ID], 2)
	assert.Equal(t, disputes[1].ID, grouped[disputed.ID][0].ID, "newest first")
	assert.Equal(t, disputes[0].ID, grouped[disputed.ID][1].ID)
	assert.Empty(t, grouped[quiet.ID])

	empty, err := disputeRepo.ListBySettlementIDs(
		ctx,
		&driversettlementports.ListSettlementDisputesBySettlementIDsRequest{TenantInfo: tenant},
	)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
