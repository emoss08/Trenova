package manualjournalservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestRequireManualJournalUser(t *testing.T) {
	t.Parallel()

	_, err := requireManualJournalUser(&serviceports.RequestActor{})
	require.Error(t, err)

	userID := pulid.MustNew("usr_")
	resolved, err := requireManualJournalUser(&serviceports.RequestActor{UserID: userID})
	require.NoError(t, err)
	require.Equal(t, userID, resolved)
}

func TestGetReturnsRequest(t *testing.T) {
	t.Parallel()

	request := approvedRequest(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_"))
	svc := &Service{repo: &fakeManualJournalRepository{entity: request}}

	result, err := svc.Get(t.Context(), &serviceports.GetManualJournalRequest{RequestID: request.ID, TenantInfo: pagination.TenantInfo{OrgID: request.OrganizationID, BuID: request.BusinessUnitID}})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, request.ID, result.ID)
}

func TestListReturnsRequests(t *testing.T) {
	t.Parallel()

	request := approvedRequest(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_"))
	svc := &Service{repo: &fakeManualJournalRepository{entity: request}}

	result, err := svc.List(t.Context(), &repositories.ListManualJournalRequest{Filter: &pagination.QueryOptions{TenantInfo: pagination.TenantInfo{OrgID: request.OrganizationID, BuID: request.BusinessUnitID}}})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, request.ID, result.Items[0].ID)
}

func TestCreateDraftPersistsGeneratedNumber(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	accountID1 := pulid.MustNew("gla_")
	accountID2 := pulid.MustNew("gla_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		CurrencyMode:             tenant.CurrencyModeSingleCurrency,
		FunctionalCurrencyCode:   "USD",
		ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAllowAll,
	}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 1_700_000_000}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID, PeriodType: fiscalperiod.PeriodTypeMonth}, nil)
	glRepo := mocks.NewMockGLAccountRepository(t)
	glRepo.EXPECT().GetByIDs(mock.Anything, repositories.GetGLAccountsByIDsRequest{TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID}, GLAccountIDs: []pulid.ID{accountID1, accountID2}}).Return([]*glaccount.GLAccount{{ID: accountID1, Status: domaintypes.StatusActive, AllowManualJE: true}, {ID: accountID2, Status: domaintypes.StatusActive, AllowManualJE: true}}, nil)

	manualRepo := &fakeManualJournalRepository{}
	svc := &Service{
		db:             fakeDBConnection{},
		repo:           manualRepo,
		accountingRepo: accountingRepo,
		generator:      testutil.TestSequenceGenerator{SingleValue: "MJR-42"},
		validator:      &Validator{fiscalRepo: fiscalRepo, glAccountRepo: glRepo},
		auditService:   &mocks.NoopAuditService{},
	}

	result, err := svc.CreateDraft(t.Context(), &serviceports.CreateManualJournalRequest{
		Description:    "Accrual",
		AccountingDate: 1_700_000_000,
		CurrencyCode:   "USD",
		Lines: []*serviceports.ManualJournalLineInput{
			{GLAccountID: accountID1, Description: "Debit", DebitAmount: 100},
			{GLAccountID: accountID2, Description: "Credit", CreditAmount: 100},
		},
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "MJR-42", result.RequestNumber)
	assert.Equal(t, manualjournal.StatusDraft, result.Status)
	assert.Equal(t, int64(100), result.TotalDebit)
	assert.Equal(t, int64(100), result.TotalCredit)
}

func TestUpdateDraftBlocksNonDraftRequest(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusApproved

	svc := &Service{repo: &fakeManualJournalRepository{entity: request}}
	result, err := svc.UpdateDraft(t.Context(), &serviceports.UpdateManualJournalDraftRequest{RequestID: request.ID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only draft")
}

func TestUpdateDraftReplacesLines(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusDraft
	accountID1 := pulid.MustNew("gla_")
	accountID2 := pulid.MustNew("gla_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAllowAll}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 1_700_000_200}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID, PeriodType: fiscalperiod.PeriodTypeMonth}, nil)
	glRepo := mocks.NewMockGLAccountRepository(t)
	glRepo.EXPECT().GetByIDs(mock.Anything, repositories.GetGLAccountsByIDsRequest{TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID}, GLAccountIDs: []pulid.ID{accountID1, accountID2}}).Return([]*glaccount.GLAccount{{ID: accountID1, Status: domaintypes.StatusActive, AllowManualJE: true}, {ID: accountID2, Status: domaintypes.StatusActive, AllowManualJE: true}}, nil)

	manualRepo := &fakeManualJournalRepository{entity: request}
	svc := &Service{
		db:             fakeDBConnection{},
		repo:           manualRepo,
		accountingRepo: accountingRepo,
		validator:      &Validator{fiscalRepo: fiscalRepo, glAccountRepo: glRepo},
		auditService:   &mocks.NoopAuditService{},
	}

	result, err := svc.UpdateDraft(t.Context(), &serviceports.UpdateManualJournalDraftRequest{
		RequestID:      request.ID,
		Description:    "Updated accrual",
		AccountingDate: 1_700_000_200,
		CurrencyCode:   "USD",
		Lines: []*serviceports.ManualJournalLineInput{
			{GLAccountID: accountID1, Description: "Updated debit", DebitAmount: 250},
			{GLAccountID: accountID2, Description: "Updated credit", CreditAmount: 250},
		},
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Updated accrual", result.Description)
	require.Len(t, result.Lines, 2)
	assert.Equal(t, "Updated debit", result.Lines[0].Description)
	assert.Equal(t, int64(250), result.TotalDebit)
	assert.Equal(t, int64(250), result.TotalCredit)
}

func TestPostCreatesJournalPostingAndMarksRequestPosted(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	request := approvedRequest(orgID, buID, userID)

	manualRepo := &fakeManualJournalRepository{entity: request}
	journalRepo := &fakeJournalPostingRepository{}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyRequireReopen,
	}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: request.AccountingDate}).Return(&fiscalperiod.FiscalPeriod{
		ID:           periodID,
		FiscalYearID: fyID,
		PeriodType:   fiscalperiod.PeriodTypeMonth,
		Status:       fiscalperiod.StatusOpen,
	}, nil)

	svc := &Service{
		db:             fakeDBConnection{},
		repo:           manualRepo,
		journalRepo:    journalRepo,
		accountingRepo: accountingRepo,
		generator:      testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator:      &Validator{fiscalRepo: fiscalRepo},
		auditService:   &mocks.NoopAuditService{},
	}

	result, err := svc.Post(t.Context(), &serviceports.GetManualJournalRequest{
		RequestID:  request.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, manualjournal.StatusPosted, result.Status)
	assert.True(t, result.PostedBatchID.IsNotNil())
	require.NotNil(t, journalRepo.last)
	assert.Equal(t, request.TotalDebit, journalRepo.last.TotalDebit)
	assert.Equal(t, request.TotalCredit, journalRepo.last.TotalCredit)
	assert.Equal(t, "ManualJournalRequest", journalRepo.last.ReferenceType)
	assert.Equal(t, periodID, journalRepo.last.FiscalPeriodID)
	assert.Equal(t, request.AccountingDate, journalRepo.last.AccountingDate)
	assert.False(t, journalRepo.last.RequiresApproval)
	require.Len(t, journalRepo.last.Lines, 2)
	assert.Equal(t, int64(1000), journalRepo.last.Lines[0].DebitAmount)
	assert.Equal(t, int64(1000), journalRepo.last.Lines[1].CreditAmount)
}

func TestPostUsesNextOpenPeriodWhenConfigured(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	closedPeriodID := pulid.MustNew("fp_")
	nextPeriodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	request := approvedRequest(orgID, buID, userID)

	manualRepo := &fakeManualJournalRepository{entity: request}
	journalRepo := &fakeJournalPostingRepository{}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyPostToNextOpen,
	}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: request.AccountingDate}).Return(&fiscalperiod.FiscalPeriod{
		ID:           closedPeriodID,
		FiscalYearID: fyID,
		PeriodNumber: 1,
		Status:       fiscalperiod.StatusClosed,
	}, nil)
	fiscalRepo.EXPECT().ListByFiscalYearID(mock.Anything, repositories.ListByFiscalYearIDRequest{FiscalYearID: fyID, OrgID: orgID, BuID: buID}).Return([]*fiscalperiod.FiscalPeriod{
		{ID: closedPeriodID, FiscalYearID: fyID, PeriodNumber: 1, Status: fiscalperiod.StatusClosed},
		{ID: nextPeriodID, FiscalYearID: fyID, PeriodNumber: 2, Status: fiscalperiod.StatusOpen, StartDate: 1_700_001_000},
	}, nil)

	svc := &Service{
		db:             fakeDBConnection{},
		repo:           manualRepo,
		journalRepo:    journalRepo,
		accountingRepo: accountingRepo,
		generator:      testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator:      &Validator{fiscalRepo: fiscalRepo},
		auditService:   &mocks.NoopAuditService{},
	}

	_, err := svc.Post(t.Context(), &serviceports.GetManualJournalRequest{
		RequestID:  request.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, journalRepo.last)
	assert.Equal(t, nextPeriodID, journalRepo.last.FiscalPeriodID)
	assert.Equal(t, int64(1_700_001_000), journalRepo.last.AccountingDate)
}

func TestPostBlocksNonApprovedRequest(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusDraft

	svc := &Service{
		db:          fakeDBConnection{},
		repo:        &fakeManualJournalRepository{entity: request},
		journalRepo: &fakeJournalPostingRepository{},
		accountingRepo: func() repositories.AccountingControlRepository {
			repo := mocks.NewMockAccountingControlRepository(t)
			repo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyRequireReopen}, nil)
			return repo
		}(),
		generator:    testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator:    &Validator{},
		auditService: &mocks.NoopAuditService{},
	}

	result, err := svc.Post(t.Context(), &serviceports.GetManualJournalRequest{
		RequestID:  request.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approved")
}

func TestPostBlocksClosedPeriodWhenReopenRequired(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	fyID := pulid.MustNew("fy_")
	request := approvedRequest(orgID, buID, userID)

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyRequireReopen,
	}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: request.AccountingDate}).Return(&fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: fyID,
		PeriodNumber: 1,
		Status:       fiscalperiod.StatusClosed,
	}, nil)

	journalRepo := &fakeJournalPostingRepository{}
	svc := &Service{
		db:             fakeDBConnection{},
		repo:           &fakeManualJournalRepository{entity: request},
		journalRepo:    journalRepo,
		accountingRepo: accountingRepo,
		generator:      testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator:      &Validator{fiscalRepo: fiscalRepo},
		auditService:   &mocks.NoopAuditService{},
	}

	result, err := svc.Post(t.Context(), &serviceports.GetManualJournalRequest{
		RequestID:  request.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reopen")
	assert.Nil(t, journalRepo.last)
}

func TestSubmitMovesToPendingApprovalWhenRequired(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusDraft

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAllowAll,
		RequireManualJEApproval:  true,
	}, nil)

	svc := &Service{
		db:             fakeDBConnection{},
		repo:           &fakeManualJournalRepository{entity: request},
		accountingRepo: accountingRepo,
		validator:      &Validator{},
		auditService:   &mocks.NoopAuditService{},
	}

	result, err := svc.Submit(t.Context(), &serviceports.GetManualJournalRequest{
		RequestID:  request.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, manualjournal.StatusPendingApproval, result.Status)
	assert.True(t, result.ApprovedByID.IsNil())
}

func TestSubmitAutoApprovesWhenApprovalNotRequired(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusDraft
	request.ApprovedAt = nil
	request.ApprovedByID = pulid.Nil

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		ManualJournalEntryPolicy: tenant.ManualJournalEntryPolicyAllowAll,
		RequireManualJEApproval:  false,
	}, nil)

	svc := &Service{
		db:             fakeDBConnection{},
		repo:           &fakeManualJournalRepository{entity: request},
		accountingRepo: accountingRepo,
		validator:      &Validator{},
		auditService:   &mocks.NoopAuditService{},
	}

	result, err := svc.Submit(t.Context(), &serviceports.GetManualJournalRequest{
		RequestID:  request.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, manualjournal.StatusApproved, result.Status)
	assert.Equal(t, userID, result.ApprovedByID)
	assert.NotNil(t, result.ApprovedAt)
}

func TestApproveMarksApproved(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusPendingApproval
	request.ApprovedAt = nil
	request.ApprovedByID = pulid.Nil

	svc := &Service{db: fakeDBConnection{}, repo: &fakeManualJournalRepository{entity: request}, validator: &Validator{}, auditService: &mocks.NoopAuditService{}}
	result, err := svc.Approve(t.Context(), &serviceports.GetManualJournalRequest{RequestID: request.ID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	assert.Equal(t, manualjournal.StatusApproved, result.Status)
	assert.Equal(t, userID, result.ApprovedByID)
	assert.NotNil(t, result.ApprovedAt)
}

func TestApproveBlocksInvalidStatus(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusDraft

	svc := &Service{db: fakeDBConnection{}, repo: &fakeManualJournalRepository{entity: request}, validator: &Validator{}, auditService: &mocks.NoopAuditService{}}
	result, err := svc.Approve(t.Context(), &serviceports.GetManualJournalRequest{RequestID: request.ID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pending")
}

func TestRejectMarksRejected(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusPendingApproval

	svc := &Service{db: fakeDBConnection{}, repo: &fakeManualJournalRepository{entity: request}, validator: &Validator{}, auditService: &mocks.NoopAuditService{}}
	result, err := svc.Reject(t.Context(), &serviceports.RejectManualJournalRequest{RequestID: request.ID, Reason: "Insufficient support", TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	assert.Equal(t, manualjournal.StatusRejected, result.Status)
	assert.Equal(t, "Insufficient support", result.RejectionReason)
	assert.Equal(t, userID, result.RejectedByID)
}

func TestCancelMarksCancelled(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	request := approvedRequest(orgID, buID, userID)
	request.Status = manualjournal.StatusApproved

	svc := &Service{db: fakeDBConnection{}, repo: &fakeManualJournalRepository{entity: request}, validator: &Validator{}, auditService: &mocks.NoopAuditService{}}
	result, err := svc.Cancel(t.Context(), &serviceports.CancelManualJournalRequest{RequestID: request.ID, Reason: "Posted in error", TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	assert.Equal(t, manualjournal.StatusCancelled, result.Status)
	assert.Equal(t, "Posted in error", result.CancelReason)
	assert.Equal(t, userID, result.CancelledByID)
}

func TestPostAllowsLockedPeriod(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	fyID := pulid.MustNew("fy_")
	request := approvedRequest(orgID, buID, userID)

	manualRepo := &fakeManualJournalRepository{entity: request}
	journalRepo := &fakeJournalPostingRepository{}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyRequireReopen}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: request.AccountingDate}).Return(&fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), FiscalYearID: fyID, Status: fiscalperiod.StatusLocked}, nil)

	svc := &Service{db: fakeDBConnection{}, repo: manualRepo, journalRepo: journalRepo, accountingRepo: accountingRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: &Validator{fiscalRepo: fiscalRepo}, auditService: &mocks.NoopAuditService{}}
	_, err := svc.Post(t.Context(), &serviceports.GetManualJournalRequest{RequestID: request.ID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, journalRepo.last)
	assert.Equal(t, request.AccountingDate, journalRepo.last.AccountingDate)
}

func TestPostErrorsWhenNoNextOpenPeriodExists(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	fyID := pulid.MustNew("fy_")
	request := approvedRequest(orgID, buID, userID)

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyPostToNextOpen}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: request.AccountingDate}).Return(&fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), FiscalYearID: fyID, PeriodNumber: 3, Status: fiscalperiod.StatusClosed}, nil)
	fiscalRepo.EXPECT().ListByFiscalYearID(mock.Anything, repositories.ListByFiscalYearIDRequest{FiscalYearID: fyID, OrgID: orgID, BuID: buID}).Return([]*fiscalperiod.FiscalPeriod{{FiscalYearID: fyID, PeriodNumber: 1, Status: fiscalperiod.StatusClosed}, {FiscalYearID: fyID, PeriodNumber: 2, Status: fiscalperiod.StatusClosed}}, nil)

	journalRepo := &fakeJournalPostingRepository{}
	svc := &Service{db: fakeDBConnection{}, repo: &fakeManualJournalRepository{entity: request}, journalRepo: journalRepo, accountingRepo: accountingRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: &Validator{fiscalRepo: fiscalRepo}, auditService: &mocks.NoopAuditService{}}
	result, err := svc.Post(t.Context(), &serviceports.GetManualJournalRequest{RequestID: request.ID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No next open")
	assert.Nil(t, journalRepo.last)
}

type fakeDBConnection struct{}

func (fakeDBConnection) DB() *bun.DB                          { return nil }
func (fakeDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (fakeDBConnection) WithTx(ctx context.Context, _ ports.TxOptions, fn func(context.Context, bun.Tx) error) error {
	return fn(ctx, bun.Tx{})
}
func (fakeDBConnection) HealthCheck(context.Context) error { return nil }
func (fakeDBConnection) IsHealthy(context.Context) bool    { return true }
func (fakeDBConnection) Close() error                      { return nil }

type fakeManualJournalRepository struct {
	entity  *manualjournal.Request
	updated *manualjournal.Request
}

func (f *fakeManualJournalRepository) List(context.Context, *repositories.ListManualJournalRequest) (*pagination.ListResult[*manualjournal.Request], error) {
	return &pagination.ListResult[*manualjournal.Request]{Items: []*manualjournal.Request{cloneRequest(f.entity)}}, nil
}

func (f *fakeManualJournalRepository) GetByID(context.Context, repositories.GetManualJournalByIDRequest) (*manualjournal.Request, error) {
	return cloneRequest(f.entity), nil
}

func (f *fakeManualJournalRepository) Create(_ context.Context, entity *manualjournal.Request) (*manualjournal.Request, error) {
	entity.SyncTotals()
	f.entity = cloneRequest(entity)
	return cloneRequest(entity), nil
}

func (f *fakeManualJournalRepository) Update(_ context.Context, entity *manualjournal.Request) (*manualjournal.Request, error) {
	entity.SyncTotals()
	f.entity = cloneRequest(entity)
	f.updated = cloneRequest(entity)
	return cloneRequest(entity), nil
}

type fakeJournalPostingRepository struct {
	last *repositories.CreateJournalPostingParams
}

func (f *fakeJournalPostingRepository) CreatePosting(_ context.Context, params repositories.CreateJournalPostingParams) error {
	copyParams := params
	copyParams.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
	f.last = &copyParams
	return nil
}

func approvedRequest(orgID, buID, userID pulid.ID) *manualjournal.Request {
	approvedAt := int64(1_700_000_100)
	return &manualjournal.Request{
		ID:                      pulid.MustNew("mjr_"),
		OrganizationID:          orgID,
		BusinessUnitID:          buID,
		RequestNumber:           "MJR-1",
		Status:                  manualjournal.StatusApproved,
		Description:             "Accrue detention",
		AccountingDate:          1_700_000_000,
		RequestedFiscalYearID:   pulid.MustNew("fy_"),
		RequestedFiscalPeriodID: pulid.MustNew("fp_"),
		CurrencyCode:            "USD",
		TotalDebit:              1000,
		TotalCredit:             1000,
		ApprovedAt:              &approvedAt,
		ApprovedByID:            userID,
		CreatedByID:             userID,
		UpdatedByID:             userID,
		Lines: []*manualjournal.Line{
			{ID: pulid.MustNew("mjrl_"), GLAccountID: pulid.MustNew("gla_"), Description: "Debit", DebitAmount: 1000},
			{ID: pulid.MustNew("mjrl_"), GLAccountID: pulid.MustNew("gla_"), Description: "Credit", CreditAmount: 1000},
		},
	}
}
