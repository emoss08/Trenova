package shipmentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func previewCreateService(
	t *testing.T,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
) (*service, *shipmentportsmock.MockShipmentRepository) {
	t.Helper()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().Get(mock.Anything, tenantports.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}).Return(control, nil).Once()
	formula := formulatemplateportsmock.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.NewFromInt(1_250)}, nil).
		Twice()

	return &service{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
		validator:   newTestValidator(t),
		commercial: newTestCommercialCalculator(t,
			formula,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		coordinator: newStateCoordinator(),
	}, repo
}

func previewActor(entity *shipment.Shipment) *permissionports.RequestActor {
	userID := pulid.MustNew("usr_")

	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}
}

func TestPreviewCreate_PricesAndValidatesWithoutSaving(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FormulaTemplateID = pulid.MustNew("fmt_")
	svc, _ := previewCreateService(t, entity, &tenant.ShipmentControl{})

	plan, err := svc.PreviewCreate(t.Context(), entity, previewActor(entity))
	require.NoError(t, err)

	assert.Same(t, entity, plan.Shipment)
	assert.True(t, plan.Shipment.ID.IsNil())
	assert.Equal(t, shipment.StatusNew, plan.Shipment.Status)
	require.True(t, plan.Shipment.FreightChargeAmount.Valid)
	assert.True(t, decimal.NewFromInt(1_250).Equal(plan.Shipment.FreightChargeAmount.Decimal))
	require.NotNil(t, plan.Rating)
	assert.True(t, decimal.NewFromInt(1_250).Equal(plan.Rating.Amount))
	assert.Equal(t, "USD", plan.Rating.Currency)
	assert.NotEmpty(t, plan.Rating.Explanation)
}

func TestPreviewCreate_RefusesADuplicateBOLTheWayCreateDoes(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.BOL = "BOL-DUP"
	entity.FormulaTemplateID = pulid.MustNew("fmt_")
	svc, repo := previewCreateService(t, entity, &tenant.ShipmentControl{
		CheckForDuplicateBOLs: true,
	})
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.Anything).
		Return([]*shipmentports.DuplicateBOLResult{
			{ID: pulid.MustNew("shp_"), ProNumber: "PRO-500"},
		}, nil).
		Once()

	plan, err := svc.PreviewCreate(t.Context(), entity, previewActor(entity))

	require.Nil(t, plan)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "bol")
}

func TestPreviewCreate_RequiresAShipment(t *testing.T) {
	t.Parallel()

	svc := &service{l: zap.NewNop()}

	_, err := svc.PreviewCreate(t.Context(), nil, nil)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
}

func unpricedCreateService(
	t *testing.T,
	entity *shipment.Shipment,
	disposition tenant.UnratedShipmentDisposition,
) *service {
	t.Helper()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().Get(mock.Anything, mock.Anything).
		Return(&tenant.ShipmentControl{}, nil).
		Once()

	return &service{
		l:           zap.NewNop(),
		repo:        shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo: controlRepo,
		validator:   rateCoverageValidator(t, billingControlWith(t, disposition)),
		commercial: newTestCommercialCalculator(t,
			formulatemplateportsmock.NewMockFormulaCalculator(t),
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		coordinator: newStateCoordinator(),
	}
}

func TestPreviewCreate_RefusesAShipmentNothingPrices(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FormulaTemplateID = pulid.Nil
	svc := unpricedCreateService(t, entity, "")

	plan, err := svc.PreviewCreate(t.Context(), entity, previewActor(entity))

	require.Nil(t, plan)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "formulaTemplateId")
}

func TestPreviewCreate_ZeroAndFlagNamesTheUnratedShipment(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FormulaTemplateID = pulid.Nil
	svc := unpricedCreateService(t, entity, tenant.UnratedShipmentDispositionZeroAndFlag)

	plan, err := svc.PreviewCreate(t.Context(), entity, previewActor(entity))

	require.NoError(t, err)
	require.Len(t, plan.Advisories, 1)
	assert.Equal(t, rateCoverageRuleKey, plan.Advisories[0].RuleKey)
	assert.Contains(t, plan.Advisories[0].Error(), "priced at zero")
}
