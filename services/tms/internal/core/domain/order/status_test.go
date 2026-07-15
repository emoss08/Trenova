package order_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/stretchr/testify/assert"
)

func TestDerive(t *testing.T) {
	tests := []struct {
		name string
		legs []shipment.Status
		want order.Status
	}{
		{
			name: "no legs is draft",
			legs: nil,
			want: order.StatusDraft,
		},
		{
			name: "all new is confirmed",
			legs: []shipment.Status{shipment.StatusNew, shipment.StatusNew},
			want: order.StatusConfirmed,
		},
		{
			name: "any in transit is in progress",
			legs: []shipment.Status{shipment.StatusNew, shipment.StatusInTransit},
			want: order.StatusInProgress,
		},
		{
			name: "mixed completed and in transit is in progress",
			legs: []shipment.Status{shipment.StatusCompleted, shipment.StatusInTransit},
			want: order.StatusInProgress,
		},
		{
			name: "all delivered variants is completed",
			legs: []shipment.Status{
				shipment.StatusReadyToInvoice,
				shipment.StatusCompleted,
				shipment.StatusInvoiced,
			},
			want: order.StatusCompleted,
		},
		{
			name: "all invoiced is billed",
			legs: []shipment.Status{shipment.StatusInvoiced, shipment.StatusInvoiced},
			want: order.StatusBilled,
		},
		{
			name: "partial invoiced stays completed",
			legs: []shipment.Status{shipment.StatusInvoiced, shipment.StatusReadyToInvoice},
			want: order.StatusCompleted,
		},
		{
			name: "all canceled is canceled",
			legs: []shipment.Status{shipment.StatusCanceled, shipment.StatusCanceled},
			want: order.StatusCanceled,
		},
		{
			name: "canceled leg is excluded from active progress",
			legs: []shipment.Status{shipment.StatusCanceled, shipment.StatusInvoiced},
			want: order.StatusBilled,
		},
		{
			name: "one canceled among moving legs stays in progress",
			legs: []shipment.Status{shipment.StatusCanceled, shipment.StatusInTransit},
			want: order.StatusInProgress,
		},
		{
			name: "assigned legs are in progress",
			legs: []shipment.Status{shipment.StatusAssigned, shipment.StatusNew},
			want: order.StatusInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, order.Derive(tt.legs))
		})
	}
}
