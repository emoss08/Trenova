package accountingcontrolresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountingControlResolvers_CheckTheAccountingControlPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"accountingControl", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AccountingControl(ctx)
			return err
		}},
		{
			"updateAccountingControl",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateAccountingControl(
					ctx,
					gqlmodel.UpdateAccountingControlInput{},
				)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceAccountingControl,
				call.operation,
			)
		})
	}
}

func TestAccountingControlFromInput_MapsEveryFieldOntoTheStoredControl(t *testing.T) {
	t.Parallel()

	existing := &accountingcontrol.AccountingControl{
		ID:             pulid.MustNew("x_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Version:        3,
		CreatedAt:      100,
		UpdatedAt:      200,
	}
	v20 := pulid.MustNew("x_")
	v20Raw := v20.String()
	v21 := pulid.MustNew("x_")
	v21Raw := v21.String()
	v22 := pulid.MustNew("x_")
	v22Raw := v22.String()
	v23 := pulid.MustNew("x_")
	v23Raw := v23.String()
	v24 := pulid.MustNew("x_")
	v24Raw := v24.String()
	v25 := pulid.MustNew("x_")
	v25Raw := v25.String()
	v26 := pulid.MustNew("x_")
	v26Raw := v26.String()
	v27 := pulid.MustNew("x_")
	v27Raw := v27.String()
	v28 := pulid.MustNew("x_")
	v28Raw := v28.String()
	v29 := pulid.MustNew("x_")
	v29Raw := v29.String()
	v30 := pulid.MustNew("x_")
	v30Raw := v30.String()
	v31 := pulid.MustNew("x_")
	v31Raw := v31.String()
	v32 := pulid.MustNew("x_")
	v32Raw := v32.String()
	v33 := pulid.MustNew("x_")
	v33Raw := v33.String()
	v34 := pulid.MustNew("x_")
	v34Raw := v34.String()
	v35 := pulid.MustNew("x_")
	v35Raw := v35.String()
	v36 := pulid.MustNew("x_")
	v36Raw := v36.String()
	v37 := pulid.MustNew("x_")
	v37Raw := v37.String()

	entity, err := accountingControlFromInput(existing, &gqlmodel.UpdateAccountingControlInput{
		Version:         3,
		AccountingBasis: accountingcontrol.AccountingBasisType("Cash"),
		RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionPolicyType(
			"OnCashReceipt",
		),
		ExpenseRecognitionPolicy: accountingcontrol.ExpenseRecognitionPolicyType(
			"OnCashDisbursement",
		),
		JournalPostingMode: accountingcontrol.JournalPostingModeType(
			"Automatic",
		),
		AutoPostSourceEvents: []accountingcontrol.JournalSourceEventType{
			"FactoringChargebackReversed",
		},
		ManualJournalEntryPolicy: accountingcontrol.ManualJournalEntryPolicy(
			"Disallow",
		),
		RequireManualJeApproval: true,
		JournalReversalPolicy: accountingcontrol.JournalReversalPolicyType(
			"NextOpenPeriod",
		),
		PeriodCloseMode: accountingcontrol.PeriodCloseModeType(
			"SystemScheduled",
		),
		RequirePeriodCloseApproval: true,
		LockedPeriodPostingPolicy: accountingcontrol.LockedPeriodPostingPolicy(
			"BlockSubledgerAllowManualJe",
		),
		ClosedPeriodPostingPolicy: accountingcontrol.ClosedPeriodPostingPolicy(
			"PostToNextOpen",
		),
		RequireReconciliationToClose: true,
		ReconciliationMode: accountingcontrol.ReconciliationModeType(
			"BlockPosting",
		),
		ReconciliationToleranceAmount:   "12.75",
		NotifyOnReconciliationException: true,
		CurrencyMode: accountingcontrol.CurrencyModeType(
			"MultiCurrency",
		),
		FunctionalCurrencyCode: "functionalCurrencyCode",
		ExchangeRateDatePolicy: accountingcontrol.ExchangeRateDatePolicy(
			"AccountingDate",
		),
		ExchangeRateOverridePolicy: accountingcontrol.ExchangeRateOverrideType(
			"Disallow",
		),
		DefaultRevenueAccountID:                 &v20Raw,
		DefaultCashAccountID:                    &v21Raw,
		DefaultUnappliedCashAccountID:           &v22Raw,
		DefaultExpenseAccountID:                 &v23Raw,
		DefaultArAccountID:                      &v24Raw,
		DefaultApAccountID:                      &v25Raw,
		DefaultTaxLiabilityAccountID:            &v26Raw,
		DefaultWriteOffAccountID:                &v27Raw,
		DefaultRetainedEarningsAccountID:        &v28Raw,
		RealizedFxGainAccountID:                 &v29Raw,
		RealizedFxLossAccountID:                 &v30Raw,
		DefaultDriverPayExpenseAccountID:        &v31Raw,
		DefaultPurchasedTransportationAccountID: &v32Raw,
		DefaultSettlementsPayableAccountID:      &v33Raw,
		DefaultDriverAdvanceAccountID:           &v34Raw,
		DefaultEscrowLiabilityAccountID:         &v35Raw,
		DefaultDriverReimbursementAccountID:     &v36Raw,
		DefaultEscrowInterestExpenseAccountID:   &v37Raw,
	})
	require.NoError(t, err)

	assert.Equal(t, existing.ID, entity.ID)
	assert.Equal(t, existing.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, existing.OrganizationID, entity.OrganizationID)
	assert.Equal(t, int64(100), entity.CreatedAt)
	assert.Equal(t, int64(3), entity.Version)
	assert.Equal(t, accountingcontrol.AccountingBasisType("Cash"), entity.AccountingBasis)
	assert.Equal(
		t,
		accountingcontrol.RevenueRecognitionPolicyType("OnCashReceipt"),
		entity.RevenueRecognitionPolicy,
	)
	assert.Equal(
		t,
		accountingcontrol.ExpenseRecognitionPolicyType("OnCashDisbursement"),
		entity.ExpenseRecognitionPolicy,
	)
	assert.Equal(
		t,
		accountingcontrol.JournalPostingModeType("Automatic"),
		entity.JournalPostingMode,
	)
	assert.Equal(
		t,
		[]accountingcontrol.JournalSourceEventType{"FactoringChargebackReversed"},
		entity.AutoPostSourceEvents,
	)
	assert.Equal(
		t,
		accountingcontrol.ManualJournalEntryPolicy("Disallow"),
		entity.ManualJournalEntryPolicy,
	)
	assert.True(t, entity.RequireManualJEApproval)
	assert.Equal(
		t,
		accountingcontrol.JournalReversalPolicyType("NextOpenPeriod"),
		entity.JournalReversalPolicy,
	)
	assert.Equal(
		t,
		accountingcontrol.PeriodCloseModeType("SystemScheduled"),
		entity.PeriodCloseMode,
	)
	assert.True(t, entity.RequirePeriodCloseApproval)
	assert.Equal(
		t,
		accountingcontrol.LockedPeriodPostingPolicy("BlockSubledgerAllowManualJe"),
		entity.LockedPeriodPostingPolicy,
	)
	assert.Equal(
		t,
		accountingcontrol.ClosedPeriodPostingPolicy("PostToNextOpen"),
		entity.ClosedPeriodPostingPolicy,
	)
	assert.True(t, entity.RequireReconciliationToClose)
	assert.Equal(
		t,
		accountingcontrol.ReconciliationModeType("BlockPosting"),
		entity.ReconciliationMode,
	)
	assert.Equal(t, "12.75", entity.ReconciliationToleranceAmount.String())
	assert.True(t, entity.NotifyOnReconciliationException)
	assert.Equal(t, accountingcontrol.CurrencyModeType("MultiCurrency"), entity.CurrencyMode)
	assert.Equal(t, "functionalCurrencyCode", entity.FunctionalCurrencyCode)
	assert.Equal(
		t,
		accountingcontrol.ExchangeRateDatePolicy("AccountingDate"),
		entity.ExchangeRateDatePolicy,
	)
	assert.Equal(
		t,
		accountingcontrol.ExchangeRateOverrideType("Disallow"),
		entity.ExchangeRateOverridePolicy,
	)
	assert.Equal(t, v20, entity.DefaultRevenueAccountID)
	assert.Equal(t, v21, entity.DefaultCashAccountID)
	assert.Equal(t, v22, entity.DefaultUnappliedCashAccountID)
	assert.Equal(t, v23, entity.DefaultExpenseAccountID)
	assert.Equal(t, v24, entity.DefaultARAccountID)
	assert.Equal(t, v25, entity.DefaultAPAccountID)
	assert.Equal(t, v26, entity.DefaultTaxLiabilityAccountID)
	assert.Equal(t, v27, entity.DefaultWriteOffAccountID)
	assert.Equal(t, v28, entity.DefaultRetainedEarningsAccountID)
	assert.Equal(t, v29, entity.RealizedFXGainAccountID)
	assert.Equal(t, v30, entity.RealizedFXLossAccountID)
	assert.Equal(t, v31, entity.DefaultDriverPayExpenseAccountID)
	assert.Equal(t, v32, entity.DefaultPurchasedTransportationAccountID)
	assert.Equal(t, v33, entity.DefaultSettlementsPayableAccountID)
	assert.Equal(t, v34, entity.DefaultDriverAdvanceAccountID)
	assert.Equal(t, v35, entity.DefaultEscrowLiabilityAccountID)
	assert.Equal(t, v36, entity.DefaultDriverReimbursementAccountID)
	assert.Equal(t, v37, entity.DefaultEscrowInterestExpenseAccountID)
}

func TestAccountingControlFromInput_NamesAMalformedField(t *testing.T) {
	t.Parallel()

	_, err := accountingControlFromInput(
		&accountingcontrol.AccountingControl{},
		&gqlmodel.UpdateAccountingControlInput{
			ReconciliationToleranceAmount: "twelve",
		},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.NotEmpty(t, multiErr.Errors)
	assert.Equal(t, "reconciliationToleranceAmount", multiErr.Errors[0].Field)
}

func TestAccountingControlResolver_RendersDecimalsExactly(t *testing.T) {
	t.Parallel()

	value, err := (&AccountingControlResolver{&Deps{}}).ReconciliationToleranceAmount(
		t.Context(),
		&accountingcontrol.AccountingControl{
			ReconciliationToleranceAmount: decimal.RequireFromString("0.725"),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "0.725", value)
}
