package driversettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type journalNumbers struct {
	seqgen.Generator
}

func (journalNumbers) GenerateJournalBatchNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "JB-1", nil
}

func (journalNumbers) GenerateJournalEntryNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "JE-1", nil
}

func paidSettlement(net int64) *driversettlement.Settlement {
	payable := pulid.MustNew("gla_")
	return &driversettlement.Settlement{
		ID:                     pulid.MustNew("dstl_"),
		OrganizationID:         pulid.MustNew("org_"),
		BusinessUnitID:         pulid.MustNew("bu_"),
		SettlementNumber:       "DS-1042",
		Status:                 driversettlement.StatusPosted,
		NetPayMinor:            net,
		PostedPayableAccountID: &payable,
	}
}

func TestPaymentLegsRelieveThePayableAgainstCash(t *testing.T) {
	t.Parallel()

	payable, cash := pulid.MustNew("gla_"), pulid.MustNew("gla_")

	owed := BuildSettlementPaymentLegs(paidSettlement(1_250), payable, cash)
	assert.Equal(t, []PostingLeg{
		{AccountID: payable, Debit: 1_250},
		{AccountID: cash, Credit: 1_250},
	}, owed)

	collected := BuildSettlementPaymentLegs(paidSettlement(-300), payable, cash)
	assert.Equal(t, []PostingLeg{
		{AccountID: cash, Debit: 300},
		{AccountID: payable, Credit: 300},
	}, collected)

	assert.Empty(t, BuildSettlementPaymentLegs(paidSettlement(0), payable, cash))
}

func TestPaymentJournalIsWrittenOnThePaidDateAgainstThePostedPayable(t *testing.T) {
	t.Parallel()

	const paidAt = int64(1_790_000_000)
	entity := paidSettlement(1_250)
	cash := pulid.MustNew("gla_")
	control := &tenant.AccountingControl{
		OrganizationID:       entity.OrganizationID,
		DefaultCashAccountID: cash,
		JournalPostingMode:   tenant.JournalPostingModeAutomatic,
	}

	accounting := mocks.NewMockAccountingControlRepository(t)
	accounting.EXPECT().GetByOrgID(mock.Anything, entity.OrganizationID).Return(control, nil).Once()

	periods := mocks.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
			Date:  paidAt,
		}).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: pulid.MustNew("fy_"),
			Status:       fiscalperiod.StatusOpen,
		}, nil).
		Once()

	var written repositories.CreateJournalPostingParams
	journals := mocks.NewMockJournalPostingRepository(t)
	journals.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
			written = params
			return nil
		}).
		Once()

	svc := &Service{
		accountingRepo:   accounting,
		fiscalPeriodRepo: periods,
		journalRepo:      journals,
		generator:        journalNumbers{},
	}
	batchID, err := svc.postPaymentJournal(t.Context(), entity, pulid.MustNew("usr_"), paidAt)
	require.NoError(t, err)
	require.NotNil(t, batchID)

	assert.Equal(t, *batchID, written.BatchID)
	assert.Equal(t, paidAt, written.AccountingDate)
	assert.Equal(t, "DriverSettlementPaid", written.SourceEventType)
	assert.Equal(t, "DriverSettlement", written.SourceObjectType)
	assert.Equal(t, entity.ID.String(), written.SourceObjectID)
	assert.Equal(t, "driver-settlement-paid:"+entity.ID.String(), written.SourceIdempotencyKey)
	assert.Equal(t, "DS-1042", written.ReferenceNumber)
	assert.True(t, written.IsPosted)
	require.Len(t, written.Lines, 2)
	assert.Equal(t, *entity.PostedPayableAccountID, written.Lines[0].GLAccountID)
	assert.Equal(t, int64(1_250), written.Lines[0].DebitAmount)
	assert.Equal(t, cash, written.Lines[1].GLAccountID)
	assert.Equal(t, int64(1_250), written.Lines[1].CreditAmount)
}

func TestPaymentJournalNeedsACashAccount(t *testing.T) {
	t.Parallel()

	entity := paidSettlement(1_250)
	accounting := mocks.NewMockAccountingControlRepository(t)
	accounting.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(&tenant.AccountingControl{}, nil).
		Once()
	_, err := (&Service{accountingRepo: accounting}).
		postPaymentJournal(t.Context(), entity, pulid.MustNew("usr_"), 1)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "accountingControl", validation.Field)
}

func TestASettlementPostedWithoutAJournalRecordsNoPaymentJournal(t *testing.T) {
	t.Parallel()

	entity := paidSettlement(1_250)
	entity.PostedPayableAccountID = nil

	batchID, err := (&Service{}).postPaymentJournal(t.Context(), entity, pulid.MustNew("usr_"), 1)
	require.NoError(t, err)
	assert.Nil(t, batchID)
}

func TestASettlementPostedBeforeThePayableSnapshotPaysFromTheDefaultPayable(t *testing.T) {
	t.Parallel()

	postedBatch := pulid.MustNew("jb_")
	entity := paidSettlement(1_250)
	entity.PostedPayableAccountID = nil
	entity.PostedJournalBatchID = &postedBatch
	control := &tenant.AccountingControl{
		DefaultSettlementsPayableAccountID: pulid.MustNew("gla_"),
		DefaultCashAccountID:               pulid.MustNew("gla_"),
	}

	journal, err := PaymentJournal(entity, control, pulid.MustNew("usr_"), 1, 1)
	require.NoError(t, err)
	require.NotNil(t, journal)
	assert.Equal(t, []PostingLeg{
		{AccountID: control.DefaultSettlementsPayableAccountID, Debit: 1_250},
		{AccountID: control.DefaultCashAccountID, Credit: 1_250},
	}, journal.Legs)

	snapshot := pulid.MustNew("gla_")
	entity.PostedPayableAccountID = &snapshot
	journal, err = PaymentJournal(entity, control, pulid.MustNew("usr_"), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, snapshot, journal.Legs[0].AccountID)

	entity.PostedPayableAccountID = nil
	control.DefaultSettlementsPayableAccountID = pulid.Nil
	_, err = PaymentJournal(entity, control, pulid.MustNew("usr_"), 1, 1)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "accountingControl", validation.Field)
}

func TestAZeroSettlementRecordsNoPaymentJournal(t *testing.T) {
	t.Parallel()

	batchID, err := (&Service{}).postPaymentJournal(t.Context(), paidSettlement(0), pulid.MustNew("usr_"), 1)
	require.NoError(t, err)
	assert.Nil(t, batchID)
}

type settlementStore struct {
	repositories.DriverSettlementRepository
	current *driversettlement.Settlement
	saved   *driversettlement.Settlement
}

func (s *settlementStore) GetByID(
	context.Context,
	repositories.GetDriverSettlementByIDRequest,
) (*driversettlement.Settlement, error) {
	copied := *s.current
	return &copied, nil
}

func (s *settlementStore) Update(
	_ context.Context,
	entity *driversettlement.Settlement,
) (*driversettlement.Settlement, error) {
	s.saved = entity
	return entity, nil
}

type silentAudit struct {
	serviceports.AuditService
}

func (silentAudit) LogAction(*serviceports.LogActionParams, ...serviceports.LogOption) error {
	return nil
}

func TestMarkPaidWritesThePaymentJournalAndKeepsItsBatch(t *testing.T) {
	t.Parallel()

	const paidAt = int64(1_790_000_000)
	entity := paidSettlement(1_250)
	store := &settlementStore{current: entity}

	accounting := mocks.NewMockAccountingControlRepository(t)
	accounting.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(&tenant.AccountingControl{
			DefaultCashAccountID: pulid.MustNew("gla_"),
			JournalPostingMode:   tenant.JournalPostingModeAutomatic,
		}, nil).
		Once()
	periods := mocks.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: pulid.MustNew("fy_"),
			Status:       fiscalperiod.StatusOpen,
		}, nil).
		Once()
	var written repositories.CreateJournalPostingParams
	journals := mocks.NewMockJournalPostingRepository(t)
	journals.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
			written = params
			return nil
		}).
		Once()

	svc := &Service{
		db:               dbtest.NopConnection{},
		settlementRepo:   store,
		accountingRepo:   accounting,
		fiscalPeriodRepo: periods,
		journalRepo:      journals,
		generator:        journalNumbers{},
		auditService:     silentAudit{},
	}
	paid, err := svc.MarkPaid(t.Context(), &serviceports.MarkSettlementPaidRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SettlementID:  entity.ID,
		PaymentMethod: "ACH",
		PaidAt:        paidAt,
	}, &serviceports.RequestActor{UserID: pulid.MustNew("usr_")})
	require.NoError(t, err)

	assert.Equal(t, driversettlement.StatusPaid, paid.Status)
	require.NotNil(t, paid.PaidJournalBatchID)
	assert.Equal(t, written.BatchID, *paid.PaidJournalBatchID)
	assert.Equal(t, "DriverSettlementPaid", written.SourceEventType)
	assert.Equal(t, paidAt, written.AccountingDate)
	assert.Same(t, store.saved, paid)
}
