package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidatorValidateCreate_SkipsHazmatSegregationWhenDisabled(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Commodities = hazardousShipmentCommodities()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{CheckHazmatSegregation: false, AllowMoveRemovals: true, MaxShipmentWeightLimit: 1000000}, nil).
		Maybe()

	v := &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			mocks.NewMockCommodityRepository(t),
			mocks.NewMockHazmatSegregationRuleRepository(t),
			mocks.NewMockShipmentRepository(t),
		).Build(),
	}

	require.Nil(t, v.ValidateCreate(t.Context(), entity))
}

func TestValidatorValidateCreate_RejectsProhibitedHazmatPair(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Commodities = hazardousShipmentCommodities()

	controlRepo := mockHazmatControlRepo(t, entity)
	commodityRepo := mockHazmatCommodityRepo(t, entity)
	ruleRepo := mocks.NewMockHazmatSegregationRuleRepository(t)
	ruleRepo.EXPECT().
		ListActiveByTenant(mock.Anything, pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}).
		Return([]*hazmatsegregationrule.HazmatSegregationRule{
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
		}, nil).
		Once()

	v := &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			commodityRepo,
			ruleRepo,
			mocks.NewMockShipmentRepository(t),
		).Build(),
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "commodities[0].commodityId")
	assertErrorField(t, multiErr, "commodities[1].commodityId")
}

func TestValidatorValidateCreate_RejectsDistanceRuleMatch(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Commodities = hazardousShipmentCommodities()

	controlRepo := mockHazmatControlRepo(t, entity)
	commodityRepo := mockHazmatCommodityRepo(t, entity)
	ruleRepo := mocks.NewMockHazmatSegregationRuleRepository(t)
	ruleRepo.EXPECT().
		ListActiveByTenant(mock.Anything, pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}).
		Return([]*hazmatsegregationrule.HazmatSegregationRule{
			{
				ID:              pulid.MustNew("hsr_"),
				Name:            "Keep oxidizers apart",
				Status:          domaintypes.StatusActive,
				ClassA:          hazardousmaterial.HazardousClass1,
				ClassB:          hazardousmaterial.HazardousClass3,
				SegregationType: hazmatsegregationrule.SegregationTypeDistance,
				MinimumDistance: float64Ptr(10),
				DistanceUnit:    "FT",
				OrganizationID:  entity.OrganizationID,
				BusinessUnitID:  entity.BusinessUnitID,
			},
		}, nil).
		Once()

	v := &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			commodityRepo,
			ruleRepo,
			mocks.NewMockShipmentRepository(t),
		).Build(),
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "commodities[0].commodityId")
	assertErrorField(t, multiErr, "commodities[1].commodityId")
}

func TestValidatorValidateCreate_MatchesSpecificHazmatMaterialsUnordered(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	items := hazardousShipmentCommodities()
	entity.Commodities = []*shipment.ShipmentCommodity{items[1], items[0]}

	controlRepo := mockHazmatControlRepo(t, entity)
	commodityRepo := mocks.NewMockCommodityRepository(t)
	leftMaterialID := pulid.MustNew("hm_")
	rightMaterialID := pulid.MustNew("hm_")
	commodityRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.GetCommoditiesByIDsRequest) bool {
			return req.TenantInfo.OrgID == entity.OrganizationID &&
				req.TenantInfo.BuID == entity.BusinessUnitID &&
				len(req.CommodityIDs) == 2
		})).
		Return([]*commodity.Commodity{
			hazardousCommodity(
				items[0].CommodityID,
				"Paint",
				hazardousmaterial.HazardousClass3,
				rightMaterialID,
			),
			hazardousCommodity(
				items[1].CommodityID,
				"Explosive",
				hazardousmaterial.HazardousClass1,
				leftMaterialID,
			),
		}, nil).
		Once()

	ruleRepo := mocks.NewMockHazmatSegregationRuleRepository(t)
	ruleRepo.EXPECT().
		ListActiveByTenant(mock.Anything, pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}).
		Return([]*hazmatsegregationrule.HazmatSegregationRule{
			{
				ID:              pulid.MustNew("hsr_"),
				Name:            "Specific pair rule",
				Status:          domaintypes.StatusActive,
				ClassA:          hazardousmaterial.HazardousClass1,
				ClassB:          hazardousmaterial.HazardousClass3,
				HazmatAID:       &leftMaterialID,
				HazmatBID:       &rightMaterialID,
				SegregationType: hazmatsegregationrule.SegregationTypeBarrier,
				OrganizationID:  entity.OrganizationID,
				BusinessUnitID:  entity.BusinessUnitID,
			},
		}, nil).
		Once()

	v := &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			commodityRepo,
			ruleRepo,
			mocks.NewMockShipmentRepository(t),
		).Build(),
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "commodities[0].commodityId")
	assertErrorField(t, multiErr, "commodities[1].commodityId")
}

func TestValidatorValidateCreate_IgnoresInactiveOrUnmatchedRules(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Commodities = hazardousShipmentCommodities()

	controlRepo := mockHazmatControlRepo(t, entity)
	commodityRepo := mockHazmatCommodityRepo(t, entity)
	ruleRepo := mocks.NewMockHazmatSegregationRuleRepository(t)
	ruleRepo.EXPECT().
		ListActiveByTenant(mock.Anything, pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}).
		Return([]*hazmatsegregationrule.HazmatSegregationRule{
			{
				ID:              pulid.MustNew("hsr_"),
				Name:            "Oxidizer mismatch",
				Status:          domaintypes.StatusActive,
				ClassA:          hazardousmaterial.HazardousClass5And1,
				ClassB:          hazardousmaterial.HazardousClass8,
				SegregationType: hazmatsegregationrule.SegregationTypeProhibited,
				OrganizationID:  entity.OrganizationID,
				BusinessUnitID:  entity.BusinessUnitID,
			},
		}, nil).
		Once()

	v := &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			commodityRepo,
			ruleRepo,
			mocks.NewMockShipmentRepository(t),
		).Build(),
	}

	require.Nil(t, v.ValidateCreate(t.Context(), entity))
}

func mockHazmatControlRepo(
	t *testing.T,
	entity *shipment.Shipment,
) *mocks.MockShipmentControlRepository {
	t.Helper()

	repo := mocks.NewMockShipmentControlRepository(t)
	repo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{CheckHazmatSegregation: true, AllowMoveRemovals: true, MaxShipmentWeightLimit: 1000000}, nil).
		Maybe()

	return repo
}

func mockHazmatCommodityRepo(
	t *testing.T,
	entity *shipment.Shipment,
) *mocks.MockCommodityRepository {
	t.Helper()

	repo := mocks.NewMockCommodityRepository(t)
	repo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.GetCommoditiesByIDsRequest) bool {
			return req.TenantInfo.OrgID == entity.OrganizationID &&
				req.TenantInfo.BuID == entity.BusinessUnitID &&
				len(req.CommodityIDs) == 2
		})).
		Return([]*commodity.Commodity{
			hazardousCommodity(
				entity.Commodities[0].CommodityID,
				"Explosive",
				hazardousmaterial.HazardousClass1,
				pulid.MustNew("hm_"),
			),
			hazardousCommodity(
				entity.Commodities[1].CommodityID,
				"Paint",
				hazardousmaterial.HazardousClass3,
				pulid.MustNew("hm_"),
			),
		}, nil).
		Once()

	return repo
}

func hazardousShipmentCommodities() []*shipment.ShipmentCommodity {
	return []*shipment.ShipmentCommodity{
		{
			CommodityID: pulid.MustNew("com_"),
			Weight:      100,
			Pieces:      10,
		},
		{
			CommodityID: pulid.MustNew("com_"),
			Weight:      120,
			Pieces:      12,
		},
	}
}

func hazardousCommodity(
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

//go:fix inline
func float64Ptr(v float64) *float64 {
	return &v
}
