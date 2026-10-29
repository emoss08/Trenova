package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/operationstest"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/testutil/mocks/commodityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/hazmatsegregationruleportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHazmatSegregationValidationAndServiceStayAligned(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.Commodities = alignmentCommodities()

	controlRepo := alignmentControlRepo(t, entity)
	commodityRepo := commodityportsmock.NewMockCommodityRepository(t)
	commodityRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req commodityports.GetCommoditiesByIDsRequest) bool {
			return req.TenantInfo.OrgID == entity.OrganizationID &&
				req.TenantInfo.BuID == entity.BusinessUnitID &&
				len(req.CommodityIDs) == 2
		})).
		Return([]*commodity.Commodity{
			alignmentCommodity(
				entity.Commodities[0].CommodityID,
				"Explosive",
				hazardousmaterial.HazardousClass1,
				pulid.MustNew("hm_"),
			),
			alignmentCommodity(
				entity.Commodities[1].CommodityID,
				"Paint",
				hazardousmaterial.HazardousClass3,
				pulid.MustNew("hm_"),
			),
		}, nil).
		Twice()
	ruleRepo := hazmatsegregationruleportsmock.NewMockHazmatSegregationRuleRepository(t)
	activeRules := []*hazmatsegregationrule.HazmatSegregationRule{
		{
			ID:              pulid.MustNew("hsr_"),
			Name:            "No explosives with flammables",
			Status:          domaintypes.StatusActive,
			ClassA:          hazardousmaterial.HazardousClass1,
			ClassB:          hazardousmaterial.HazardousClass3,
			SegregationType: hazmatsegregationrule.SegregationTypeProhibited,
			OrganizationID:  entity.OrganizationID,
			BusinessUnitID:  entity.BusinessUnitID,
		},
	}
	ruleRepo.EXPECT().
		ListActiveByTenant(mock.Anything, pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}).
		Return(activeRules, nil).
		Once()
	ruleRepo.EXPECT().
		ListActiveByTenant(mock.Anything, pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}).
		Return(activeRules, nil).
		Once()

	v := shipmentvalidator.New(shipmentvalidator.Params{
		ControlRepo:               controlRepo,
		CustomerRepo:              operationstest.NewCustomerRepository(t),
		CommodityRepo:             commodityRepo,
		HazmatSegregationRuleRepo: ruleRepo,
		ShipmentRepo:              shipmentportsmock.NewMockShipmentRepository(t),
	})

	svc := assembled(&service{
		l:              zap.NewNop(),
		controlRepo:    controlRepo,
		commodityRepo:  commodityRepo,
		hazmatRuleRepo: ruleRepo,
		validator:      v,
		eventService:   noopShipmentEventService{},
		coordinator:    newStateCoordinator(),
	})

	validationErr := v.ValidateCreate(t.Context(), entity)
	require.NotNil(t, validationErr)

	serviceErr := svc.loading.CheckHazmatSegregation(
		t.Context(),
		&shipmentports.CheckHazmatSegregationRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			CommodityIDs: []pulid.ID{
				entity.Commodities[0].CommodityID,
				entity.Commodities[1].CommodityID,
			},
		},
	)
	require.Error(t, serviceErr)

	var serviceMultiErr *errortypes.MultiError
	require.ErrorAs(t, serviceErr, &serviceMultiErr)
	require.Equal(t, validationErr.Error(), serviceMultiErr.Error())
	shipmentservicetest.AssertErrorField(t, serviceMultiErr, "commodities[0].commodityId")
	shipmentservicetest.AssertErrorField(t, serviceMultiErr, "commodities[1].commodityId")
}

func alignmentControlRepo(
	t *testing.T,
	entity *shipment.Shipment,
) *tenantportsmock.MockShipmentControlRepository {
	t.Helper()

	repo := tenantportsmock.NewMockShipmentControlRepository(t)
	repo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			CheckHazmatSegregation: true,
			AllowMoveRemovals:      true,
			MaxShipmentWeightLimit: 1000000,
		}, nil)

	return repo
}

func alignmentCommodities() []*shipment.ShipmentCommodity {
	return []*shipment.ShipmentCommodity{
		{CommodityID: pulid.MustNew("com_"), Weight: 100, Pieces: 10},
		{CommodityID: pulid.MustNew("com_"), Weight: 120, Pieces: 12},
	}
}

func alignmentCommodity(
	id pulid.ID,
	name string,
	class hazardousmaterial.HazardousClass,
	hazmatID pulid.ID,
) *commodity.Commodity {
	return &commodity.Commodity{
		ID:                  id,
		Name:                name,
		HazardousMaterialID: hazmatID,
		HazardousMaterial: &hazardousmaterial.HazardousMaterial{
			ID:    hazmatID,
			Name:  name + " Hazmat",
			Class: class,
		},
	}
}
