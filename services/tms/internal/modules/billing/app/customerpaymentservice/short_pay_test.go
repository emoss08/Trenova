package customerpaymentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func shortPaysUnderThreshold(
	t *testing.T,
	orgID any,
) *tenantportsmock.MockInvoiceAdjustmentControlRepository {
	t.Helper()
	repo := tenantportsmock.NewMockInvoiceAdjustmentControlRepository(t)
	repo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(
		&invoiceadjustmentcontrol.InvoiceAdjustmentControl{
			WriteOffApprovalPolicy:    invoiceadjustmentcontrol.WriteOffApprovalPolicyRequireApprovalAboveThreshold,
			WriteOffApprovalThreshold: decimal.RequireFromString("1000000"),
		}, nil,
	).Maybe()
	return repo
}

func TestShortPaysFollowTheWriteOffPolicy(t *testing.T) {
	t.Parallel()

	threshold := func(amount string) *invoiceadjustmentcontrol.InvoiceAdjustmentControl {
		return &invoiceadjustmentcontrol.InvoiceAdjustmentControl{
			WriteOffApprovalPolicy:    invoiceadjustmentcontrol.WriteOffApprovalPolicyRequireApprovalAboveThreshold,
			WriteOffApprovalThreshold: decimal.RequireFromString(amount),
		}
	}
	tests := map[string]struct {
		control *invoiceadjustmentcontrol.InvoiceAdjustmentControl
		minor   int64
		want    shortPayOutcome
	}{
		"write-offs disallowed": {
			control: &invoiceadjustmentcontrol.InvoiceAdjustmentControl{
				WriteOffApprovalPolicy: invoiceadjustmentcontrol.WriteOffApprovalPolicyDisallow,
			},
			minor: 1,
			want:  shortPayRefused,
		},
		"always needs approval": {
			control: &invoiceadjustmentcontrol.InvoiceAdjustmentControl{
				WriteOffApprovalPolicy: invoiceadjustmentcontrol.WriteOffApprovalPolicyAlwaysRequireApproval,
			},
			minor: 1,
			want:  shortPayNeedsApproval,
		},
		"under the threshold": {control: threshold("25.00"), minor: 2499, want: shortPayAllowed},
		"at the threshold": {
			control: threshold("25.00"),
			minor:   2500,
			want:    shortPayNeedsApproval,
		},
		"an unknown policy": {
			control: &invoiceadjustmentcontrol.InvoiceAdjustmentControl{},
			minor:   1,
			want:    shortPayRefused,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, shortPayPolicy(tc.control, tc.minor))
		})
	}
}

func TestCheckShortPaysReportsTheApplicationThatNeedsApproval(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	repo := tenantportsmock.NewMockInvoiceAdjustmentControlRepository(t)
	repo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(
		&invoiceadjustmentcontrol.InvoiceAdjustmentControl{
			WriteOffApprovalPolicy: invoiceadjustmentcontrol.WriteOffApprovalPolicyAlwaysRequireApproval,
		}, nil,
	).Once()
	v := NewValidator(ValidatorParams{AdjustmentCtrlRepo: repo})
	me := errortypes.NewMultiError()

	err := v.checkShortPays(t.Context(), orgID, []*customerpayment.Application{
		{AppliedAmountMinor: 100},
		{AppliedAmountMinor: 1, ShortPayAmountMinor: 99_999},
	}, me)

	require.NoError(t, err)
	require.True(t, me.HasErrors())
	assert.Contains(t, me.Error(), "applications[1].shortPayAmountMinor")
}

func TestCheckShortPaysReadsNoPolicyWithoutAShortPay(t *testing.T) {
	t.Parallel()

	repo := tenantportsmock.NewMockInvoiceAdjustmentControlRepository(t)
	v := NewValidator(ValidatorParams{AdjustmentCtrlRepo: repo})
	me := errortypes.NewMultiError()

	err := v.checkShortPays(t.Context(), pulid.MustNew("org_"), []*customerpayment.Application{
		{AppliedAmountMinor: 100},
		nil,
	}, me)

	require.NoError(t, err)
	assert.False(t, me.HasErrors())
}
