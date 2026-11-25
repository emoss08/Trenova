package shipmentpredicate

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func optionSQL(t *testing.T, opts *shipmentports.ShipmentOptions) (string, []any) {
	t.Helper()

	preds := OptionPredicates(opts)
	require.NotEmpty(t, preds)

	return renderShipments(t, preds...)
}

func TestAttributePredicates_NoneSetAddsNothing(t *testing.T) {
	t.Parallel()

	assert.Empty(t, OptionPredicates(&shipmentports.ShipmentOptions{
		EquipmentTypeText: "  ",
		CommodityText:     " ",
		StopPlaces:        []shipmentports.ShipmentStopPlace{{}},
	}))
}

func TestStopPlace_WindowAndPlaceMeetOnOneStop(t *testing.T) {
	t.Parallel()

	sql, args := optionSQL(t, &shipmentports.ShipmentOptions{
		StopPlaces: []shipmentports.ShipmentStopPlace{{
			Cities:     []string{"Los Angeles"},
			Types:      []shipmentvocab.StopType{shipmentvocab.StopTypePickup},
			WindowFrom: 1_700_000_000,
			WindowTo:   1_700_086_400,
		}},
	})

	assert.Equal(t, 1, strings.Count(sql, `FROM "stops"`), "the city, type and window read one stop")
	assert.Contains(t, sql, `"locations"`)
	assert.Contains(t, sql, `"scheduled_window_start" <= $`)
	assert.Contains(t, sql, `"scheduled_window_start") >= $`)
	assert.Contains(t, args, []string{"los angeles"})
	assert.Contains(t, args, int64(1_700_086_400))
	assert.Contains(t, args, int64(1_700_000_000))
}

func TestStopPlace_WindowWithoutATypeReadsPickupsAndDeliveries(t *testing.T) {
	t.Parallel()

	sql, _ := optionSQL(t, &shipmentports.ShipmentOptions{
		StopPlaces: []shipmentports.ShipmentStopPlace{{WindowTo: 1_700_086_400}},
	})

	assert.Contains(t, sql, `'Pickup'`)
	assert.Contains(t, sql, `'Delivery'`)
	assert.Contains(t, sql, `"scheduled_window_start" > $`)
	assert.NotContains(t, sql, `"locations"`)
}

func TestAttributePredicates_EquipmentMatchesTrailerOrTractorType(t *testing.T) {
	t.Parallel()

	sql, args := optionSQL(t, &shipmentports.ShipmentOptions{EquipmentTypeText: "Re_efer"})
	assert.Contains(t, sql, `"trailer_type_id" IN (SELECT`)
	assert.Contains(t, sql, `"tractor_type_id" IN (SELECT`)
	assert.Contains(t, sql, `"equipment_types"`)
	assert.Contains(t, args, `%Re\_efer%`)

	typeID := pulid.MustNew("et_")
	sql, args = optionSQL(t, &shipmentports.ShipmentOptions{
		EquipmentTypeIDs: []pulid.ID{typeID},
	})
	assert.NotContains(t, sql, `"equipment_types"`)
	assert.Contains(t, args, []string{typeID.String()})
}

func TestAttributePredicates_UnassignedReadsDriversAndCarriers(t *testing.T) {
	t.Parallel()

	unassigned := true
	sql, _ := optionSQL(t, &shipmentports.ShipmentOptions{Unassigned: &unassigned})
	assert.Contains(t, sql, `"has_active_assignment" IS FALSE`)
	assert.Contains(t, sql, `NOT (EXISTS`)
	assert.Contains(t, sql, `"carrier_assignments"`)

	assigned := false
	sql, _ = optionSQL(t, &shipmentports.ShipmentOptions{Unassigned: &assigned})
	assert.Contains(t, sql, `"has_active_assignment" IS TRUE OR EXISTS`)
	assert.NotContains(t, sql, `NOT (EXISTS`)
}

func TestAttributePredicates_StatusAgeAndBillingReadiness(t *testing.T) {
	t.Parallel()

	notReady := false
	sql, args := optionSQL(t, &shipmentports.ShipmentOptions{
		InStatusSince: 1_700_000_000,
		BillingReady:  &notReady,
	})
	assert.Contains(t, sql, `"status_changed_at" <= $`)
	assert.Contains(t, sql, `NOT (`)
	assert.Contains(t, sql, `"billing_transfer_status"`)
	assert.Contains(t, args, int64(1_700_000_000))

	ready := true
	sql, _ = optionSQL(t, &shipmentports.ShipmentOptions{BillingReady: &ready})
	assert.NotContains(t, sql, `NOT (`)
	assert.Contains(t, sql, `"billing_transfer_status"`)
}

func TestAttributePredicates_CommodityTextMatchesWhatTheLoadCarries(t *testing.T) {
	t.Parallel()

	sql, args := optionSQL(t, &shipmentports.ShipmentOptions{CommodityText: "frozen"})

	assert.Contains(t, sql, `"shipment_commodities"`)
	assert.Contains(t, sql, `"commodities"`)
	assert.Contains(t, args, "%frozen%")
}
