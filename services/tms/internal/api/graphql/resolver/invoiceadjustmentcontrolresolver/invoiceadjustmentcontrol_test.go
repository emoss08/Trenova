package invoiceadjustmentcontrolresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceAdjustmentControlResolvers_CheckTheInvoiceAdjustmentControlPermission(
	t *testing.T,
) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"invoiceAdjustmentControl",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).InvoiceAdjustmentControl(ctx)
				return err
			},
		},
		{
			"updateInvoiceAdjustmentControl",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateInvoiceAdjustmentControl(
					ctx,
					gqlmodel.UpdateInvoiceAdjustmentControlInput{},
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
				permission.ResourceInvoiceAdjustmentControl,
				call.operation,
			)
		})
	}
}

func TestInvoiceAdjustmentControlFromInput_MapsEveryFieldOntoTheStoredControl(t *testing.T) {
	t.Parallel()

	existing := &invoiceadjustmentcontrol.InvoiceAdjustmentControl{
		ID:             pulid.MustNew("x_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Version:        3,
		CreatedAt:      100,
		UpdatedAt:      200,
	}

	entity, err := invoiceAdjustmentControlFromInput(
		existing,
		&gqlmodel.UpdateInvoiceAdjustmentControlInput{
			Version: 3,
			PartiallyPaidInvoiceAdjustmentPolicy: invoiceadjustmentcontrol.AdjustmentEligibilityPolicy(
				"AllowWithoutApproval",
			),
			PaidInvoiceAdjustmentPolicy: invoiceadjustmentcontrol.AdjustmentEligibilityPolicy(
				"AllowWithoutApproval",
			),
			DisputedInvoiceAdjustmentPolicy: invoiceadjustmentcontrol.AdjustmentEligibilityPolicy(
				"AllowWithoutApproval",
			),
			AdjustmentAccountingDatePolicy: invoiceadjustmentcontrol.AdjustmentAccountingDatePolicy(
				"AlwaysNextOpen",
			),
			ClosedPeriodAdjustmentPolicy: invoiceadjustmentcontrol.ClosedPeriodAdjustmentPolicy(
				"PostInNextOpenPeriodWithApproval",
			),
			AdjustmentReasonRequirement: invoiceadjustmentcontrol.RequirementPolicy(
				"Required",
			),
			StandardAdjustmentApprovalPolicy: invoiceadjustmentcontrol.ApprovalPolicy(
				"AmountThreshold",
			),
			StandardAdjustmentApprovalThreshold: "12.75",
			WriteOffApprovalPolicy: invoiceadjustmentcontrol.WriteOffApprovalPolicy(
				"RequireApprovalAboveThreshold",
			),
			WriteOffApprovalThreshold:      "12.75",
			RequireSeparateApprover:        true,
			RerateVarianceTolerancePercent: "12.75",
			ReplacementInvoiceReviewPolicy: invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicy(
				"AlwaysRequireReview",
			),
			CustomerCreditBalancePolicy: invoiceadjustmentcontrol.CustomerCreditBalancePolicy(
				"AllowUnappliedCredit",
			),
			OverCreditPolicy: invoiceadjustmentcontrol.OverCreditPolicy(
				"AllowWithApproval",
			),
			SupersededInvoiceVisibilityPolicy: invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicy(
				"ShowCurrentAndSupersededExternally",
			),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, existing.ID, entity.ID)
	assert.Equal(t, existing.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, existing.OrganizationID, entity.OrganizationID)
	assert.Equal(t, int64(100), entity.CreatedAt)
	assert.Equal(t, int64(3), entity.Version)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.AdjustmentEligibilityPolicy("AllowWithoutApproval"),
		entity.PartiallyPaidInvoiceAdjustmentPolicy,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.AdjustmentEligibilityPolicy("AllowWithoutApproval"),
		entity.PaidInvoiceAdjustmentPolicy,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.AdjustmentEligibilityPolicy("AllowWithoutApproval"),
		entity.DisputedInvoiceAdjustmentPolicy,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.AdjustmentAccountingDatePolicy("AlwaysNextOpen"),
		entity.AdjustmentAccountingDatePolicy,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.ClosedPeriodAdjustmentPolicy("PostInNextOpenPeriodWithApproval"),
		entity.ClosedPeriodAdjustmentPolicy,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.RequirementPolicy("Required"),
		entity.AdjustmentReasonRequirement,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.ApprovalPolicy("AmountThreshold"),
		entity.StandardAdjustmentApprovalPolicy,
	)
	assert.Equal(t, "12.75", entity.StandardAdjustmentApprovalThreshold.String())
	assert.Equal(
		t,
		invoiceadjustmentcontrol.WriteOffApprovalPolicy("RequireApprovalAboveThreshold"),
		entity.WriteOffApprovalPolicy,
	)
	assert.Equal(t, "12.75", entity.WriteOffApprovalThreshold.String())
	assert.True(t, entity.RequireSeparateApprover)
	assert.Equal(t, "12.75", entity.RerateVarianceTolerancePercent.String())
	assert.Equal(
		t,
		invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicy("AlwaysRequireReview"),
		entity.ReplacementInvoiceReviewPolicy,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.CustomerCreditBalancePolicy("AllowUnappliedCredit"),
		entity.CustomerCreditBalancePolicy,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.OverCreditPolicy("AllowWithApproval"),
		entity.OverCreditPolicy,
	)
	assert.Equal(
		t,
		invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicy(
			"ShowCurrentAndSupersededExternally",
		),
		entity.SupersededInvoiceVisibilityPolicy,
	)
}

func TestInvoiceAdjustmentControlFromInput_NamesAMalformedField(t *testing.T) {
	t.Parallel()

	_, err := invoiceAdjustmentControlFromInput(
		&invoiceadjustmentcontrol.InvoiceAdjustmentControl{},
		&gqlmodel.UpdateInvoiceAdjustmentControlInput{
			StandardAdjustmentApprovalThreshold: "twelve",
			WriteOffApprovalThreshold:           "1",
			RerateVarianceTolerancePercent:      "1",
		},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.NotEmpty(t, multiErr.Errors)
	assert.Equal(t, "standardAdjustmentApprovalThreshold", multiErr.Errors[0].Field)
}

func TestInvoiceAdjustmentControlResolver_RendersDecimalsExactly(t *testing.T) {
	t.Parallel()

	value, err := (&InvoiceAdjustmentControlResolver{&Deps{}}).StandardAdjustmentApprovalThreshold(
		t.Context(),
		&invoiceadjustmentcontrol.InvoiceAdjustmentControl{
			StandardAdjustmentApprovalThreshold: decimal.RequireFromString("0.725"),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "0.725", value)
}
