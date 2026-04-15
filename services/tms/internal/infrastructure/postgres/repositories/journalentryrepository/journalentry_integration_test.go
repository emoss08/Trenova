//go:build integration

package journalentryrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
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
}

func TestListJournalEntries(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()})

	var org struct{ ID, BusinessUnitID pulid.ID }
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user struct{ ID pulid.ID }
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	nowTime := time.Now().UTC()
	fy, err := fyRepo.Create(ctx, &fiscalyear.FiscalYear{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: fiscalyear.StatusOpen, Year: nowTime.Year(), Name: "FY", StartDate: nowTime.Add(-24 * time.Hour).Unix(), EndDate: nowTime.Add(24 * time.Hour).Unix(), IsCurrent: true})
	require.NoError(t, err)
	fp, err := fpRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "Period 1", StartDate: nowTime.Add(-24 * time.Hour).Unix(), EndDate: nowTime.Add(24 * time.Hour).Unix()})
	require.NoError(t, err)

	batchID := pulid.MustNew("jb_")
	now := nowTime.Unix()
	_, err = db.NewInsert().Model(&journalBatchRecord{ID: batchID, OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchNumber: "JB-LIST-1", BatchType: "System", Status: "Posted", Description: "Batch", AccountingDate: now, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, CreatedByID: user.ID}).Exec(ctx)
	require.NoError(t, err)
	entryOneID := pulid.MustNew("je_")
	entryTwoID := pulid.MustNew("je_")
	_, err = db.NewInsert().Model(&journalEntryRecord{ID: entryOneID, OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchID: batchID, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, EntryNumber: "JE-100", EntryDate: now, EntryType: "Standard", AccountingDate: now, Status: "Posted", ReferenceType: "InvoicePosted", ReferenceID: pulid.MustNew("inv_").String(), Description: "Invoice posting", TotalDebit: 100, TotalCredit: 100, CreatedByID: user.ID}).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&journalEntryRecord{ID: entryTwoID, OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchID: batchID, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, EntryNumber: "JE-200", EntryDate: now + 10, EntryType: "Standard", AccountingDate: now + 10, Status: "Pending", ReferenceType: "ManualJournalPosted", ReferenceID: pulid.MustNew("mjr_").String(), Description: "Manual journal", TotalDebit: 200, TotalCredit: 200, CreatedByID: user.ID}).Exec(ctx)
	require.NoError(t, err)

	result, err := repo.List(ctx, &repositories.ListJournalEntriesRequest{Filter: &pagination.QueryOptions{TenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}, Pagination: pagination.Info{Limit: 10}}, FiscalPeriodID: fp.ID, Status: "Posted", ReferenceType: "InvoicePosted"})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 1)
	assert.Equal(t, entryOneID, result.Items[0].ID)
	assert.Equal(t, 1, result.Total)
}
