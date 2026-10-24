//go:build integration

package journalentryrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type journalBatchRecord struct {
	bun.BaseModel  `bun:"table:journal_batches"`
	ID             pulid.ID `bun:"id,pk"`
	OrganizationID pulid.ID `bun:"organization_id,pk"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk"`
	BatchNumber    string   `bun:"batch_number"`
	BatchType      string   `bun:"batch_type"`
	Status         string   `bun:"status"`
	Description    string   `bun:"description"`
	AccountingDate int64    `bun:"accounting_date"`
	FiscalYearID   pulid.ID `bun:"fiscal_year_id"`
	FiscalPeriodID pulid.ID `bun:"fiscal_period_id"`
	CreatedByID    pulid.ID `bun:"created_by_id"`
}

type journalEntryRecord struct {
	bun.BaseModel  `bun:"table:journal_entries"`
	ID             pulid.ID `bun:"id,pk"`
	OrganizationID pulid.ID `bun:"organization_id,pk"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk"`
	BatchID        pulid.ID `bun:"batch_id"`
	FiscalYearID   pulid.ID `bun:"fiscal_year_id"`
	FiscalPeriodID pulid.ID `bun:"fiscal_period_id"`
	EntryNumber    string   `bun:"entry_number"`
	EntryDate      int64    `bun:"entry_date"`
	EntryType      string   `bun:"entry_type"`
	AccountingDate int64    `bun:"accounting_date"`
	Status         string   `bun:"status"`
	ReferenceType  string   `bun:"reference_type"`
	ReferenceID    string   `bun:"reference_id"`
	Description    string   `bun:"description"`
	TotalDebit     int64    `bun:"total_debit"`
	TotalCredit    int64    `bun:"total_credit"`
	CreatedByID    pulid.ID `bun:"created_by_id"`
	IsPosted       bool     `bun:"is_posted,nullzero"`
	IsReversal     bool     `bun:"is_reversal,nullzero"`
	ReversalOfID   pulid.ID `bun:"reversal_of_id,nullzero"`
	ReversedByID   pulid.ID `bun:"reversed_by_id,nullzero"`
}

func TestListJournalEntries(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})

	var org struct{ ID, BusinessUnitID pulid.ID }
	require.NoError(
		t,
		db.NewSelect().
			Table("organizations").
			Column("id", "business_unit_id").
			Limit(1).
			Scan(ctx, &org),
	)
	var user struct{ ID pulid.ID }
	require.NoError(
		t,
		db.NewSelect().
			Table("users").
			Column("id").
			Where("current_organization_id = ?", org.ID).
			Where("business_unit_id = ?", org.BusinessUnitID).
			Limit(1).
			Scan(ctx, &user),
	)
	entryTime := time.Date(2021, time.March, 15, 12, 0, 0, 0, time.UTC)
	fy, fp := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             entryTime.Unix(),
	})

	batchID := pulid.MustNew("jb_")
	entryAt := entryTime.Unix()
	_, err = db.NewInsert().
		Model(&journalBatchRecord{ID: batchID, OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchNumber: "JB-LIST-1", BatchType: "System", Status: "Posted", Description: "Batch", AccountingDate: entryAt, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, CreatedByID: user.ID}).
		Exec(ctx)
	require.NoError(t, err)
	entryOneID := pulid.MustNew("je_")
	entryTwoID := pulid.MustNew("je_")
	_, err = db.NewInsert().
		Model(&journalEntryRecord{ID: entryOneID, OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchID: batchID, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, EntryNumber: "JE-100", EntryDate: entryAt, EntryType: "Standard", AccountingDate: entryAt, Status: "Posted", ReferenceType: "InvoicePosted", ReferenceID: pulid.MustNew("inv_").String(), Description: "Invoice posting", TotalDebit: 100, TotalCredit: 100, CreatedByID: user.ID}).
		Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().
		Model(&journalEntryRecord{ID: entryTwoID, OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchID: batchID, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, EntryNumber: "JE-200", EntryDate: entryAt + 10, EntryType: "Standard", AccountingDate: entryAt + 10, Status: "Pending", ReferenceType: "ManualJournalPosted", ReferenceID: pulid.MustNew("mjr_").String(), Description: "Manual journal", TotalDebit: 200, TotalCredit: 200, CreatedByID: user.ID}).
		Exec(ctx)
	require.NoError(t, err)

	result, err := repo.List(
		ctx,
		&journalentryports.ListJournalEntriesRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
				Pagination: pagination.Info{Limit: 10},
			},
			FiscalPeriodID: fp.ID,
			Status:         "Posted",
			ReferenceType:  "InvoicePosted",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 1)
	assert.Equal(t, entryOneID, result.Items[0].ID)
	assert.Equal(t, 1, result.Total)
}

func TestReversibleSelectOptionsListOnlyPostedEntriesNotYetReversedNewestFirst(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})

	var org struct{ ID, BusinessUnitID pulid.ID }
	require.NoError(
		t,
		db.NewSelect().
			Table("organizations").
			Column("id", "business_unit_id").
			Limit(1).
			Scan(ctx, &org),
	)
	var user struct{ ID pulid.ID }
	require.NoError(
		t,
		db.NewSelect().
			Table("users").
			Column("id").
			Where("current_organization_id = ?", org.ID).
			Where("business_unit_id = ?", org.BusinessUnitID).
			Limit(1).
			Scan(ctx, &user),
	)
	entryTime := time.Date(2021, time.April, 15, 12, 0, 0, 0, time.UTC)
	fy, fp := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		At:             entryTime.Unix(),
	})
	entryAt := entryTime.Unix()

	batchID := pulid.MustNew("jb_")
	_, err = db.NewInsert().
		Model(&journalBatchRecord{ID: batchID, OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchNumber: "JB-REV-1", BatchType: "System", Status: "Posted", Description: "Batch", AccountingDate: entryAt, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, CreatedByID: user.ID}).
		Exec(ctx)
	require.NoError(t, err)

	entry := func(number, status string, at int64) *journalEntryRecord {
		return &journalEntryRecord{ID: pulid.MustNew("je_"), OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchID: batchID, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, EntryNumber: number, EntryDate: at, EntryType: "Standard", AccountingDate: at, Status: status, ReferenceType: "ManualJournalPosted", ReferenceID: pulid.MustNew("mjr_").String(), Description: "Manual journal", TotalDebit: 100, TotalCredit: 100, CreatedByID: user.ID}
	}

	older := entry("JE-REV-510", "Posted", entryAt)
	older.IsPosted = true
	newer := entry("JE-REV-520", "Posted", entryAt+86400)
	newer.IsPosted = true
	pending := entry("JE-REV-530", "Pending", entryAt+2*86400)
	reversed := entry("JE-REV-540", "Reversed", entryAt+3*86400)
	reversed.IsPosted = true
	reversal := entry("JE-REV-550", "Posted", entryAt+4*86400)
	reversal.IsPosted = true
	reversal.IsReversal = true
	reversal.ReversalOfID = reversed.ID

	for _, record := range []*journalEntryRecord{older, newer, pending} {
		_, err = db.NewInsert().Model(record).Exec(ctx)
		require.NoError(t, err)
	}
	_, err = db.NewInsert().Model(reversed).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(reversal).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewUpdate().
		Table("journal_entries").
		Set("reversed_by_id = ?", reversal.ID).
		Where("id = ?", reversed.ID).
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Exec(ctx)
	require.NoError(t, err)

	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	numbers := func(req *journalentryports.JournalEntrySelectOptionsRequest) []string {
		result, listErr := repo.SelectOptions(ctx, req)
		require.NoError(t, listErr)
		got := make([]string, 0, len(result.Items))
		for _, item := range result.Items {
			got = append(got, item.EntryNumber)
		}
		return got
	}

	assert.Equal(
		t,
		[]string{"JE-REV-520", "JE-REV-510"},
		numbers(&journalentryports.JournalEntrySelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenant,
				Pagination: pagination.Info{Limit: 10},
				Query:      "je-rev-5",
			},
			ReversibleOnly: true,
		}),
	)
	assert.Equal(
		t,
		[]string{"JE-REV-510"},
		numbers(&journalentryports.JournalEntrySelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenant,
				Pagination: pagination.Info{Limit: 10},
				Query:      "510",
			},
			ReversibleOnly: true,
		}),
	)
	assert.Equal(
		t,
		[]string{"JE-REV-550", "JE-REV-540", "JE-REV-530", "JE-REV-520", "JE-REV-510"},
		numbers(&journalentryports.JournalEntrySelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenant,
				Pagination: pagination.Info{Limit: 10},
				Query:      "je-rev-5",
			},
		}),
	)
}
