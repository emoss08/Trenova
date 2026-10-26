package journalreversalservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalreversalportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateJournalReversalPendingApprovalWhenConfigured(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	entryRepo := journalentryportsmock.NewMockJournalEntryRepository(t)
	entryRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalentryports.GetJournalEntryByIDRequest{
				ID:         entryID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(postedEntry(entryID, orgID, buID), nil)
	reversalRepo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	reversalRepo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*journalreversal.Reversal")).
		RunAndReturn(func(_ context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
			copy := *entity
			return &copy, nil
		})
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{JournalReversalPolicy: tenant.JournalReversalPolicyNextOpenPeriod, RequireManualJEApproval: true}, nil)
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 1_700_000_000}).
		Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID, Status: fiscalperiod.StatusOpen}, nil)

	svc := &Service{
		journalEntryRepo:    entryRepo,
		journalReversalRepo: reversalRepo,
		accountingRepo:      accountingRepo,
		validator:           &Validator{fiscalRepo: fiscalRepo},
		auditService:        &mocks.NoopAuditService{},
	}
	result, err := svc.Create(
		t.Context(),
		&journalreversalports.CreateJournalReversalRequest{
			OriginalJournalEntryID:  entryID,
			RequestedAccountingDate: 1_700_000_000,
			ReasonCode:              "ERR",
			ReasonText:              "reverse",
			TenantInfo: pagination.TenantInfo{
				OrgID:  orgID,
				BuID:   buID,
				UserID: userID,
			},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, journalreversal.StatusPendingApproval, result.Status)
	assert.True(t, result.ApprovedByID.IsNil())
}

func TestCreateJournalReversalBlockedWhenPolicyDisallows(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	entryRepo := journalentryportsmock.NewMockJournalEntryRepository(t)
	entryRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalentryports.GetJournalEntryByIDRequest{
				ID:         entryID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(postedEntry(entryID, orgID, buID), nil)
	reversalRepo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{JournalReversalPolicy: tenant.JournalReversalPolicyDisallow}, nil)

	svc := &Service{
		journalEntryRepo:    entryRepo,
		journalReversalRepo: reversalRepo,
		accountingRepo:      accountingRepo,
		validator:           &Validator{},
		auditService:        &mocks.NoopAuditService{},
	}
	result, err := svc.Create(
		t.Context(),
		&journalreversalports.CreateJournalReversalRequest{
			OriginalJournalEntryID:  entryID,
			RequestedAccountingDate: 1_700_000_000,
			ReasonCode:              "ERR",
			ReasonText:              "reverse",
			TenantInfo: pagination.TenantInfo{
				OrgID:  orgID,
				BuID:   buID,
				UserID: userID,
			},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestListAndGetJournalReversal(t *testing.T) {
	t.Parallel()

	entity := &journalreversal.Reversal{
		ID:     pulid.MustNew("jrev_"),
		Status: journalreversal.StatusRequested,
	}
	repo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	repo.EXPECT().
		List(mock.Anything, mock.Anything).
		Return(
			&pagination.ListResult[*journalreversal.Reversal]{
				Items:      []*journalreversal.Reversal{entity},
				Total:      1,
				TotalExact: true,
			},
			nil,
		)
	repo.EXPECT().
		GetByID(
			mock.Anything,
			journalreversalports.GetJournalReversalByIDRequest{
				ID:         entity.ID,
				TenantInfo: pagination.TenantInfo{},
			},
		).
		Return(entity, nil)
	svc := &Service{journalReversalRepo: repo}

	list, err := svc.List(
		t.Context(),
		&journalreversalports.ListJournalReversalsRequest{
			Filter: &pagination.QueryOptions{Pagination: pagination.Info{Limit: 10}},
		},
	)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)

	got, err := svc.Get(
		t.Context(),
		&journalreversalports.GetJournalReversalRequest{
			ReversalID: entity.ID,
			TenantInfo: pagination.TenantInfo{},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, entity.ID, got.ID)
}

func TestApproveRejectAndCancelJournalReversal(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	base := &journalreversal.Reversal{
		ID:             pulid.MustNew("jrev_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         journalreversal.StatusRequested,
	}

	approveRepo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	approveRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalreversalports.GetJournalReversalByIDRequest{
				ID:         base.ID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(base, nil)
	approveRepo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*journalreversal.Reversal")).
		RunAndReturn(func(_ context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
			copy := *entity
			return &copy, nil
		})
	svcApprove := &Service{
		journalReversalRepo: approveRepo,
		validator:           &Validator{},
		auditService:        &mocks.NoopAuditService{},
	}
	approved, err := svcApprove.Approve(
		t.Context(),
		&journalreversalports.GetJournalReversalRequest{
			ReversalID: base.ID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, journalreversal.StatusApproved, approved.Status)
	assert.Equal(t, userID, approved.ApprovedByID)

	rejectEntity := &journalreversal.Reversal{
		ID:             pulid.MustNew("jrev_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         journalreversal.StatusRequested,
	}
	rejectRepo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	rejectRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalreversalports.GetJournalReversalByIDRequest{
				ID:         rejectEntity.ID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(rejectEntity, nil)
	rejectRepo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*journalreversal.Reversal")).
		RunAndReturn(func(_ context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
			copy := *entity
			return &copy, nil
		})
	svcReject := &Service{
		journalReversalRepo: rejectRepo,
		validator:           &Validator{},
		auditService:        &mocks.NoopAuditService{},
	}
	rejected, err := svcReject.Reject(
		t.Context(),
		&journalreversalports.RejectJournalReversalRequest{
			ReversalID: rejectEntity.ID,
			Reason:     "bad",
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, journalreversal.StatusRejected, rejected.Status)
	assert.Equal(t, "bad", rejected.RejectionReason)

	cancelEntity := &journalreversal.Reversal{
		ID:             pulid.MustNew("jrev_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         journalreversal.StatusApproved,
	}
	cancelRepo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	cancelRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalreversalports.GetJournalReversalByIDRequest{
				ID:         cancelEntity.ID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(cancelEntity, nil)
	cancelRepo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*journalreversal.Reversal")).
		RunAndReturn(func(_ context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
			copy := *entity
			return &copy, nil
		})
	svcCancel := &Service{
		journalReversalRepo: cancelRepo,
		validator:           &Validator{},
		auditService:        &mocks.NoopAuditService{},
	}
	cancelled, err := svcCancel.Cancel(
		t.Context(),
		&journalreversalports.CancelJournalReversalRequest{
			ReversalID: cancelEntity.ID,
			Reason:     "stop",
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, journalreversal.StatusCancelled, cancelled.Status)
	assert.Equal(t, "stop", cancelled.CancelReason)
}

func TestPostBlocksAlreadyReversedEntry(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	original := postedEntry(entryID, orgID, buID)
	original.ReversedByID = pulid.MustNew("je_")
	reversal := &journalreversal.Reversal{
		ID:                      pulid.MustNew("jrev_"),
		OrganizationID:          orgID,
		BusinessUnitID:          buID,
		OriginalJournalEntryID:  entryID,
		Status:                  journalreversal.StatusApproved,
		RequestedAccountingDate: 1_700_000_000,
		ResolvedFiscalYearID:    pulid.MustNew("fy_"),
		ResolvedFiscalPeriodID:  pulid.MustNew("fp_"),
		ReasonCode:              "ERR",
		ReasonText:              "reverse me",
		RequestedByID:           userID,
	}

	entryRepo := journalentryportsmock.NewMockJournalEntryRepository(t)
	entryRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalentryports.GetJournalEntryByIDRequest{
				ID:         entryID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(cloneEntry(original), nil)
	reversalRepo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	reversalRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalreversalports.GetJournalReversalByIDRequest{
				ID:         reversal.ID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(reversal, nil)
	svc := &Service{
		db:                  dbtest.NopConnection{},
		journalEntryRepo:    entryRepo,
		journalReversalRepo: reversalRepo,
		journalPostingRepo:  journalentryportsmock.NewMockJournalPostingRepository(t),
		sequenceGenerator:   testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		auditService:        &mocks.NoopAuditService{},
	}
	result, err := svc.Post(
		t.Context(),
		&journalreversalports.GetJournalReversalRequest{
			ReversalID: reversal.ID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been reversed")
}

func TestPostCreatesReversalAndMarksOriginalReversed(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	reversalID := pulid.MustNew("jrev_")
	original := postedEntry(entryID, orgID, buID)
	reversal := &journalreversal.Reversal{
		ID:                      reversalID,
		OrganizationID:          orgID,
		BusinessUnitID:          buID,
		OriginalJournalEntryID:  entryID,
		Status:                  journalreversal.StatusApproved,
		RequestedAccountingDate: 1_700_000_000,
		ResolvedFiscalYearID:    pulid.MustNew("fy_"),
		ResolvedFiscalPeriodID:  pulid.MustNew("fp_"),
		ReasonCode:              "ERR",
		ReasonText:              "reverse me",
		RequestedByID:           userID,
	}

	reversalRepo := journalreversalportsmock.NewMockJournalReversalRepository(t)
	reversalRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalreversalports.GetJournalReversalByIDRequest{
				ID:         reversalID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(reversal, nil)
	reversalRepo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*journalreversal.Reversal")).
		RunAndReturn(func(_ context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
			copy := *entity
			return &copy, nil
		})
	entryRepo := journalentryportsmock.NewMockJournalEntryRepository(t)
	entryRepo.EXPECT().
		GetByID(
			mock.Anything,
			journalentryports.GetJournalEntryByIDRequest{
				ID:         entryID,
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
			},
		).
		Return(cloneEntry(original), nil)
	var markReq *journalentryports.MarkJournalEntryReversedRequest
	entryRepo.EXPECT().
		MarkReversed(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req journalentryports.MarkJournalEntryReversedRequest) error {
			copy := req
			markReq = &copy
			return nil
		})
	postingRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	var postingParams *journalentryports.CreateJournalPostingParams
	postingRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			copyParams := params
			copyParams.Lines = append([]journalentryports.JournalPostingLine(nil), params.Lines...)
			postingParams = &copyParams
			return nil
		})
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{FunctionalCurrencyCode: "USD"}, nil)
	svc := &Service{
		db:                  dbtest.NopConnection{},
		journalEntryRepo:    entryRepo,
		journalReversalRepo: reversalRepo,
		journalPostingRepo:  postingRepo,
		accountingRepo:      accountingRepo,
		sequenceGenerator:   testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		auditService:        &mocks.NoopAuditService{},
	}

	result, err := svc.Post(
		t.Context(),
		&journalreversalports.GetJournalReversalRequest{
			ReversalID: reversalID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, journalreversal.StatusPosted, result.Status)
	require.NotNil(t, postingParams)
	assert.True(t, postingParams.IsReversal)
	assert.Equal(t, entryID, postingParams.ReversalOfID)
	require.Len(t, postingParams.Lines, 2)
	assert.Equal(t, original.Lines[0].CreditAmount, postingParams.Lines[0].DebitAmount)
	assert.Equal(t, original.Lines[0].DebitAmount, postingParams.Lines[0].CreditAmount)
	require.NotNil(t, markReq)
	assert.Equal(t, entryID, markReq.OriginalEntryID)
	assert.Equal(t, result.ReversalJournalEntryID, markReq.ReversalEntryID)
}

func postedEntry(id, orgID, buID pulid.ID) *journalentry.JournalEntry {
	return &journalentry.JournalEntry{
		ID:             id,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		EntryNumber:    "JE-1",
		Status:         "Posted",
		IsPosted:       true,
		Lines: []*journalentry.JournalEntryLine{
			{
				GLAccountID:  pulid.MustNew("gla_"),
				LineNumber:   1,
				Description:  "Debit",
				DebitAmount:  1000,
				CreditAmount: 0,
				NetAmount:    1000,
			},
			{
				GLAccountID:  pulid.MustNew("gla_"),
				LineNumber:   2,
				Description:  "Credit",
				DebitAmount:  0,
				CreditAmount: 1000,
				NetAmount:    -1000,
			},
		},
	}
}

func cloneEntry(src *journalentry.JournalEntry) *journalentry.JournalEntry {
	if src == nil {
		return nil
	}
	copy := *src
	copy.Lines = make([]*journalentry.JournalEntryLine, 0, len(src.Lines))
	for _, line := range src.Lines {
		if line == nil {
			copy.Lines = append(copy.Lines, nil)
			continue
		}
		l := *line
		copy.Lines = append(copy.Lines, &l)
	}
	return &copy
}
