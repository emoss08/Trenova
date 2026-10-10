package ratequoteservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/stretchr/testify/assert"
)

func TestHasMoveWithoutDistance(t *testing.T) {
	t.Parallel()

	miles := 812.0
	zero := 0.0
	tests := []struct {
		name  string
		moves []*shipment.ShipmentMove
		want  bool
	}{
		{name: "every move has miles", moves: []*shipment.ShipmentMove{{Distance: &miles}}},
		{
			name:  "a move never routed",
			moves: []*shipment.ShipmentMove{{Distance: &miles}, {}},
			want:  true,
		},
		{name: "a move at zero miles", moves: []*shipment.ShipmentMove{{Distance: &zero}}, want: true},
		{name: "no moves"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, hasMoveWithoutDistance(&shipment.Shipment{Moves: tt.moves}))
		})
	}
}
