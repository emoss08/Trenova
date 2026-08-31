package driverpay

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validPayCode() PayCode {
	return PayCode{
		ID:             pulid.MustNew("payc_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Direction:      PayCodeDirectionEarning,
		Code:           "SAFETY",
		Name:           "Safety Bonus",
		Taxable:        true,
	}
}

func TestPayCode_Validate(t *testing.T) {
	t.Parallel()

	invalidDefault := int64(0)
	validDefault := int64(2_500)

	tests := []struct {
		name    string
		mutate  func(p *PayCode)
		wantErr bool
	}{
		{
			name:    "valid pay code passes",
			mutate:  func(p *PayCode) {},
			wantErr: false,
		},
		{
			name:    "code with digits dashes and underscores passes",
			mutate:  func(p *PayCode) { p.Code = "9TRK-LEASE_2" },
			wantErr: false,
		},
		{
			name:    "missing code fails",
			mutate:  func(p *PayCode) { p.Code = "" },
			wantErr: true,
		},
		{
			name:    "lowercase code fails",
			mutate:  func(p *PayCode) { p.Code = "safety" },
			wantErr: true,
		},
		{
			name:    "code starting with dash fails",
			mutate:  func(p *PayCode) { p.Code = "-SAFETY" },
			wantErr: true,
		},
		{
			name:    "code with spaces fails",
			mutate:  func(p *PayCode) { p.Code = "SAFETY BONUS" },
			wantErr: true,
		},
		{
			name:    "code longer than 20 characters fails",
			mutate:  func(p *PayCode) { p.Code = "ABCDEFGHIJKLMNOPQRSTU" },
			wantErr: true,
		},
		{
			name:    "missing name fails",
			mutate:  func(p *PayCode) { p.Name = "" },
			wantErr: true,
		},
		{
			name:    "invalid direction fails",
			mutate:  func(p *PayCode) { p.Direction = "Both" },
			wantErr: true,
		},
		{
			name:    "non active or inactive status fails",
			mutate:  func(p *PayCode) { p.Status = "Archived" },
			wantErr: true,
		},
		{
			name:    "inactive status passes",
			mutate:  func(p *PayCode) { p.Status = domaintypes.StatusInactive },
			wantErr: false,
		},
		{
			name:    "non positive default amount fails",
			mutate:  func(p *PayCode) { p.DefaultAmountMinor = &invalidDefault },
			wantErr: true,
		},
		{
			name:    "positive default amount passes",
			mutate:  func(p *PayCode) { p.DefaultAmountMinor = &validDefault },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entity := validPayCode()
			tt.mutate(&entity)
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestPayCode_LineIsReimbursement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direction PayCodeDirection
		taxable   bool
		want      bool
	}{
		{
			name:      "non taxable earning is a reimbursement",
			direction: PayCodeDirectionEarning,
			taxable:   false,
			want:      true,
		},
		{
			name:      "taxable earning is not",
			direction: PayCodeDirectionEarning,
			taxable:   true,
			want:      false,
		},
		{
			name:      "non taxable deduction is not",
			direction: PayCodeDirectionDeduction,
			taxable:   false,
			want:      false,
		},
		{
			name:      "taxable deduction is not",
			direction: PayCodeDirectionDeduction,
			taxable:   true,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code := PayCode{Direction: tt.direction, Taxable: tt.taxable}
			assert.Equal(t, tt.want, code.LineIsReimbursement())
		})
	}
}

func TestSystemPayCodes(t *testing.T) {
	t.Parallel()

	codes := SystemPayCodes()
	assert.NotEmpty(t, codes)

	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		assert.True(
			t,
			code.Direction.IsValid(),
			"direction for %s must be valid",
			code.Code,
		)
		assert.Regexp(t, payCodePattern, code.Code)
		assert.NotEmpty(t, code.Name)

		key := code.Direction.String() + ":" + code.Code
		_, dup := seen[key]
		assert.False(t, dup, "duplicate system pay code %s", key)
		seen[key] = struct{}{}
	}
}
