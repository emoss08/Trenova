package shipmentservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceCalculateTotals_UsesFormulaTemplateAndNestedAdditionalCharges(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromFloat(2.5))
	entity.Moves[0].Distance = ptrFloat64(100)
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
	}

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := mocks.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req repositories.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			assert.Equal(t, entity.FormulaTemplateID, req.TemplateID)
			assert.Equal(t, entity.OrganizationID, req.TenantInfo.OrgID)
			assert.Equal(t, entity.BusinessUnitID, req.TenantInfo.BuID)
			return &formulatemplate.FormulaTemplate{
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "freightChargeAmount * totalDistance",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(formula, accessorialRepo),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromInt(250).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.NewFromInt(10).Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromInt(260).Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_RejectsMissingFormulaTemplateID(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FormulaTemplateID = pulid.Nil

	accessorialRepo := mocks.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{})
	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(formula, accessorialRepo),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.Nil(t, resp)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "formulaTemplateId")
}

func TestServiceCalculateTotals_RejectsNilShipment(t *testing.T) {
	t.Parallel()

	accessorialRepo := mocks.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{})
	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(formula, accessorialRepo),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), nil, pulid.MustNew("usr_"))

	require.Nil(t, resp)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "shipment")
}

func TestServiceCalculateTotals_CalculatesPerUnitAndPercentageCharges(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromFloat(3))
	entity.Moves[0].Distance = ptrFloat64(10)
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodPerUnit,
			Amount:              decimal.NewFromInt(5),
			Unit:                2,
		},
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodPercentage,
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
	}

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := mocks.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req repositories.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "freightChargeAmount * totalDistance",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(formula, accessorialRepo),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromInt(30).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.NewFromInt(13).Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromInt(43).Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_UsesAdditionalChargeOverridesForFormulaOtherChargeAmount(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("9102.44"))
	entity.OtherChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(50))
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(200),
			Unit:                1,
		},
	}

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := mocks.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req repositories.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "freightChargeAmount + otherChargeAmount",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(formula, accessorialRepo),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.RequireFromString("9302.44").Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.NewFromInt(200).Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.RequireFromString("9502.44").Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_PropagatesFormulaErrors(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := mocks.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req repositories.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "missingVariable +",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(formula, accessorialRepo),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.Nil(t, resp)
	require.Error(t, err)
}

func TestServiceCalculateTotals_UsesCommodityRollupsInFormula(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Weight = nil
	entity.Pieces = nil
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			CommodityID: pulid.MustNew("com_"),
			Pieces:      5,
			Weight:      1000,
			Commodity: &commodity.Commodity{
				LinearFeetPerUnit: ptrFloat(2),
				HazardousMaterial: &hazardousmaterial.HazardousMaterial{
					ID: pulid.MustNew("hm_"),
				},
			},
		},
		{
			CommodityID: pulid.MustNew("com_"),
			Pieces:      3,
			Weight:      700,
			Commodity: &commodity.Commodity{
				LinearFeetPerUnit: ptrFloat(1.5),
			},
		},
	}

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := mocks.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req repositories.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "totalWeight + totalLinearFeet + (hasHazmat ? 100 : 0)",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(formula, accessorialRepo),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromFloat(1814.5).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.Zero.Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromFloat(1814.5).Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_HydratesCommodityDetailsBeforeFormula(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Weight = nil
	entity.Pieces = nil
	firstCommodityID := pulid.MustNew("com_")
	secondCommodityID := pulid.MustNew("com_")
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			CommodityID: firstCommodityID,
			Pieces:      4,
			Weight:      800,
		},
		{
			CommodityID: secondCommodityID,
			Pieces:      2,
			Weight:      200,
		},
	}

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()

	commodityRepo := mocks.NewMockCommodityRepository(t)
	commodityRepo.EXPECT().
		GetByIDs(mock.Anything, repositories.GetCommoditiesByIDsRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			CommodityIDs: []pulid.ID{
				firstCommodityID,
				secondCommodityID,
			},
		}).
		Return([]*commodity.Commodity{
			{
				ID:                firstCommodityID,
				LinearFeetPerUnit: ptrFloat(1.25),
				HazardousMaterial: &hazardousmaterial.HazardousMaterial{ID: pulid.MustNew("hm_")},
			},
			{
				ID:                secondCommodityID,
				LinearFeetPerUnit: ptrFloat(2),
			},
		}, nil).
		Once()
	accessorialRepo := mocks.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req repositories.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "totalLinearFeet + (hasHazmat ? 25 : 0)",
			}, nil
		},
	})

	svc := &service{
		l:             zap.NewNop(),
		repo:          mocks.NewMockShipmentRepository(t),
		controlRepo:   controlRepo,
		commodityRepo: commodityRepo,
		validator:     NewTestValidator(t),
		auditService:  mocks.NewMockAuditService(t),
		commercial:    newTestCommercialCalculator(formula, accessorialRepo),
		coordinator:   newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromFloat(34).Equal(resp.FreightChargeAmount))
}

func newTestFormulaService(
	t *testing.T,
	repo repositories.FormulaTemplateRepository,
) *formula.Service {
	t.Helper()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	eng := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})

	return formula.NewService(formula.ServiceParams{
		Logger:   zap.NewNop(),
		Registry: registry,
		Engine:   eng,
		Resolver: res,
		Repo:     repo,
	})
}

type stubFormulaTemplateRepository struct {
	getByIDFn func(context.Context, repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error)
}

func (s *stubFormulaTemplateRepository) Create(
	context.Context,
	*formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) Update(
	context.Context,
	*formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) GetByID(
	ctx context.Context,
	req repositories.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) GetByIDs(
	context.Context,
	repositories.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) List(
	context.Context,
	*repositories.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) BulkUpdateStatus(
	context.Context,
	*repositories.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) BulkDuplicate(
	context.Context,
	*repositories.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) CountUsages(
	context.Context,
	*repositories.GetTemplateUsageRequest,
) (*repositories.GetTemplateUsageResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) SelectOptions(
	context.Context,
	*repositories.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, errors.New("not implemented")
}

//go:fix inline
func ptrFloat64(value float64) *float64 {
	return &value
}

//go:fix inline
func ptrFloat(value float64) *float64 {
	return &value
}

var _ repositories.FormulaTemplateRepository = (*stubFormulaTemplateRepository)(nil)
