package billingqueueservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type passthroughDB struct {
	ports.DBConnection
}

func (passthroughDB) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type countingGenerator struct {
	seqgen.Generator
	next int
}

func (g *countingGenerator) GenerateInvoiceNumber(
	_ context.Context,
	_, _ pulid.ID,
	_, _ string,
) (string, error) {
	g.next++
	return "INV-" + string(rune('0'+g.next)), nil
}

func transferFixture(t *testing.T) (*shipment.Shipment, pulid.ID, pagination.TenantInfo) {
	t.Helper()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	amd := pulid.MustNew("cus_")
	charge := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              "Flat",
		Amount:              decimal.NewFromInt(120),
		Unit:                1,
	}
	chargeID := charge.ID
	shp := &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		CustomerID:          pulid.MustNew("cus_"),
		OrderID:             pulid.MustNew("ord_"),
		Status:              shipment.StatusReadyToInvoice,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1000)),
		AdditionalCharges:   []*shipment.AdditionalCharge{charge},
		ChargeAllocations: []*shipment.ChargeAllocation{{
			ID:                 pulid.MustNew("chal_"),
			ChargeKind:         shipment.ChargeAllocationKindAccessorial,
			AdditionalChargeID: &chargeID,
			BillToCustomerID:   amd,
			Method:             shipment.ChargeAllocationMethodPercent,
			Percent:            decimal.NewNullDecimal(decimal.NewFromInt(100)),
		}},
	}

	return shp, amd, tenantInfo
}

func testValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*billingqueue.BillingQueueItem]().
			WithModelName("Billing Queue Item").
			WithCustomRule(createStatusConstraintsRule()).
			Build(),
	}
}

func TestTransferToBillingItemsCreatesOneItemPerPayer(t *testing.T) {
	t.Parallel()

	shp, amd, tenantInfo := transferFixture(t)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == shp.ID && req.ShipmentOptions.ExpandShipmentDetails
		})).
		Return(shp, nil).
		Once()

	repo := mocks.NewMockBillingQueueRepository(t)
	repo.EXPECT().
		ExistsByShipmentPayerAndType(mock.Anything, tenantInfo, shp.ID, mock.Anything, billingqueue.BillTypeInvoice).
		Return(false, nil).
		Twice()
	var created []*billingqueue.BillingQueueItem
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*billingqueue.BillingQueueItem")).
		RunAndReturn(func(_ context.Context, item *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			item.ID = pulid.MustNew("bqi_")
			created = append(created, item)
			return item, nil
		}).
		Twice()

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("repositories.GetCustomerByIDRequest")).
		RunAndReturn(func(_ context.Context, req repositories.GetCustomerByIDRequest) (*customer.Customer, error) {
			return &customer.Customer{ID: req.ID, BillingProfile: &customer.CustomerBillingProfile{}}, nil
		}).
		Twice()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Twice()

	svc := &service{
		l:            zap.NewNop(),
		db:           passthroughDB{},
		repo:         repo,
		shipmentRepo: shipmentRepo,
		customerRepo: customerRepo,
		generator:    &countingGenerator{},
		auditService: audit,
		realtime:     realtime,
		validator:    testValidator(),
	}

	result, err := svc.TransferToBillingItems(t.Context(), &services.TransferToBillingRequest{
		ShipmentID: shp.ID,
		BillType:   billingqueue.BillTypeInvoice,
		TenantInfo: tenantInfo,
	}, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Len(t, result.Items, 2)
	require.Len(t, created, 2)
	require.NotNil(t, result.Primary)
	assert.Equal(t, shp.CustomerID, result.Primary.BillToCustomerID, "the shipment's own payer is primary")

	byPayer := make(map[pulid.ID]*billingqueue.BillingQueueItem, 2)
	for _, item := range created {
		byPayer[item.BillToCustomerID] = item
		assert.Equal(t, shp.ID, item.ShipmentID)
		assert.Equal(t, shp.OrderID, item.OrderID)
		assert.Equal(t, billingqueue.StatusReadyForReview, item.Status)
		assert.NotEmpty(t, item.Number)
	}
	require.Contains(t, byPayer, shp.CustomerID)
	require.Contains(t, byPayer, amd)
	assert.True(t, byPayer[shp.CustomerID].AllocatedTotalAmount.Equal(decimal.NewFromInt(1000)))
	assert.Equal(t, int64(100_000), byPayer[shp.CustomerID].AllocatedTotalAmountMinor)
	assert.True(t, byPayer[amd].AllocatedTotalAmount.Equal(decimal.NewFromInt(120)))
	assert.Equal(t, int64(12_000), byPayer[amd].AllocatedTotalAmountMinor)
	assert.NotEqual(t, byPayer[shp.CustomerID].Number, byPayer[amd].Number, "every payer's item gets its own number")
}

func TestTransferToBillingItemsRefusesWhenAPayerIsAlreadyQueued(t *testing.T) {
	t.Parallel()

	shp, amd, tenantInfo := transferFixture(t)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(shp, nil).
		Once()

	repo := mocks.NewMockBillingQueueRepository(t)
	repo.EXPECT().
		ExistsByShipmentPayerAndType(mock.Anything, tenantInfo, shp.ID, shp.CustomerID, billingqueue.BillTypeInvoice).
		Return(false, nil).
		Once()
	repo.EXPECT().
		ExistsByShipmentPayerAndType(mock.Anything, tenantInfo, shp.ID, amd, billingqueue.BillTypeInvoice).
		Return(true, nil).
		Once()

	svc := &service{l: zap.NewNop(), repo: repo, shipmentRepo: shipmentRepo}

	result, err := svc.TransferToBillingItems(t.Context(), &services.TransferToBillingRequest{
		ShipmentID: shp.ID,
		BillType:   billingqueue.BillTypeInvoice,
		TenantInfo: tenantInfo,
	}, &services.RequestActor{})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errortypes.IsConflictError(err), "nothing is written when any payer is already queued")
}

func TestTransferToBillingItemsRequiresAReadyToInvoiceShipment(t *testing.T) {
	t.Parallel()

	shp, _, tenantInfo := transferFixture(t)
	shp.Status = shipment.StatusCompleted

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(shp, nil).Once()

	svc := &service{l: zap.NewNop(), shipmentRepo: shipmentRepo}

	_, err := svc.TransferToBillingItems(t.Context(), &services.TransferToBillingRequest{
		ShipmentID: shp.ID,
		BillType:   billingqueue.BillTypeInvoice,
		TenantInfo: tenantInfo,
	}, &services.RequestActor{})
	require.Error(t, err)

	_, err = svc.TransferToBillingItems(t.Context(), nil, &services.RequestActor{})
	require.Error(t, err)
}

func TestShouldAutoApprove(t *testing.T) {
	t.Parallel()

	svc := &service{}
	intel := pulid.MustNew("cus_")
	amd := pulid.MustNew("cus_")

	assert.True(t, svc.shouldAutoApprove(&services.TransferToBillingRequest{AutoApprove: true}, amd))
	assert.True(t, svc.shouldAutoApprove(&services.TransferToBillingRequest{AutoApprovePayerIDs: []pulid.ID{intel, amd}}, amd))
	assert.False(t, svc.shouldAutoApprove(&services.TransferToBillingRequest{AutoApprovePayerIDs: []pulid.ID{intel}}, amd))
	assert.False(t, svc.shouldAutoApprove(&services.TransferToBillingRequest{}, amd))
}
