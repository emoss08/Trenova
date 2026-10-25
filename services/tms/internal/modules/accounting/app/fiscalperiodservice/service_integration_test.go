//go:build integration

package fiscalperiodservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalyearports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	inttestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
		l:              zap.NewNop(),
		db:             conn,
		repo:           fpRepo,
		fiscalYearRepo: fyRepo,
		auditService:   &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
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
		ctx,
		conn,
		kernelports.TxOptions{},
		func(lockCtx context.Context) error {
			_, err := fpRepo.GetByIDForUpdate(lockCtx, fiscalperiodports.GetFiscalPeriodByIDRequest{
				ID:         period.ID,
				TenantInfo: tenantInfo,
			})
			return err
		},
	)
	lock.WaitLocked(t)

	closed, err := svc.Close(ctx, fiscalperiodports.CloseFiscalPeriodRequest{
		ID:         period.ID,
		TenantInfo: tenantInfo,
	}, data.User.ID)

	require.Nil(t, closed)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.Contains(t, err.Error(), "fiscal period is busy")

	lock.Release()
	lock.Wait(t)

	periodAfter, err := fpRepo.GetByID(ctx, fiscalperiodports.GetFiscalPeriodByIDRequest{
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
	fpRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	validator := &Validator{db: conn}
	svc := &Service{
		l:              zap.NewNop(),
		db:             conn,
		repo:           fpRepo,
		fiscalYearRepo: fyRepo,
		validator:      validator,
		auditService:   &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
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

	insertApprovedManualJournal(t, ctx, db, data, period, "MJR-CL-1")

	closed, err := svc.Close(
		ctx,
		fiscalperiodports.CloseFiscalPeriodRequest{
			ID: period.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: data.Organization.ID,
				BuID:  data.BusinessUnit.ID,
			},
		},
		data.User.ID,
	)
	require.Nil(t, closed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manual journal requests")
}

func TestCloseBlockedByPendingJournalSource(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	validator := &Validator{db: conn}
	svc := &Service{
		l:              zap.NewNop(),
		db:             conn,
		repo:           fpRepo,
		fiscalYearRepo: fyRepo,
		validator:      validator,
		auditService:   &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
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

	insertPendingJournalSource(t, ctx, db, data, period)

	closed, err := svc.Close(
		ctx,
		fiscalperiodports.CloseFiscalPeriodRequest{
			ID: period.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: data.Organization.ID,
				BuID:  data.BusinessUnit.ID,
			},
		},
		data.User.ID,
	)
	require.Nil(t, closed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accounting sources")
}

func TestGetCloseBlockersReturnsManualJournalBlocker(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	validator := &Validator{db: conn}
	svc := &Service{
		l:              zap.NewNop(),
		db:             conn,
		repo:           fpRepo,
		fiscalYearRepo: fyRepo,
		validator:      validator,
		auditService:   &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	fy := mustCreateIntegrationFiscalYear(
		t,
		ctx,
		fyRepo,
		&fiscalyear.FiscalYear{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			Status:         fiscalyear.StatusOpen,
			Year:           2026,
			Name:           "FY 2026",
			StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
			EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
			IsCurrent:      true,
		},
	)
	period := mustCreateFiscalPeriod(
		t,
		ctx,
		fpRepo,
		&fiscalperiod.FiscalPeriod{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			FiscalYearID:   fy.ID,
			PeriodNumber:   1,
			PeriodType:     fiscalperiod.PeriodTypeMonth,
			Status:         fiscalperiod.StatusOpen,
			Name:           "Period 1 - January 2026",
			StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
			EndDate:        time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
		},
	)

	insertApprovedManualJournal(t, ctx, db, data, period, "MJR-CL-2")

	result, err := svc.GetCloseBlockers(
		ctx,
		fiscalperiodports.GetFiscalPeriodByIDRequest{
			ID: period.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: data.Organization.ID,
				BuID:  data.BusinessUnit.ID,
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.CanClose)
	require.NotEmpty(t, result.Blockers)
	assert.Contains(t, result.Blockers[0].Message, "manual journal requests")
}

func TestLockSucceedsFromOpenStatus(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	svc := &Service{
		l:              zap.NewNop(),
		db:             conn,
		repo:           fpRepo,
		fiscalYearRepo: fyRepo,
		auditService:   &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
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

	locked, err := svc.Lock(
		ctx,
		&fiscalperiodports.LockFiscalPeriodRequest{
			ID: period.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: data.Organization.ID,
				BuID:  data.BusinessUnit.ID,
			},
		},
		data.User.ID,
	)
	require.NoError(t, err)
	require.NotNil(t, locked)
	assert.Equal(t, fiscalperiod.StatusLocked, locked.Status)
}

func TestCloseBlockedWhenPeriodCloseModeIsSystemScheduled(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	validator := &Validator{db: conn, accountingRepo: accountingRepo}
	svc := &Service{
		l:              zap.NewNop(),
		db:             conn,
		repo:           fpRepo,
		fiscalYearRepo: fyRepo,
		validator:      validator,
		auditService:   &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, data.Organization.ID).
		Return(&tenant.AccountingControl{PeriodCloseMode: tenant.PeriodCloseModeSystemScheduled, RequirePeriodCloseApproval: false}, nil)

	fy := mustCreateIntegrationFiscalYear(
		t,
		ctx,
		fyRepo,
		&fiscalyear.FiscalYear{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			Status:         fiscalyear.StatusOpen,
			Year:           2026,
			Name:           "FY 2026",
			StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
			EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
			IsCurrent:      true,
		},
	)
	period := mustCreateFiscalPeriod(
		t,
		ctx,
		fpRepo,
		&fiscalperiod.FiscalPeriod{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			FiscalYearID:   fy.ID,
			PeriodNumber:   1,
			PeriodType:     fiscalperiod.PeriodTypeMonth,
			Status:         fiscalperiod.StatusOpen,
			Name:           "Period 1 - January 2026",
			StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
			EndDate:        time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
		},
	)

	closed, err := svc.Close(
		ctx,
		fiscalperiodports.CloseFiscalPeriodRequest{
			ID: period.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: data.Organization.ID,
				BuID:  data.BusinessUnit.ID,
			},
		},
		data.User.ID,
	)
	require.Nil(t, closed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "system-scheduled close")
}

func TestCloseBlockedWhenPeriodCloseApprovalRequired(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()})
	fpRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	validator := &Validator{db: conn, accountingRepo: accountingRepo}
	svc := &Service{
		l:              zap.NewNop(),
		db:             conn,
		repo:           fpRepo,
		fiscalYearRepo: fyRepo,
		validator:      validator,
		auditService:   &mocks.NoopAuditService{},
	}

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, data.Organization.ID).
		Return(&tenant.AccountingControl{PeriodCloseMode: tenant.PeriodCloseModeManualOnly, RequirePeriodCloseApproval: true}, nil)

	fy := mustCreateIntegrationFiscalYear(
		t,
		ctx,
		fyRepo,
		&fiscalyear.FiscalYear{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			Status:         fiscalyear.StatusOpen,
			Year:           2026,
			Name:           "FY 2026",
			StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
			EndDate:        time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
			IsCurrent:      true,
		},
	)
	period := mustCreateFiscalPeriod(
		t,
		ctx,
		fpRepo,
		&fiscalperiod.FiscalPeriod{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			FiscalYearID:   fy.ID,
			PeriodNumber:   1,
			PeriodType:     fiscalperiod.PeriodTypeMonth,
			Status:         fiscalperiod.StatusOpen,
			Name:           "Period 1 - January 2026",
			StartDate:      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
			EndDate:        time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
		},
	)

	closed, err := svc.Close(
		ctx,
		fiscalperiodports.CloseFiscalPeriodRequest{
			ID: period.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: data.Organization.ID,
				BuID:  data.BusinessUnit.ID,
			},
		},
		data.User.ID,
	)
	require.Nil(t, closed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approval is required")
}

func insertApprovedManualJournal(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	data *seedtest.TestData,
	period *fiscalperiod.FiscalPeriod,
	number string,
) {
	t.Helper()

	require.NoError(t, manualjournaldb.Requests.Create(ctx, db, &manualjournal.Request{
		OrganizationID:          data.Organization.ID,
		BusinessUnitID:          data.BusinessUnit.ID,
		RequestNumber:           number,
		Status:                  manualjournal.StatusApproved,
		Description:             "Close blocker",
		AccountingDate:          period.StartDate,
		RequestedFiscalYearID:   period.FiscalYearID,
		RequestedFiscalPeriodID: period.ID,
		CurrencyCode:            "USD",
		TotalDebit:              100,
		TotalCredit:             100,
		CreatedByID:             data.User.ID,
	}))
}

func insertPendingJournalSource(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	data *seedtest.TestData,
	period *fiscalperiod.FiscalPeriod,
) {
	t.Helper()

	batch := &journalentry.JournalBatch{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		BatchNumber:    "JB-PEND-1",
		BatchType:      "System",
		Status:         "Pending",
		Description:    "Pending source",
		AccountingDate: period.StartDate,
		FiscalYearID:   period.FiscalYearID,
		FiscalPeriodID: period.ID,
		CreatedByID:    data.User.ID,
	}
	require.NoError(t, journalentrydb.JournalBatches.Create(ctx, db, batch))

	entry := &journalentry.JournalEntry{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		BatchID:        batch.ID,
		FiscalYearID:   period.FiscalYearID,
		FiscalPeriodID: period.ID,
		EntryNumber:    "JE-PEND-1",
		EntryDate:      period.StartDate,
		EntryType:      journalentry.EntryTypeStandard,
		AccountingDate: period.StartDate,
		Status:         journalentry.StatusPending,
		ReferenceType:  "InvoicePosted",
		ReferenceID:    pulid.MustNew("inv_").String(),
		Description:    "Pending source",
		TotalDebit:     100,
		TotalCredit:    100,
		CreatedByID:    data.User.ID,
	}
	require.NoError(t, journalentrydb.JournalEntries.Create(ctx, db, entry))

	require.NoError(t, journalsourcedb.Sources.Create(ctx, db, &journalsource.Source{
		OrganizationID:   data.Organization.ID,
		BusinessUnitID:   data.BusinessUnit.ID,
		SourceObjectType: "Invoice",
		SourceObjectID:   pulid.MustNew("inv_").String(),
		SourceEventType:  "InvoicePosted",
		Status:           "Pending",
		JournalBatchID:   batch.ID,
		JournalEntryID:   entry.ID,
	}))
}

func mustCreateIntegrationFiscalYear(
	t *testing.T,
	ctx context.Context,
	repo fiscalyearports.FiscalYearRepository,
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
	repo fiscalperiodports.FiscalPeriodRepository,
	entity *fiscalperiod.FiscalPeriod,
) *fiscalperiod.FiscalPeriod {
	t.Helper()

	created, err := repo.Create(ctx, entity)
	require.NoError(t, err)
	return created
}
