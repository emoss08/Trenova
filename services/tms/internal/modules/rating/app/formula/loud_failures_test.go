package formula_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataContract_StatusComparesAsText(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	entity := ratedShipment()
	entity.Status = shipment.StatusInTransit

	cases := map[string]string{
		`status == "InTransit" ? 1 : 0`:                    "1",
		`status in ["New", "InTransit"] ? 1 : 0`:           "1",
		`status != "Canceled" ? 1 : 0`:                     "1",
		`hasPrefix(status, "In") ? 1 : 0`:                  "1",
		`status in ["ReadyToInvoice", "Invoiced"] ? 1 : 0`: "0",
	}
	for expression, want := range cases {
		assertAmount(t, want, evaluate(t, svc, entity, expression), expression)
	}
}

func TestShipmentSchema_StatusEnumIsTheDomainsStatuses(t *testing.T) {
	t.Parallel()

	registry, err := formula.NewSchemaRegistry()
	require.NoError(t, err)
	definition, ok := registry.Get("shipment")
	require.True(t, ok)

	domain := make([]string, 0, 10)
	for _, stage := range shipment.Stages() {
		for _, status := range stage.Statuses() {
			domain = append(domain, string(status))
		}
	}
	slices.Sort(domain)

	schema := slices.Clone(definition.Properties["status"].Enum)
	slices.Sort(schema)

	assert.Equal(t, domain, schema)
}

func TestDataContract_TemperatureDifferentialIsZeroOnADryVan(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	expression := `requiresTemperatureControl ? temperatureDifferential * 2 : 0`

	assertAmount(t, "0", evaluate(t, svc, ratedShipment(), expression), "dry van")

	low, high := int16(34), int16(40)
	reefer := ratedShipment()
	reefer.TemperatureMin = &low
	reefer.TemperatureMax = &high
	assertAmount(t, "12", evaluate(t, svc, reefer, expression), "reefer")
}

const builtinOverrideExpression = `sum(lookup("fsc", fuelPrice) * linehaul, stopFee, 25)`

func TestValidateLookupTables_SeesTablesInsideTheEnginesOwnFunctions(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	err := svc.ValidateLookupTables(t.Context(), builtinOverrideExpression, testTenant())
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Contains(t, err.Error(), "fsc")

	known := setupServiceWithMatrixData(t, freightClassMatrix())
	err = known.ValidateLookupTables(
		t.Context(),
		`sum(lookup("class_rates", "85"), 25)`,
		testTenant(),
	)
	require.NoError(t, err)
}

func TestSplitStatements_ParsesTheEnginesOwnFunctions(t *testing.T) {
	t.Parallel()

	statements, err := formula.SplitStatements(
		"let fuel = sum(fuelPrice, stopFee, 25);\nfuel * 2",
	)
	require.NoError(t, err)
	require.Len(t, statements, 2)
	assert.Equal(t, "fuel", statements[0].Name)
}
