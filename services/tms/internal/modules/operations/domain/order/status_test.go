package order_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/stretchr/testify/assert"
)

func TestDerive(t *testing.T) {
	tests := []struct {
		name string
		legs []shipmentvocab.Status
		want order.Status
	}{
		{
			name: "no legs is draft",
			legs: nil,
			want: order.StatusDraft,
		},
		{
			name: "all new is confirmed",
			legs: []shipmentvocab.Status{shipmentvocab.StatusNew, shipmentvocab.StatusNew},
			want: order.StatusConfirmed,
		},
		{
			name: "any in transit is in progress",
			legs: []shipmentvocab.Status{shipmentvocab.StatusNew, shipmentvocab.StatusInTransit},
			want: order.StatusInProgress,
		},
		{
			name: "mixed completed and in transit is in progress",
			legs: []shipmentvocab.Status{
				shipmentvocab.StatusCompleted,
				shipmentvocab.StatusInTransit,
			},
			want: order.StatusInProgress,
		},
		{
			name: "all delivered variants is completed",
			legs: []shipmentvocab.Status{
				shipmentvocab.StatusReadyToInvoice,
				shipmentvocab.StatusCompleted,
				shipmentvocab.StatusInvoiced,
			},
			want: order.StatusCompleted,
		},
		{
			name: "all invoiced is billed",
			legs: []shipmentvocab.Status{
				shipmentvocab.StatusInvoiced,
				shipmentvocab.StatusInvoiced,
			},
			want: order.StatusBilled,
		},
		{
			name: "partial invoiced stays completed",
			legs: []shipmentvocab.Status{
				shipmentvocab.StatusInvoiced,
				shipmentvocab.StatusReadyToInvoice,
			},
			want: order.StatusCompleted,
		},
		{
			name: "all canceled is canceled",
			legs: []shipmentvocab.Status{
				shipmentvocab.StatusCanceled,
				shipmentvocab.StatusCanceled,
			},
			want: order.StatusCanceled,
		},
		{
			name: "canceled leg is excluded from active progress",
			legs: []shipmentvocab.Status{
				shipmentvocab.StatusCanceled,
				shipmentvocab.StatusInvoiced,
			},
			want: order.StatusBilled,
		},
		{
			name: "one canceled among moving legs stays in progress",
			legs: []shipmentvocab.Status{
				shipmentvocab.StatusCanceled,
				shipmentvocab.StatusInTransit,
			},
			want: order.StatusInProgress,
		},
		{
			name: "assigned legs are in progress",
			legs: []shipmentvocab.Status{shipmentvocab.StatusAssigned, shipmentvocab.StatusNew},
			want: order.StatusInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, order.Derive(tt.legs))
		})
	}
}
