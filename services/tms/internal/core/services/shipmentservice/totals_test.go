package shipmentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/commodityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
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
	entity.Moves[0].Distance = new(float64(100))
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
	}

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req formulatemplateports.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			assert.Equal(t, entity.FormulaTemplateID, req.TemplateID)
			assert.Equal(t, entity.OrganizationID, req.TenantInfo.OrgID)
			assert.Equal(t, entity.BusinessUnitID, req.TenantInfo.BuID)
			return &formulatemplate.FormulaTemplate{
				Status:     formulatemplate.StatusActive,
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "freightChargeAmount * totalDistance",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(t, formula, accessorialRepo),
		eventService: noopShipmentEventService{},
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

	accessorialRepo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{})
	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(t, formula, accessorialRepo),
		eventService: noopShipmentEventService{},
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

	accessorialRepo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{})
	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(t, formula, accessorialRepo),
		eventService: noopShipmentEventService{},
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
	entity.Moves[0].Distance = new(float64(10))
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

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req formulatemplateports.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				Status:     formulatemplate.StatusActive,
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "freightChargeAmount * totalDistance",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(t, formula, accessorialRepo),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromInt(30).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.NewFromInt(13).Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromInt(43).Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_UsesAdditionalChargeOverridesForFormulaOtherChargeAmount(
	t *testing.T,
) {
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

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req formulatemplateports.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				Status:     formulatemplate.StatusActive,
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "freightChargeAmount + otherChargeAmount",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(t, formula, accessorialRepo),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	// decimal.Equal ignores exponent, so the comparison cannot be assert.Equal.
	// The message carries both sides, which "Should be true" does not.
	assert.True(t, decimal.RequireFromString("9302.44").Equal(resp.FreightChargeAmount),
		"freight: want 9302.44, got %s", resp.FreightChargeAmount)
	assert.True(t, decimal.NewFromInt(200).Equal(resp.OtherChargeAmount),
		"other: want 200, got %s", resp.OtherChargeAmount)
	assert.True(t, decimal.RequireFromString("9502.44").Equal(resp.TotalChargeAmount),
		"total: want 9502.44, got %s", resp.TotalChargeAmount)
}

func TestServiceCalculateTotals_PropagatesFormulaErrors(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req formulatemplateports.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				Status:     formulatemplate.StatusActive,
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "missingVariable +",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(t, formula, accessorialRepo),
		eventService: noopShipmentEventService{},
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
				LinearFeetPerUnit: new(float64(2)),
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
				LinearFeetPerUnit: new(1.5),
			},
		},
	}

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	accessorialRepo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req formulatemplateports.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				Status:     formulatemplate.StatusActive,
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "totalWeight + totalLinearFeet + (hasHazmat ? 100 : 0)",
			}, nil
		},
	})

	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo:  controlRepo,
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(t, formula, accessorialRepo),
		eventService: noopShipmentEventService{},
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

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()

	commodityRepo := commodityportsmock.NewMockCommodityRepository(t)
	commodityRepo.EXPECT().
		GetByIDs(mock.Anything, commodityports.GetCommoditiesByIDsRequest{
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
				LinearFeetPerUnit: new(1.25),
				HazardousMaterial: &hazardousmaterial.HazardousMaterial{ID: pulid.MustNew("hm_")},
			},
			{
				ID:                secondCommodityID,
				LinearFeetPerUnit: new(float64(2)),
			},
		}, nil).
		Once()
	accessorialRepo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	formula := newTestFormulaService(t, &stubFormulaTemplateRepository{
		getByIDFn: func(
			_ context.Context,
			req formulatemplateports.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				Status:     formulatemplate.StatusActive,
				ID:         req.TemplateID,
				SchemaID:   "shipment",
				Expression: "totalLinearFeet + (hasHazmat ? 25 : 0)",
			}, nil
		},
	})

	svc := &service{
		l:             zap.NewNop(),
		repo:          shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo:   controlRepo,
		commodityRepo: commodityRepo,
		validator:     newTestValidator(t),
		auditService:  auditportsmock.NewMockAuditService(t),
		commercial:    newTestCommercialCalculator(t, formula, accessorialRepo),
		eventService:  noopShipmentEventService{},
		coordinator:   newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromFloat(34).Equal(resp.FreightChargeAmount))
}

func newTestFormulaService(
	t *testing.T,
	repo formulatemplateports.FormulaTemplateRepository,
) *formula.Service {
	t.Helper()

	registry := schema.NewRegistry()
	registerShipmentSchema(t, registry)
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	eng, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	require.NoError(t, err)

	return formula.NewService(formula.ServiceParams{
		Logger:         zap.NewNop(),
		Registry:       registry,
		Engine:         eng,
		Resolver:       res,
		Repo:           repo,
		VersionRepo:    stubTotalsVersionRepo{},
		RateMatrixRepo: stubTotalsMatrixRepo{},
	})
}

type stubTotalsVersionRepo struct {
	formulatemplateports.FormulaTemplateVersionRepository
}

func (stubTotalsVersionRepo) GetEffectiveVersion(
	_ context.Context,
	_ *formulatemplateports.GetEffectiveVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return nil, nil
}

// ListScheduled is reached through the effective-version cache whenever a
// formula resolves. The embedded interface is nil, so leaving it unimplemented
// panics rather than failing: these tests have no scheduled versions.
func (stubTotalsVersionRepo) ListScheduled(
	_ context.Context,
	_ *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return nil, nil
}

type stubTotalsMatrixRepo struct {
	ratematrixports.RateMatrixRepository
}

func (stubTotalsMatrixRepo) GetLookupData(
	_ context.Context,
	_ *ratematrixports.GetRateMatrixLookupDataRequest,
) ([]*ratematrixports.RateMatrixLookupData, error) {
	return nil, nil
}

func registerShipmentSchema(t *testing.T, registry *schema.Registry) {
	t.Helper()

	const shipmentSchema = `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://trenova.com/schemas/formula/shipment.schema.json",
		"type": "object",
		"x-formula-context": {
			"category": "shipment",
			"entities": ["Shipment"]
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": ["Customer", "Moves.Stops", "Commodities.Commodity", "Commodities.Commodity.HazardousMaterial"]
		},
		"properties": {
			"weight": {
				"type": ["number", "null"],
				"x-source": { "field": "Weight", "nullable": true, "transform": "int64ToFloat64" }
			},
			"pieces": {
				"type": ["integer", "null"],
				"x-source": { "field": "Pieces", "nullable": true }
			},
			"freightChargeAmount": {
				"type": "number",
				"x-source": { "field": "FreightChargeAmount", "transform": "decimalToFloat64" }
			},
			"otherChargeAmount": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeOtherChargeAmount" }
			},
			"currentTotalCharge": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeCurrentTotalCharge" }
			},
			"ratingUnit": {
				"type": "integer",
				"x-source": { "field": "RatingUnit" }
			},
			"totalDistance": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeTotalDistance" }
			},
			"totalStops": {
				"type": "integer",
				"x-source": { "computed": true, "function": "computeTotalStops" }
			},
			"totalWeight": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeTotalWeight" }
			},
			"totalPieces": {
				"type": "integer",
				"x-source": { "computed": true, "function": "computeTotalPieces" }
			},
			"totalLinearFeet": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeTotalLinearFeet" }
			},
			"hasHazmat": {
				"type": "boolean",
				"x-source": { "computed": true, "function": "computeHasHazmat" }
			},
			"requiresTemperatureControl": {
				"type": "boolean",
				"x-source": { "computed": true, "function": "computeRequiresTemperatureControl" }
			},
			"temperatureDifferential": {
				"type": "number",
				"x-source": { "computed": true, "function": "computeTemperatureDifferential" }
			}
		}
	}`

	require.NoError(t, registry.Register("shipment", []byte(shipmentSchema)))
}

type stubFormulaTemplateRepository struct {
	getByIDFn func(context.Context, formulatemplateports.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error)
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
	req formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) GetByIDs(
	context.Context,
	formulatemplateports.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) FindByNames(
	context.Context,
	formulatemplateports.GetFormulaTemplatesByNamesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) List(
	context.Context,
	*formulatemplateports.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) ListConnection(
	context.Context,
	*formulatemplateports.ListFormulaTemplateConnectionRequest,
) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) BulkUpdateStatus(
	context.Context,
	*formulatemplateports.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) BulkDuplicate(
	context.Context,
	*formulatemplateports.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) CountUsages(
	context.Context,
	*formulatemplateports.GetTemplateUsageRequest,
) (*formulatemplateports.GetTemplateUsageResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) CountStatsByIDs(
	context.Context,
	*formulatemplateports.GetFormulaTemplateStatsRequest,
) (map[pulid.ID]formulatemplateports.TemplateStats, error) {
	return nil, errors.New("not implemented")
}

func (s *stubFormulaTemplateRepository) SelectOptions(
	context.Context,
	*formulatemplateports.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, errors.New("not implemented")
}

//go:fix inline
//go:fix inline
func ptrFloat64(value float64) *float64 {
	return new(value)
}

//go:fix inline
//go:fix inline
func ptrFloat(value float64) *float64 {
	return new(value)
}

var _ formulatemplateports.FormulaTemplateRepository = (*stubFormulaTemplateRepository)(nil)

func (stubTotalsMatrixRepo) GetLookupStamp(context.Context, pagination.TenantInfo) (string, error) {
	return "", nil
}
