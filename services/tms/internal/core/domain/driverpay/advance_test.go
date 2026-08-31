package driverpay

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validPayAdvance() PayAdvance {
	return PayAdvance{
		ID:             pulid.MustNew("padv_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		WorkerID:       pulid.MustNew("wrk_"),
		Status:         AdvanceStatusOutstanding,
		Source:         AdvanceSourceCash,
		IssuedDate:     1_700_000_000,
		AmountMinor:    50_000,
		CurrencyCode:   "USD",
	}
}

func TestPayAdvance_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(a *PayAdvance)
		wantErr bool
	}{
		{
			name:    "valid advance passes",
			mutate:  func(a *PayAdvance) {},
			wantErr: false,
		},
		{
			name:    "missing worker fails",
			mutate:  func(a *PayAdvance) { a.WorkerID = "" },
			wantErr: true,
		},
		{
			name:    "missing issued date fails",
			mutate:  func(a *PayAdvance) { a.IssuedDate = 0 },
			wantErr: true,
		},
		{
			name:    "invalid status fails",
			mutate:  func(a *PayAdvance) { a.Status = "Pending" },
			wantErr: true,
		},
		{
			name:    "invalid source fails",
			mutate:  func(a *PayAdvance) { a.Source = "Venmo" },
			wantErr: true,
		},
		{
			name:    "zero amount fails",
			mutate:  func(a *PayAdvance) { a.AmountMinor = 0 },
			wantErr: true,
		},
		{
			name:    "negative amount fails",
			mutate:  func(a *PayAdvance) { a.AmountMinor = -1 },
			wantErr: true,
		},
		{
			name:    "negative recovered amount fails",
			mutate:  func(a *PayAdvance) { a.RecoveredMinor = -1 },
			wantErr: true,
		},
		{
			name:    "negative written off amount fails",
			mutate:  func(a *PayAdvance) { a.WrittenOffMinor = -1 },
			wantErr: true,
		},
		{
			name: "recovered plus written off exceeding amount fails",
			mutate: func(a *PayAdvance) {
				a.RecoveredMinor = 30_000
				a.WrittenOffMinor = 30_000
			},
			wantErr: true,
		},
		{
			name: "recovered plus written off equal to amount passes",
			mutate: func(a *PayAdvance) {
				a.Status = AdvanceStatusRecovered
				a.RecoveredMinor = 30_000
				a.WrittenOffMinor = 20_000
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entity := validPayAdvance()
			tt.mutate(&entity)
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestPayAdvance_OutstandingMinor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		amount     int64
		recovered  int64
		writtenOff int64
		want       int64
	}{
		{name: "nothing recovered", amount: 50_000, want: 50_000},
		{name: "partially recovered", amount: 50_000, recovered: 20_000, want: 30_000},
		{
			name:       "recovered and written off",
			amount:     50_000,
			recovered:  20_000,
			writtenOff: 10_000,
			want:       20_000,
		},
		{name: "fully recovered", amount: 50_000, recovered: 50_000, want: 0},
		{name: "over recovered clamps at zero", amount: 50_000, recovered: 60_000, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			advance := PayAdvance{
				AmountMinor:     tt.amount,
				RecoveredMinor:  tt.recovered,
				WrittenOffMinor: tt.writtenOff,
			}
			assert.Equal(t, tt.want, advance.OutstandingMinor())
		})
	}
}

func TestPayAdvance_SyncStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		amount     int64
		recovered  int64
		writtenOff int64
		want       AdvanceStatus
	}{
		{name: "untouched advance is outstanding", amount: 50_000, want: AdvanceStatusOutstanding},
		{
			name:      "partial recovery",
			amount:    50_000,
			recovered: 20_000,
			want:      AdvanceStatusPartiallyRecovered,
		},
		{
			name:      "full recovery",
			amount:    50_000,
			recovered: 50_000,
			want:      AdvanceStatusRecovered,
		},
		{
			name:       "full write off",
			amount:     50_000,
			writtenOff: 50_000,
			want:       AdvanceStatusWrittenOff,
		},
		{
			name:       "recovery then write off of the remainder",
			amount:     50_000,
			recovered:  30_000,
			writtenOff: 20_000,
			want:       AdvanceStatusWrittenOff,
		},
		{
			name:       "partial write off still partially recovered",
			amount:     50_000,
			recovered:  10_000,
			writtenOff: 10_000,
			want:       AdvanceStatusPartiallyRecovered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			advance := PayAdvance{
				AmountMinor:     tt.amount,
				RecoveredMinor:  tt.recovered,
				WrittenOffMinor: tt.writtenOff,
			}
			advance.SyncStatus()
			assert.Equal(t, tt.want, advance.Status)
		})
	}
}
