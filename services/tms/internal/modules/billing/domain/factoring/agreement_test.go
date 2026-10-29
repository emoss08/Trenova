package factoring_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func intPtr(v int) *int { return &v }

func validAgreement() *factoring.Agreement {
	return &factoring.Agreement{
		OrganizationID:       "org_1",
		BusinessUnitID:       "bu_1",
		FactoringCompanyID:   factorID,
		Status:               factoring.AgreementStatusDraft,
		EffectiveFrom:        now,
		AdvanceRateBP:        9000,
		ReserveRateBP:        1000,
		ChargebackWindowDays: 90,
		Recourse:             factoring.RecourseFull,
		AccountingTreatment:  factoring.AccountingTreatmentSale,
		FundingDays:          1,
		SubmissionTime:       600,
		FeeTiers: []*factoring.FeeTier{
			{DaysFrom: 30, RateBP: 500},
			{DaysFrom: 0, DaysTo: intPtr(30), RateBP: 300},
		},
	}
}

func errorFields(a *factoring.Agreement) []string {
	multiErr := errortypes.NewMultiError()
	a.Validate(multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	return fields
}

func TestAgreementValidate(t *testing.T) {
	t.Parallel()

	assert.Empty(t, errorFields(validAgreement()), "tiers may be given in any order")

	cases := map[string]struct {
		mutate func(*factoring.Agreement)
		field  string
	}{
		"advance and reserve over 100%": {
			func(a *factoring.Agreement) { a.ReserveRateBP = 1500 }, "reserveRateBp",
		},
		"ends before it starts": {
			func(a *factoring.Agreement) { end := now; a.EffectiveTo = &end }, "effectiveTo",
		},
		"no fee tiers": {
			func(a *factoring.Agreement) { a.FeeTiers = nil }, "feeTiers",
		},
		"a gap between tiers": {
			func(a *factoring.Agreement) { a.FeeTiers[0].DaysFrom = 31 }, "feeTiers[0].daysFrom",
		},
		"an open-ended tier before the last": {
			func(a *factoring.Agreement) { a.FeeTiers[1].DaysTo = nil }, "feeTiers[1].daysTo",
		},
		"submission time past midnight": {
			func(a *factoring.Agreement) { a.SubmissionTime = 1440 }, "submissionTime",
		},
		"chargeback window too long": {
			func(a *factoring.Agreement) { a.ChargebackWindowDays = 731 }, "chargebackWindowDays",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := validAgreement()
			tc.mutate(a)
			assert.Contains(t, errorFields(a), tc.field)
		})
	}
}

func TestAgreementPostingAccounts(t *testing.T) {
	t.Parallel()

	defaultAR := pulid.MustNew("gla_")
	defaultCash := pulid.MustNew("gla_")
	agreement := &factoring.Agreement{}

	accounts := agreement.PostingAccounts(defaultAR, defaultCash)
	assert.Equal(t, defaultCash, accounts.Cash)
	fields := func(missing []factoring.MissingAccount) []string {
		out := make([]string, 0, len(missing))
		for _, m := range missing {
			out = append(out, m.Field)
		}
		return out
	}
	assert.Equal(
		t,
		[]string{"feeExpenseAccountId", "dueFromFactorAccountId"},
		fields(accounts.Missing(factoring.AccountingTreatmentSale)),
	)
	assert.Equal(
		t,
		[]string{"feeExpenseAccountId", "liabilityAccountId"},
		fields(accounts.Missing(factoring.AccountingTreatmentSecuredBorrowing)),
	)

	agreement.CashAccountID = pulid.MustNew("gla_")
	agreement.FeeExpenseAccountID = pulid.MustNew("gla_")
	agreement.DueFromFactorAccountID = pulid.MustNew("gla_")
	accounts = agreement.PostingAccounts(pulid.Nil, pulid.Nil)
	assert.Equal(t, agreement.CashAccountID, accounts.Cash)
	assert.Equal(
		t,
		[]string{"defaultArAccountId"},
		fields(accounts.Missing(factoring.AccountingTreatmentSale)),
	)

	agreement.ChargebackWindowDays = 90
	assert.Equal(t, now+90*86_400, agreement.ChargebackDeadline(now))
}
