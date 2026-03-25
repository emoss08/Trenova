package shipmentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func TestGetByID_ExpandedDetailsPreloadsCommodityHazmat(t *testing.T) {
	t.Parallel()

	db, dbMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		dbMock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	moveRepo := mocks.NewMockShipmentMoveRepository(t)

	repo := &repository{
		db:             postgres.NewTestConnection(bunDB),
		l:              zap.NewNop(),
		moveRepository: moveRepo,
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")
	commodityID := pulid.MustNew("com_")
	shipmentCommodityID := pulid.MustNew("sc_")
	hazmatID := pulid.MustNew("hm_")

	dbMock.ExpectQuery(`(?s)SELECT .* FROM "shipments" AS "sp".*WHERE .*sp\.organization_id = .*sp\.business_unit_id = .*sp\.id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "service_type_id", "customer_id", "formula_template_id", "status", "pro_number", "bol", "rating_unit",
		}).AddRow(
			shipmentID, buID, orgID, pulid.MustNew("svc_"), pulid.MustNew("cus_"), pulid.MustNew("fmt_"), shipment.StatusNew, "PRO-1", "BOL-1", 1,
		))
	dbMock.ExpectQuery(`(?s)SELECT .* FROM "additional_charges" AS "ac".*"shipment_id" IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "accessorial_charge_id", "method", "amount", "unit",
		}))
	dbMock.ExpectQuery(`(?s)SELECT .* FROM "shipment_commodities" AS "sc".*"shipment_id" IN`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "business_unit_id", "organization_id", "shipment_id", "commodity_id", "weight", "pieces",
			"commodity__id", "commodity__business_unit_id", "commodity__organization_id", "commodity__hazardous_material_id", "commodity__name", "commodity__description",
			"commodity__hazardous_material__id", "commodity__hazardous_material__business_unit_id", "commodity__hazardous_material__organization_id",
			"commodity__hazardous_material__name", "commodity__hazardous_material__description", "commodity__hazardous_material__class", "commodity__hazardous_material__packing_group",
		}).AddRow(
			shipmentCommodityID, buID, orgID, shipmentID, commodityID, 100, 10,
			commodityID, buID, orgID, hazmatID, "Paint", "Flammable paint",
			hazmatID, buID, orgID, "UN1263", "Flammable liquid", "Class3", "II",
		))
	moveRepo.EXPECT().
		GetMovesByShipmentID(mock.Anything, mock.MatchedBy(func(req *repositories.GetMovesByShipmentIDRequest) bool {
			return req != nil &&
				req.ShipmentID == shipmentID &&
				req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.ExpandMoveDetails
		})).
		Return([]*shipment.ShipmentMove{
			{
				ID:             pulid.MustNew("sm_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
				ShipmentID:     shipmentID,
				Status:         shipment.MoveStatusAssigned,
			},
			{
				ID:             pulid.MustNew("sm_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
				ShipmentID:     shipmentID,
				Status:         shipment.MoveStatusNew,
			},
		}, nil).
		Once()

	entity, err := repo.GetByID(t.Context(), &repositories.GetShipmentByIDRequest{
		ID: shipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})

	require.NoError(t, err)
	require.Len(t, entity.Moves, 2)
	require.Len(t, entity.Commodities, 1)
	require.NotNil(t, entity.Commodities[0].Commodity)
	require.NotNil(t, entity.Commodities[0].Commodity.HazardousMaterial)
	assert.Equal(t, hazmatID, entity.Commodities[0].Commodity.HazardousMaterial.ID)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}
