package driversettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/identitytest"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementjournal"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type paidJournalNumbers struct {
	seqgen.Generator
}

func (paidJournalNumbers) GenerateJournalBatchNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "JB-1", nil
}

func (paidJournalNumbers) GenerateJournalEntryNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "JE-1", nil
}

func postedSettlement(net int64) *driversettlement.Settlement {
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

type settlementStore struct {
	driversettlementports.DriverSettlementRepository
	current *driversettlement.Settlement
	saved   *driversettlement.Settlement
}

func (s *settlementStore) GetByID(
	context.Context,
	driversettlementports.GetDriverSettlementByIDRequest,
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
	auditports.AuditService
}

func (silentAudit) LogAction(*auditports.LogActionParams, ...auditports.LogOption) error {
	return nil
}

func TestMarkPaidWritesThePaymentJournalAndKeepsItsBatch(t *testing.T) {
	t.Parallel()

	const paidAt = int64(1_790_000_000)
	entity := postedSettlement(1_250)
	store := &settlementStore{current: entity}

	accounting := tenantportsmock.NewMockAccountingControlRepository(t)
	accounting.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(&accountingcontrol.AccountingControl{
			DefaultCashAccountID: pulid.MustNew("gla_"),
			JournalPostingMode:   accountingcontrol.JournalPostingModeAutomatic,
		}, nil).
		Once()
	periods := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: pulid.MustNew("fy_"),
			Status:       fiscalperiod.StatusOpen,
		}, nil).
		Once()
	var written journalentryports.CreateJournalPostingParams
	journals := journalentryportsmock.NewMockJournalPostingRepository(t)
	journals.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			written = params
			return nil
		}).
		Once()

	svc := &Service{
		db:             dbtest.NopConnection{},
		settlementRepo: store,
		journals: driversettlementjournal.New(&driversettlementjournal.Params{
			AccountingRepo:   accounting,
			FiscalPeriodRepo: periods,
			JournalRepo:      journals,
			Generator:        paidJournalNumbers{},
		}),
		auditService: silentAudit{},
	}
	paid, err := svc.MarkPaid(t.Context(), &carriersettlementports.MarkSettlementPaidRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SettlementID:  entity.ID,
		PaymentMethod: "ACH",
		PaidAt:        paidAt,
	}, &permissionports.RequestActor{UserID: pulid.MustNew("usr_")})
	require.NoError(t, err)

	assert.Equal(t, driversettlement.StatusPaid, paid.Status)
	require.NotNil(t, paid.PaidJournalBatchID)
	assert.Equal(t, written.BatchID, *paid.PaidJournalBatchID)
	assert.Equal(t, "DriverSettlementPaid", written.SourceEventType)
	assert.Equal(t, paidAt, written.AccountingDate)
	assert.Same(t, store.saved, paid)
}

func TestMarkPaidStampsTheExchangeRateOfAForeignCurrencySettlement(t *testing.T) {
	t.Parallel()

	const paidAt = int64(1_790_000_000)
	entity := postedSettlement(1_250)
	entity.CurrencyCode = "CAD"
	store := &settlementStore{current: entity}

	accounting := tenantportsmock.NewMockAccountingControlRepository(t)
	accounting.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(&accountingcontrol.AccountingControl{
			DefaultCashAccountID: pulid.MustNew("gla_"),
			JournalPostingMode:   accountingcontrol.JournalPostingModeAutomatic,
		}, nil).
		Once()
	periods := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: pulid.MustNew("fy_"),
			Status:       fiscalperiod.StatusOpen,
		}, nil).
		Once()
	journals := journalentryportsmock.NewMockJournalPostingRepository(t)
	journals.EXPECT().CreatePosting(mock.Anything, mock.Anything).Return(nil).Once()

	svc := &Service{
		db:             dbtest.NopConnection{},
		settlementRepo: store,
		journals: driversettlementjournal.New(&driversettlementjournal.Params{
			AccountingRepo:   accounting,
			FiscalPeriodRepo: periods,
			JournalRepo:      journals,
			Generator:        paidJournalNumbers{},
		}),
		auditService: silentAudit{},
		stamper: identitytest.NewFixedForTest(
			t,
			"USD",
			decimal.RequireFromString("0.7312"),
		),
	}
	paid, err := svc.MarkPaid(t.Context(), &carriersettlementports.MarkSettlementPaidRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SettlementID:  entity.ID,
		PaymentMethod: "ACH",
		PaidAt:        paidAt,
	}, &permissionports.RequestActor{UserID: pulid.MustNew("usr_")})
	require.NoError(t, err)

	require.True(t, paid.PaidExchangeRate.Valid)
	assert.True(t, decimal.RequireFromString("0.7312").Equal(paid.PaidExchangeRate.Decimal))
	require.NotNil(t, paid.PaidExchangeRateDate)
	assert.Equal(t, timeutils.DayStartUTC(paidAt), *paid.PaidExchangeRateDate)
	assert.False(t, paid.ExchangeRate.Valid, "paying does not touch the posting stamp")
}
