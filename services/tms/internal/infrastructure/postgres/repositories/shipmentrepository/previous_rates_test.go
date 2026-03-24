package shipmentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPreviousRates_ReturnsPricingSummaries(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mock.ExpectQuery(`WITH "origin_shipments" AS .*origin_stop\.location_id = .*"destination_shipments" AS .*delivery_stop\.location_id = .*SELECT COUNT\(\*\) FROM shipments AS sp.*sp\.organization_id = .*sp\.business_unit_id = .*sp\.shipment_type_id = .*sp\.service_type_id = .*sp\.status = .*sp\.id IN \(SELECT shipment_id FROM origin_shipments\).*sp\.id IN \(SELECT shipment_id FROM destination_shipments\).*sp\.customer_id = .*sp\.id != .*`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`WITH "origin_shipments" AS .*SELECT sp\.id AS shipment_id, sp\.pro_number, sp\.customer_id, sp\.service_type_id, sp\.shipment_type_id, sp\.formula_template_id, sp\.freight_charge_amount, sp\.other_charge_amount, sp\.total_charge_amount, sp\.rating_unit, sp\.pieces, sp\.weight, sp\.created_at FROM shipments AS sp.*ORDER BY "sp"\."created_at" DESC LIMIT 50`).
		WillReturnRows(sqlmock.NewRows([]string{
			"shipment_id", "pro_number", "customer_id", "service_type_id", "shipment_type_id", "formula_template_id",
			"freight_charge_amount", "other_charge_amount", "total_charge_amount", "rating_unit", "pieces", "weight", "created_at",
		}).AddRow(
			pulid.MustNew("shp_"),
			"PRO-100",
			pulid.MustNew("cus_"),
			pulid.MustNew("svc_"),
			pulid.MustNew("sht_"),
			pulid.MustNew("fmt_"),
			"125.00",
			"10.00",
			"135.00",
			1,
			5,
			1200,
			1710000000,
		))

	customerID := pulid.MustNew("cus_")
	excludeID := pulid.MustNew("shp_")
	result, err := repo.GetPreviousRates(t.Context(), &repositories.GetPreviousRatesRequest{
		TenantInfo:            pagination.TenantInfo{OrgID: orgID, BuID: buID},
		OriginLocationID:      pulid.MustNew("loc_"),
		DestinationLocationID: pulid.MustNew("loc_"),
		ShipmentTypeID:        pulid.MustNew("sht_"),
		ServiceTypeID:         pulid.MustNew("svc_"),
		CustomerID:            &customerID,
		ExcludeShipmentID:     &excludeID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "PRO-100", result.Items[0].ProNumber)
	assert.Equal(t, 1, result.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}
