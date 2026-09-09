package fuelpurchase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCardStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from fuelpurchase.CardStatus
		to   fuelpurchase.CardStatus
		want bool
	}{
		{fuelpurchase.CardStatusActive, fuelpurchase.CardStatusSuspended, true},
		{fuelpurchase.CardStatusActive, fuelpurchase.CardStatusCancelled, true},
		{fuelpurchase.CardStatusActive, fuelpurchase.CardStatusActive, false},
		{fuelpurchase.CardStatusSuspended, fuelpurchase.CardStatusActive, true},
		{fuelpurchase.CardStatusSuspended, fuelpurchase.CardStatusCancelled, true},
		{fuelpurchase.CardStatusSuspended, fuelpurchase.CardStatusSuspended, false},
		{fuelpurchase.CardStatusCancelled, fuelpurchase.CardStatusActive, false},
		{fuelpurchase.CardStatusCancelled, fuelpurchase.CardStatusSuspended, false},
		{fuelpurchase.CardStatusCancelled, fuelpurchase.CardStatusCancelled, false},
		{fuelpurchase.CardStatus("Lost"), fuelpurchase.CardStatusActive, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.from.CanTransitionTo(tt.to))
		})
	}

	assert.True(t, fuelpurchase.CardStatusCancelled.IsTerminal())
	assert.False(t, fuelpurchase.CardStatusSuspended.IsTerminal())
}

func TestImportStatus_Guards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status     fuelpurchase.ImportStatus
		canStage   bool
		canCommit  bool
		canDiscard bool
		terminal   bool
	}{
		{fuelpurchase.ImportStatusPending, true, false, true, false},
		{fuelpurchase.ImportStatusParsed, true, true, true, false},
		{fuelpurchase.ImportStatusFailed, true, false, true, false},
		{fuelpurchase.ImportStatusCommitted, false, false, false, true},
		{fuelpurchase.ImportStatusDiscarded, false, false, false, true},
		{fuelpurchase.ImportStatus("Bogus"), false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.canStage, tt.status.CanStage(), "CanStage")
			assert.Equal(t, tt.canCommit, tt.status.CanCommit(), "CanCommit")
			assert.Equal(t, tt.canDiscard, tt.status.CanDiscard(), "CanDiscard")
			assert.Equal(t, tt.terminal, tt.status.IsTerminal(), "IsTerminal")
		})
	}
}

func TestImportRowStatus_WillCommit(t *testing.T) {
	t.Parallel()

	assert.True(t, fuelpurchase.ImportRowStatusNew.WillCommit())
	for _, status := range []fuelpurchase.ImportRowStatus{
		fuelpurchase.ImportRowStatusDuplicateInFile,
		fuelpurchase.ImportRowStatusAlreadyImported,
		fuelpurchase.ImportRowStatusError,
		fuelpurchase.ImportRowStatusCommitted,
		fuelpurchase.ImportRowStatusSkipped,
	} {
		assert.False(t, status.WillCommit(), "%s must not commit", status)
		assert.True(t, status.IsValid())
	}
}

func TestQuantityUnit_ToGallons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		unit     fuelpurchase.QuantityUnit
		quantity string
		want     string
	}{
		{"100 litres", fuelpurchase.QuantityUnitLitre, "100", "26.417"},
		{"one gallon of litres", fuelpurchase.QuantityUnitLitre, "3.785411784", "1.000"},
		{"one litre", fuelpurchase.QuantityUnitLitre, "1", "0.264"},
		{"half-up litres", fuelpurchase.QuantityUnitLitre, "1000", "264.172"},
		{"gallons pass through", fuelpurchase.QuantityUnitGallon, "10", "10.000"},
		{"gallons rounded to 3dp", fuelpurchase.QuantityUnitGallon, "12.3456", "12.346"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.unit.ToGallons(decimal.RequireFromString(tt.quantity))
			assert.Equal(t, tt.want, got.StringFixed(3))
		})
	}
}

func TestEnums_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, fuelpurchase.CardProviderComdata.IsValid())
	assert.True(t, fuelpurchase.CardProviderEFS.IsValid())
	assert.True(t, fuelpurchase.CardProviderWEX.IsValid())
	assert.True(t, fuelpurchase.CardProviderOther.IsValid())
	assert.False(t, fuelpurchase.CardProvider("Fleetcor").IsValid())

	assert.True(t, fuelpurchase.QuantityUnitGallon.IsValid())
	assert.True(t, fuelpurchase.QuantityUnitLitre.IsValid())
	assert.False(t, fuelpurchase.QuantityUnit("Barrel").IsValid())

	assert.True(t, fuelpurchase.PurchaseSourceManual.IsValid())
	assert.True(t, fuelpurchase.PurchaseSourceCardImport.IsValid())
	assert.False(t, fuelpurchase.PurchaseSource("API").IsValid())

	assert.True(t, fuelpurchase.SourceFormatCSV.IsValid())
	assert.True(t, fuelpurchase.SourceFormatXLSX.IsValid())
	assert.False(t, fuelpurchase.SourceFormat("PDF").IsValid())

	for _, provider := range []fuelpurchase.CardProvider{
		fuelpurchase.CardProviderComdata,
		fuelpurchase.CardProviderEFS,
		fuelpurchase.CardProviderWEX,
		fuelpurchase.CardProviderOther,
	} {
		assert.NotEmpty(t, provider.Label())
	}
}
