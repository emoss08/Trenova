package formula_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/pkg/rules"
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

func TestCheck_PassesTheTenantsReferencedTablesSoAnUnknownTableIsRefused(t *testing.T) {
	t.Parallel()

	svc := setupServiceWithMatrixData(t, append(freightClassMatrix(), zoneWeightMatrix()))

	check := func(expression string) *rules.FormulaCheck {
		checked, err := svc.Check(t.Context(), &formula.CheckRequest{
			SchemaID:   "shipment",
			TenantInfo: testTenant(),
			Expect:     rules.ExpectAmount,
			Sources:    []formula.CheckSource{{Scope: formula.MainScope, Source: expression}},
		})
		require.NoError(t, err)
		return checked[0].Check
	}

	assert.True(t, check(`sum(lookup("class_rates", "85"), 25)`).Accepted())

	unknown := check(`sum(lookup("fsc", 1), 25)`).FirstError()
	require.NotNil(t, unknown)
	assert.Equal(t, "rate-table", unknown.Code)
	assert.Contains(t, unknown.Message.String(), "fsc")

	wrongShape := check(`lookup("zone_weight", "Z1")`).FirstError()
	require.NotNil(t, wrongShape, "a two-axis table read by lookup is refused")
	assert.Equal(t, "rate-table", wrongShape.Code)

	assert.True(t, check(`lookup2("zone_weight", "Z1", 1200)`).Accepted())
}
