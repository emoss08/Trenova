//go:build integration

package fiscalperiodservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
	inttestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestCloseReturnsConflictWhenFiscalPeriodIsLocked(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	fpRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	svc := &Service{
		l:            zap.NewNop(),
		db:           conn,
		repo:         fpRepo,
		auditService: &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	fy := mustCreateIntegrationFiscalYear(t, ctx, fyRepo, &fiscalyear.FiscalYear{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         fiscalyear.StatusOpen,
		Year:           2026,
		Name:           "FY 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:      true,
	})
	period := mustCreateFiscalPeriod(t, ctx, fpRepo, &fiscalperiod.FiscalPeriod{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		FiscalYearID:   fy.ID,
		PeriodNumber:   1,
		PeriodType:     fiscalperiod.PeriodTypeMonth,
		Status:         fiscalperiod.StatusOpen,
		Name:           "Period 1 - January 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
	})
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	lock := inttestutil.HoldTxLock(
		t,
		conn,
		ports.TxOptions{},
		func(lockCtx context.Context, _ bun.Tx) error {
			_, err := fpRepo.GetByIDForUpdate(lockCtx, repositories.GetFiscalPeriodByIDRequest{
				ID:         period.ID,
				TenantInfo: tenantInfo,
			})
			return err
		},
	)
	lock.WaitLocked(t)

	closed, err := svc.Close(ctx, repositories.CloseFiscalPeriodRequest{
		ID:         period.ID,
		TenantInfo: tenantInfo,
	}, data.User.ID)

	require.Nil(t, closed)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.Contains(t, err.Error(), "fiscal period is busy")

	lock.Release()
	lock.Wait(t)

	periodAfter, err := fpRepo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
		ID:         period.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)
	assert.Equal(t, fiscalperiod.StatusOpen, periodAfter.Status)
	assert.Nil(t, periodAfter.ClosedAt)
	assert.True(t, periodAfter.ClosedByID.IsNil())
}

func TestCloseBlockedByApprovedManualJournalRequest(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()})
	validator := &Validator{db: conn}
	svc := &Service{l: zap.NewNop(), db: conn, repo: fpRepo, validator: validator, auditService: &mocks.NoopAuditService{}}

	data := seedtest.SeedFullTestData(t, ctx, db)
	fy := mustCreateIntegrationFiscalYear(t, ctx, fyRepo, &fiscalyear.FiscalYear{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         fiscalyear.StatusOpen,
		Year:           2026,
		Name:           "FY 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:      true,
	})
	period := mustCreateFiscalPeriod(t, ctx, fpRepo, &fiscalperiod.FiscalPeriod{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		FiscalYearID:   fy.ID,
		PeriodNumber:   1,
		PeriodType:     fiscalperiod.PeriodTypeMonth,
		Status:         fiscalperiod.StatusOpen,
		Name:           "Period 1 - January 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
	})

	_, err := db.NewInsert().Model(&manualJournalRequestRecord{
		ID:                      pulid.MustNew("mjr_"),
		OrganizationID:          data.Organization.ID,
		BusinessUnitID:          data.BusinessUnit.ID,
		RequestNumber:           "MJR-CL-1",
		Status:                  "Approved",
		Description:             "Close blocker",
		AccountingDate:          period.StartDate,
		RequestedFiscalYearID:   fy.ID,
		RequestedFiscalPeriodID: period.ID,
		CurrencyCode:            "USD",
		TotalDebitMinor:         100,
		TotalCreditMinor:        100,
		CreatedByID:             data.User.ID,
	}).Exec(ctx)
	require.NoError(t, err)

	closed, err := svc.Close(ctx, repositories.CloseFiscalPeriodRequest{ID: period.ID, TenantInfo: pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}}, data.User.ID)
	require.Nil(t, closed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manual journal requests")
}

func TestCloseBlockedByPendingJournalSource(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()})
	validator := &Validator{db: conn}
	svc := &Service{l: zap.NewNop(), db: conn, repo: fpRepo, validator: validator, auditService: &mocks.NoopAuditService{}}

	data := seedtest.SeedFullTestData(t, ctx, db)
	fy := mustCreateIntegrationFiscalYear(t, ctx, fyRepo, &fiscalyear.FiscalYear{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         fiscalyear.StatusOpen,
		Year:           2026,
		Name:           "FY 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:      true,
	})
	period := mustCreateFiscalPeriod(t, ctx, fpRepo, &fiscalperiod.FiscalPeriod{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		FiscalYearID:   fy.ID,
		PeriodNumber:   1,
		PeriodType:     fiscalperiod.PeriodTypeMonth,
		Status:         fiscalperiod.StatusOpen,
		Name:           "Period 1 - January 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
	})

	batchID := pulid.MustNew("jb_")
	entryID := pulid.MustNew("je_")
	_, err := db.NewInsert().Model(&journalBatchRecord{
		ID:             batchID,
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		BatchNumber:    "JB-PEND-1",
		BatchType:      "System",
		Status:         "Pending",
		Description:    "Pending source",
		AccountingDate: period.StartDate,
		FiscalYearID:   fy.ID,
		FiscalPeriodID: period.ID,
		CreatedByID:    data.User.ID,
	}).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&journalEntryRecord{
		ID:             entryID,
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		BatchID:        batchID,
		FiscalYearID:   fy.ID,
		FiscalPeriodID: period.ID,
		EntryNumber:    "JE-PEND-1",
		EntryDate:      period.StartDate,
		EntryType:      "Standard",
		AccountingDate: period.StartDate,
		Status:         "Pending",
		ReferenceType:  "InvoicePosted",
		ReferenceID:    pulid.MustNew("inv_").String(),
		Description:    "Pending source",
		TotalDebit:     100,
		TotalCredit:    100,
		CreatedByID:    data.User.ID,
	}).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&journalSourceRecord{
		ID:               pulid.MustNew("jsrc_"),
		OrganizationID:   data.Organization.ID,
		BusinessUnitID:   data.BusinessUnit.ID,
		SourceObjectType: "Invoice",
		SourceObjectID:   pulid.MustNew("inv_").String(),
		SourceEventType:  "InvoicePosted",
		Status:           "Pending",
		JournalBatchID:   batchID,
		JournalEntryID:   entryID,
	}).Exec(ctx)
	require.NoError(t, err)

	closed, err := svc.Close(ctx, repositories.CloseFiscalPeriodRequest{ID: period.ID, TenantInfo: pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}}, data.User.ID)
	require.Nil(t, closed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accounting sources")
}

func TestLockSucceedsFromOpenStatus(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()})
	svc := &Service{l: zap.NewNop(), db: conn, repo: fpRepo, auditService: &mocks.NoopAuditService{}}

	data := seedtest.SeedFullTestData(t, ctx, db)
	fy := mustCreateIntegrationFiscalYear(t, ctx, fyRepo, &fiscalyear.FiscalYear{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         fiscalyear.StatusOpen,
		Year:           2026,
		Name:           "FY 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:      true,
	})
	period := mustCreateFiscalPeriod(t, ctx, fpRepo, &fiscalperiod.FiscalPeriod{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		FiscalYearID:   fy.ID,
		PeriodNumber:   1,
		PeriodType:     fiscalperiod.PeriodTypeMonth,
		Status:         fiscalperiod.StatusOpen,
		Name:           "Period 1 - January 2026",
		StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
	})

	locked, err := svc.Lock(ctx, repositories.LockFiscalPeriodRequest{ID: period.ID, TenantInfo: pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}}, data.User.ID)
	require.NoError(t, err)
	require.NotNil(t, locked)
	assert.Equal(t, fiscalperiod.StatusLocked, locked.Status)
}

type manualJournalRequestRecord struct {
	bun.BaseModel `bun:"table:manual_journal_requests"`

	ID                      pulid.ID `bun:"id,pk"`
	OrganizationID          pulid.ID `bun:"organization_id,pk"`
	BusinessUnitID          pulid.ID `bun:"business_unit_id,pk"`
	RequestNumber           string   `bun:"request_number"`
	Status                  string   `bun:"status"`
	Description             string   `bun:"description"`
	AccountingDate          int64    `bun:"accounting_date"`
	RequestedFiscalYearID   pulid.ID `bun:"requested_fiscal_year_id"`
	RequestedFiscalPeriodID pulid.ID `bun:"requested_fiscal_period_id"`
	CurrencyCode            string   `bun:"currency_code"`
	TotalDebitMinor         int64    `bun:"total_debit_minor"`
	TotalCreditMinor        int64    `bun:"total_credit_minor"`
	CreatedByID             pulid.ID `bun:"created_by_id"`
}

type journalBatchRecord struct {
	bun.BaseModel `bun:"table:journal_batches"`

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
	bun.BaseModel `bun:"table:journal_entries"`

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

type journalSourceRecord struct {
	bun.BaseModel `bun:"table:journal_sources"`

	ID               pulid.ID `bun:"id,pk"`
	OrganizationID   pulid.ID `bun:"organization_id,pk"`
	BusinessUnitID   pulid.ID `bun:"business_unit_id,pk"`
	SourceObjectType string   `bun:"source_object_type"`
	SourceObjectID   string   `bun:"source_object_id"`
	SourceEventType  string   `bun:"source_event_type"`
	Status           string   `bun:"status"`
	JournalBatchID   pulid.ID `bun:"journal_batch_id"`
	JournalEntryID   pulid.ID `bun:"journal_entry_id"`
}

func mustCreateIntegrationFiscalYear(
	t *testing.T,
	ctx context.Context,
	repo repositories.FiscalYearRepository,
	entity *fiscalyear.FiscalYear,
) *fiscalyear.FiscalYear {
	t.Helper()

	created, err := repo.Create(ctx, entity)
	require.NoError(t, err)
	return created
}

func mustCreateFiscalPeriod(
	t *testing.T,
	ctx context.Context,
	repo repositories.FiscalPeriodRepository,
	entity *fiscalperiod.FiscalPeriod,
) *fiscalperiod.FiscalPeriod {
	t.Helper()

	created, err := repo.Create(ctx, entity)
	require.NoError(t, err)
	return created
}
