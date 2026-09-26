package carriersettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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

	periods := mocks.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
			Date:  paidAt,
		}).
		Return(period, nil).
		Once()

	var posted repositories.CreateJournalPostingParams
	journals := mocks.NewMockJournalPostingRepository(t)
	journals.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
			posted = params
			return nil
		}).
		Once()

	svc := &Service{journalRepo: journals, fiscalPeriodRepo: periods, generator: numberingOnly{}}
	_, err := svc.createJournalPosting(t.Context(), &createJournalPostingParams{
		Entity:         entity,
		Actor:          &serviceports.RequestActor{UserID: pulid.MustNew("usr_")},
		Control:        &tenant.AccountingControl{},
		Legs:           []PostingLeg{{AccountID: payableAccountID, Debit: 100}, {AccountID: cashAccountID, Credit: 100}},
		Description:    "Payment of carrier settlement CS-1042",
		SourceEvent:    tenant.JournalSourceEventCarrierSettlementPaid,
		IdempotencyKey: "carrier-settlement-paid:" + entity.ID.String(),
		AccountingDate: paidAt,
	})
	require.NoError(t, err)
	assert.Equal(t, paidAt, posted.AccountingDate)
	assert.Equal(t, period.ID, posted.FiscalPeriodID)
}
