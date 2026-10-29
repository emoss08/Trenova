package carriersettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type numberingOnly struct {
	seqgen.Generator
}

func (numberingOnly) GenerateJournalBatchNumber(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
	string,
) (string, error) {
	return "JB-1", nil
}

func (numberingOnly) GenerateJournalEntryNumber(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
	string,
) (string, error) {
	return "JE-1", nil
}

func TestPaymentJournalIsDatedWhenThePaymentWasMade(t *testing.T) {
	t.Parallel()

	const paidAt = int64(1_790_000_000)
	entity := newPostingSettlement()
	entity.SettlementNumber = "CS-1042"
	period := &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: pulid.MustNew("fy_"),
		Status:       fiscalperiod.StatusOpen,
	}

	periods := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
			Date:  paidAt,
		}).
		Return(period, nil).
		Once()

	var posted journalentryports.CreateJournalPostingParams
	journals := journalentryportsmock.NewMockJournalPostingRepository(t)
	journals.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			posted = params
			return nil
		}).
		Once()

	svc := &Service{journalRepo: journals, fiscalPeriodRepo: periods, generator: numberingOnly{}}
	actor := &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}
	draft, err := svc.draftJournal(t.Context(), entity, actor.UserID, &createJournalPostingParams{
		Control: &accountingcontrol.AccountingControl{},
		Legs: []PostingLeg{
			{AccountID: payableAccountID, Debit: 100},
			{AccountID: cashAccountID, Credit: 100},
		},
		Description:    "Payment of carrier settlement CS-1042",
		SourceEvent:    accountingcontrol.JournalSourceEventCarrierSettlementPaid,
		IdempotencyKey: "carrier-settlement-paid:" + entity.ID.String(),
		AccountingDate: paidAt,
	})
	require.NoError(t, err)
	assert.Equal(t, paidAt, draft.plan.AccountingDate)
	_, err = svc.writeJournal(t.Context(), entity, actor, draft)
	require.NoError(t, err)
	assert.Equal(t, paidAt, posted.AccountingDate)
	assert.Equal(t, period.ID, posted.FiscalPeriodID)
}
