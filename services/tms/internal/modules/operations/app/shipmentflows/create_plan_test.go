package shipmentflows

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	control *shipmentcontrol.ShipmentControl,
) (*rig, *shipmentportsmock.MockShipmentRepository) {
	t.Helper()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().Get(mock.Anything, tenantports.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}).Return(control, nil).Once()
	freight := decimal.NewFromInt(1_250)

	return assembled(&rig{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
		validator:   newServiceValidator(t),
		commercial: newTestCommercialCalculator(t,
			freight,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		coordinator: newStateCoordinator(),
	}), repo
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

	entity := shipmentservicetest.ValidShipment()
	entity.FormulaTemplateID = pulid.MustNew("fmt_")
	svc, _ := previewCreateService(t, entity, &shipmentcontrol.ShipmentControl{})

	plan, err := svc.writer.PreviewCreate(t.Context(), entity, previewActor(entity))
	require.NoError(t, err)

	assert.Same(t, entity, plan.Shipment)
	assert.True(t, plan.Shipment.ID.IsNil())
	assert.Equal(t, shipmentvocab.StatusNew, plan.Shipment.Status)
	require.True(t, plan.Shipment.FreightChargeAmount.Valid)
	assert.True(t, decimal.NewFromInt(1_250).Equal(plan.Shipment.FreightChargeAmount.Decimal))
	require.NotNil(t, plan.Rating)
	assert.True(t, decimal.NewFromInt(1_250).Equal(plan.Rating.Amount))
	assert.Equal(t, "USD", plan.Rating.Currency)
	assert.NotEmpty(t, plan.Rating.Explanation)
}

func TestPreviewCreate_RefusesADuplicateBOLTheWayCreateDoes(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.BOL = "BOL-DUP"
	entity.FormulaTemplateID = pulid.MustNew("fmt_")
	svc, repo := previewCreateService(t, entity, &shipmentcontrol.ShipmentControl{
		CheckForDuplicateBOLs: true,
	})
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.Anything).
		Return([]*shipmentports.DuplicateBOLResult{
			{ID: pulid.MustNew("shp_"), ProNumber: "PRO-500"},
		}, nil).
		Once()

	plan, err := svc.writer.PreviewCreate(t.Context(), entity, previewActor(entity))

	require.Nil(t, plan)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "bol")
}

func TestPreviewCreate_RequiresAShipment(t *testing.T) {
	t.Parallel()

	svc := assembled(&rig{l: zap.NewNop()})

	_, err := svc.writer.PreviewCreate(t.Context(), nil, nil)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
}

func unpricedCreateService(
	t *testing.T,
	entity *shipment.Shipment,
	disposition billingcontrol.UnratedShipmentDisposition,
) *rig {
	t.Helper()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().Get(mock.Anything, mock.Anything).
		Return(&shipmentcontrol.ShipmentControl{}, nil).
		Once()

	return assembled(&rig{
		l:           zap.NewNop(),
		repo:        shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo: controlRepo,
		validator:   coverageValidator(t, shipmentservicetest.BillingControlWith(t, disposition)),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewEngine(t),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		coordinator: newStateCoordinator(),
	})
}

func TestPreviewCreate_RefusesAShipmentNothingPrices(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.FormulaTemplateID = pulid.Nil
	svc := unpricedCreateService(t, entity, "")

	plan, err := svc.writer.PreviewCreate(t.Context(), entity, previewActor(entity))

	require.Nil(t, plan)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "formulaTemplateId")
}

func TestPreviewCreate_ZeroAndFlagNamesTheUnratedShipment(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.FormulaTemplateID = pulid.Nil
	svc := unpricedCreateService(t, entity, billingcontrol.UnratedShipmentDispositionZeroAndFlag)

	plan, err := svc.writer.PreviewCreate(t.Context(), entity, previewActor(entity))

	require.NoError(t, err)
	require.Len(t, plan.Advisories, 1)
	assert.Equal(t, shipmentvocab.RateCoverageRuleKey, plan.Advisories[0].RuleKey)
	assert.Contains(t, plan.Advisories[0].Error(), "priced at zero")
}
