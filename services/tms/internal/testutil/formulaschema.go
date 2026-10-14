package testutil

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/stretchr/testify/require"
)

func RegisterShipmentFormulaSchema(t *testing.T, registry *schema.Registry) {
	t.Helper()

	shipmentSchema, err := formula.ShipmentSchemaJSON()
	require.NoError(t, err)
	require.NoError(t, registry.Register("shipment", shipmentSchema))
}
