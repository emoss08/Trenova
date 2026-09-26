package invoiceledger

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerledger"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/journalposting"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixedNumbers struct{}

func (fixedNumbers) GenerateJournalBatchNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "JB-1", nil
}

func (fixedNumbers) GenerateJournalEntryNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "JE-1", nil
}

type capturedLedger struct {
	entries []*customerledger.CustomerLedgerEntry
}

func (c *capturedLedger) AppendEntries(
	_ context.Context,
	entries []*customerledger.CustomerLedgerEntry,
) error {
	c.entries = append(c.entries, entries...)
	return nil
}

var (
	orgID     = pulid.MustNew("org_")
	buID      = pulid.MustNew("bu_")
	arID      = pulid.MustNew("gla_")
	revenueID = pulid.MustNew("gla_")
)

func accrualControl() *tenant.AccountingControl {
	return &tenant.AccountingControl{
		OrganizationID:           orgID,
		AccountingBasis:          tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost,
		JournalPostingMode:       tenant.JournalPostingModeAutomatic,
		DefaultARAccountID:       arID,
		DefaultRevenueAccountID:  revenueID,
	}
}

func memo(billType billingqueue.BillType, total int64) *invoice.Invoice {
	postedAt := int64(1_700_000_000)
	return &invoice.Invoice{
		ID:               pulid.MustNew("inv_"),
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		CustomerID:       pulid.MustNew("cus_"),
		Number:           "CM-1",
		BillType:         billType,
		TotalAmountMinor: total,
		PostedAt:         &postedAt,
		InvoiceDate:      1_699_000_000,
	}
}

func poster(
	t *testing.T,
	control *tenant.AccountingControl,
	periods journalposting.PeriodReader,
	journals repositories.JournalPostingRepository,
	ledger *capturedLedger,
) *Poster {
	t.Helper()
	controls := mocks.NewMockAccountingControlRepository(t)
	controls.EXPECT().GetByOrgID(mock.Anything, orgID).Return(control, nil).Maybe()
	return &Poster{
		Controls: controls,
		Policy: accountingcontrolpolicyservice.New(
			accountingcontrolpolicyservice.Params{Logger: zap.NewNop()},
		),
		Writer: journalposting.Writer{Periods: periods, Numbers: fixedNumbers{}, Journals: journals},
		Ledger: ledger,
	}
}

func TestSourceEventFollowsTheBillType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, tenant.JournalSourceEventInvoicePosted, SourceEvent(billingqueue.BillTypeInvoice))
	assert.Equal(t, tenant.JournalSourceEventCreditMemoPosted, SourceEvent(billingqueue.BillTypeCreditMemo))
	assert.Equal(t, tenant.JournalSourceEventDebitMemoPosted, SourceEvent(billingqueue.BillTypeDebitMemo))
}

func TestACreditMemoPostsRevenueAgainstARAndACreditLedgerLine(t *testing.T) {
	t.Parallel()

	entity := memo(billingqueue.BillTypeCreditMemo, -4_500)
	related := pulid.MustNew("inv_")
	periods := mocks.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{
			OrgID: orgID, BuID: buID, Date: entity.InvoiceDate,
		}).
		Return(&fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), Status: fiscalperiod.StatusOpen}, nil).
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
	ledger := &capturedLedger{}

	result, err := poster(t, accrualControl(), periods, journals, ledger).Post(t.Context(), &Request{
		Invoice:          entity,
		ActorID:          pulid.MustNew("usr_"),
		AccountingDate:   entity.InvoiceDate,
		RelatedInvoiceID: related,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Journal)

	assert.Equal(t, "CreditMemoPosted", written.SourceEventType)
	assert.Equal(t, "invoice-posted:"+entity.ID.String(), written.SourceIdempotencyKey)
	assert.Equal(t, entity.InvoiceDate, written.AccountingDate)
	require.Len(t, written.Lines, 2)
	assert.Equal(t, revenueID, written.Lines[0].GLAccountID)
	assert.Equal(t, int64(4_500), written.Lines[0].DebitAmount)
	assert.Equal(t, arID, written.Lines[1].GLAccountID)
	assert.Equal(t, int64(4_500), written.Lines[1].CreditAmount)
	assert.Equal(t, entity.CustomerID, written.Lines[1].CustomerID)

	require.Len(t, ledger.entries, 1)
	assert.Equal(t, int64(-4_500), ledger.entries[0].AmountMinor)
	assert.Equal(t, related, ledger.entries[0].RelatedInvoiceID)
	assert.Equal(t, entity.InvoiceDate, ledger.entries[0].TransactionDate)
	assert.Equal(t, "CreditMemoPosted", ledger.entries[0].SourceEventType)
}

func TestLedgerOnlyRecordsTheLineWithoutAJournal(t *testing.T) {
	t.Parallel()

	entity := memo(billingqueue.BillTypeCreditMemo, -800)
	ledger := &capturedLedger{}

	result, err := poster(
		t,
		accrualControl(),
		mocks.NewMockFiscalPeriodRepository(t),
		mocks.NewMockJournalPostingRepository(t),
		ledger,
	).Post(t.Context(), &Request{
		Invoice:        entity,
		ActorID:        pulid.MustNew("usr_"),
		AccountingDate: entity.InvoiceDate,
		LedgerOnly:     true,
	})
	require.NoError(t, err)
	assert.Nil(t, result.Journal)
	require.Len(t, ledger.entries, 1)
	assert.Equal(t, int64(-800), ledger.entries[0].AmountMinor)
	assert.Equal(t, entity.InvoiceDate, ledger.entries[0].TransactionDate)
}

func TestNothingIsPostedWhenRevenueIsNotRecognizedOnInvoicePost(t *testing.T) {
	t.Parallel()

	control := accrualControl()
	control.AccountingBasis = tenant.AccountingBasisCash
	ledger := &capturedLedger{}

	_, err := poster(
		t,
		control,
		mocks.NewMockFiscalPeriodRepository(t),
		mocks.NewMockJournalPostingRepository(t),
		ledger,
	).Post(t.Context(), &Request{
		Invoice: memo(billingqueue.BillTypeCreditMemo, -800),
		ActorID: pulid.MustNew("usr_"),
	})
	require.ErrorIs(t, err, ErrNoLedgerEntry)
	assert.Empty(t, ledger.entries)
}

func TestADebitMemoRaisesTheLedger(t *testing.T) {
	t.Parallel()

	ledger := &capturedLedger{}
	_, err := poster(
		t,
		accrualControl(),
		mocks.NewMockFiscalPeriodRepository(t),
		mocks.NewMockJournalPostingRepository(t),
		ledger,
	).Post(t.Context(), &Request{
		Invoice:    memo(billingqueue.BillTypeDebitMemo, 300),
		ActorID:    pulid.MustNew("usr_"),
		LedgerOnly: true,
	})
	require.NoError(t, err)
	require.Len(t, ledger.entries, 1)
	assert.Equal(t, int64(300), ledger.entries[0].AmountMinor)
}
