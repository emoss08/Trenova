package journalreversalservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestCreateJournalReversalPendingApprovalWhenConfigured(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	entryRepo := &fakeJournalEntryRepository{entry: postedEntry(entryID, orgID, buID)}
	reversalRepo := &fakeJournalReversalRepository{}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{JournalReversalPolicy: tenant.JournalReversalPolicyNextOpenPeriod, RequireManualJEApproval: true}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 1_700_000_000}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID, Status: fiscalperiod.StatusOpen}, nil)

	svc := &Service{journalEntryRepo: entryRepo, journalReversalRepo: reversalRepo, accountingRepo: accountingRepo, validator: &Validator{fiscalRepo: fiscalRepo}, auditService: &mocks.NoopAuditService{}}
	result, err := svc.Create(t.Context(), &serviceports.CreateJournalReversalRequest{OriginalJournalEntryID: entryID, RequestedAccountingDate: 1_700_000_000, ReasonCode: "ERR", ReasonText: "reverse", TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, journalreversal.StatusPendingApproval, result.Status)
	assert.True(t, result.ApprovedByID.IsNil())
}

func TestPostCreatesReversalAndMarksOriginalReversed(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entryID := pulid.MustNew("je_")
	reversalID := pulid.MustNew("jrev_")
	original := postedEntry(entryID, orgID, buID)
	reversal := &journalreversal.Reversal{ID: reversalID, OrganizationID: orgID, BusinessUnitID: buID, OriginalJournalEntryID: entryID, Status: journalreversal.StatusApproved, RequestedAccountingDate: 1_700_000_000, ResolvedFiscalYearID: pulid.MustNew("fy_"), ResolvedFiscalPeriodID: pulid.MustNew("fp_"), ReasonCode: "ERR", ReasonText: "reverse me", RequestedByID: userID}

	reversalRepo := &fakeJournalReversalRepository{entity: reversal}
	entryRepo := &fakeJournalEntryRepository{entry: original}
	postingRepo := &fakeReversalPostingRepository{}
	svc := &Service{db: fakeReversalDB{}, journalEntryRepo: entryRepo, journalReversalRepo: reversalRepo, journalPostingRepo: postingRepo, sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, auditService: &mocks.NoopAuditService{}}

	result, err := svc.Post(t.Context(), &serviceports.GetJournalReversalRequest{ReversalID: reversalID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, journalreversal.StatusPosted, result.Status)
	require.NotNil(t, postingRepo.last)
	assert.True(t, postingRepo.last.IsReversal)
	assert.Equal(t, entryID, postingRepo.last.ReversalOfID)
	require.Len(t, postingRepo.last.Lines, 2)
	assert.Equal(t, original.Lines[0].CreditAmount, postingRepo.last.Lines[0].DebitAmount)
	assert.Equal(t, original.Lines[0].DebitAmount, postingRepo.last.Lines[0].CreditAmount)
	require.NotNil(t, entryRepo.markReq)
	assert.Equal(t, entryID, entryRepo.markReq.OriginalEntryID)
	assert.Equal(t, result.ReversalJournalEntryID, entryRepo.markReq.ReversalEntryID)
}

type fakeReversalDB struct{}

func (fakeReversalDB) DB() *bun.DB                          { return nil }
func (fakeReversalDB) DBForContext(context.Context) bun.IDB { return nil }
func (fakeReversalDB) WithTx(ctx context.Context, _ ports.TxOptions, fn func(context.Context, bun.Tx) error) error {
	return fn(ctx, bun.Tx{})
}
func (fakeReversalDB) HealthCheck(context.Context) error { return nil }
func (fakeReversalDB) IsHealthy(context.Context) bool    { return true }
func (fakeReversalDB) Close() error                      { return nil }

type fakeJournalEntryRepository struct {
	entry   *journalentry.Entry
	markReq *repositories.MarkJournalEntryReversedRequest
}

func (f *fakeJournalEntryRepository) GetByID(context.Context, repositories.GetJournalEntryByIDRequest) (*journalentry.Entry, error) {
	return cloneEntry(f.entry), nil
}
func (f *fakeJournalEntryRepository) MarkReversed(_ context.Context, req repositories.MarkJournalEntryReversedRequest) error {
	f.markReq = &req
	return nil
}

type fakeJournalReversalRepository struct {
	entity *journalreversal.Reversal
}

func (f *fakeJournalReversalRepository) List(context.Context, *repositories.ListJournalReversalsRequest) (*pagination.ListResult[*journalreversal.Reversal], error) {
	return &pagination.ListResult[*journalreversal.Reversal]{Items: []*journalreversal.Reversal{f.entity}, Total: 1}, nil
}
func (f *fakeJournalReversalRepository) GetByID(context.Context, repositories.GetJournalReversalByIDRequest) (*journalreversal.Reversal, error) {
	if f.entity == nil {
		return nil, nil
	}
	copy := *f.entity
	return &copy, nil
}
func (f *fakeJournalReversalRepository) Create(_ context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
	copy := *entity
	f.entity = &copy
	return &copy, nil
}
func (f *fakeJournalReversalRepository) Update(_ context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
	copy := *entity
	f.entity = &copy
	return &copy, nil
}

type fakeReversalPostingRepository struct {
	last *repositories.CreateJournalPostingParams
}

func (f *fakeReversalPostingRepository) CreatePosting(_ context.Context, params repositories.CreateJournalPostingParams) error {
	copyParams := params
	copyParams.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
	f.last = &copyParams
	return nil
}

func postedEntry(id, orgID, buID pulid.ID) *journalentry.Entry {
	return &journalentry.Entry{ID: id, OrganizationID: orgID, BusinessUnitID: buID, EntryNumber: "JE-1", Status: "Posted", IsPosted: true, Lines: []*journalentry.Line{{GLAccountID: pulid.MustNew("gla_"), LineNumber: 1, Description: "Debit", DebitAmount: 1000, CreditAmount: 0, NetAmount: 1000}, {GLAccountID: pulid.MustNew("gla_"), LineNumber: 2, Description: "Credit", DebitAmount: 0, CreditAmount: 1000, NetAmount: -1000}}}
}

func cloneEntry(src *journalentry.Entry) *journalentry.Entry {
	if src == nil {
		return nil
	}
	copy := *src
	copy.Lines = make([]*journalentry.Line, 0, len(src.Lines))
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
