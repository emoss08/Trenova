package invoice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/stretchr/testify/assert"
)

func TestIsAllowedTransition(t *testing.T) {
	tests := []struct {
		name string
		from invoice.Status
		to   invoice.Status
		want bool
	}{
		{name: "draft posts", from: invoice.StatusDraft, to: invoice.StatusPosted, want: true},
		{name: "draft voids directly", from: invoice.StatusDraft, to: invoice.StatusVoided, want: true},
		{name: "posted voids", from: invoice.StatusPosted, to: invoice.StatusVoided, want: true},
		{name: "posted never returns to draft", from: invoice.StatusPosted, to: invoice.StatusDraft, want: false},
		{name: "voided is terminal", from: invoice.StatusVoided, to: invoice.StatusPosted, want: false},
		{name: "voided stays voided", from: invoice.StatusVoided, to: invoice.StatusDraft, want: false},
		{name: "same status is a no-op", from: invoice.StatusPosted, to: invoice.StatusPosted, want: true},
		{name: "unknown status goes nowhere", from: invoice.Status("Bogus"), to: invoice.StatusPosted, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, invoice.IsAllowedTransition(tt.from, tt.to))
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	assert.True(t, invoice.IsTerminalStatus(invoice.StatusVoided))
	assert.False(t, invoice.IsTerminalStatus(invoice.StatusDraft))
	assert.False(t, invoice.IsTerminalStatus(invoice.StatusPosted))
}

func TestVoidDispositionIsValid(t *testing.T) {
	assert.True(t, invoice.VoidDispositionRebill.IsValid())
	assert.True(t, invoice.VoidDispositionDoNotRebill.IsValid())
	assert.False(t, invoice.VoidDisposition("").IsValid())
	assert.False(t, invoice.VoidDisposition("Later").IsValid())
}

func TestDisputeResolutionRequiresAdjustment(t *testing.T) {
	assert.True(t, invoice.DisputeResolutionCreditIssued.RequiresAdjustment())
	assert.True(t, invoice.DisputeResolutionWrittenOff.RequiresAdjustment())
	assert.False(t, invoice.DisputeResolutionInvoiceUpheld.RequiresAdjustment())
	assert.False(t, invoice.DisputeResolutionRebilled.RequiresAdjustment())
	assert.False(t, invoice.DisputeResolutionCustomerWithdrew.RequiresAdjustment())
}
