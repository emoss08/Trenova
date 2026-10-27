//go:build integration

package shipmentservice

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueuerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/chargeallocationrepository"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/billingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/userrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/orderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentadditionalchargerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommodityrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmenteventrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderderivation"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/customerrepository"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// queryCounter counts every statement the database is sent, by the kind of
// statement and the table it touches, so a test can say how much work one
// operation costs and not only whether it succeeded.
type queryCounter struct {
	enabled atomic.Bool
	mu      sync.Mutex
	total   int
	byKey   map[string]int
}

func (q *queryCounter) BeforeQuery(ctx context.Context, _ *keel.QueryEvent) context.Context {
	return ctx
}

func (q *queryCounter) AfterQuery(_ context.Context, event *keel.QueryEvent) {
	if !q.enabled.Load() {
		return
	}
	key := event.Op
	if event.Table != "" {
		key += " " + event.Table
	}
	q.mu.Lock()
	q.total++
	q.byKey[key]++
	q.mu.Unlock()
}

func (q *queryCounter) start() {
	q.mu.Lock()
	q.total = 0
	q.byKey = make(map[string]int)
	q.mu.Unlock()
	q.enabled.Store(true)
}

func (q *queryCounter) stop() (int, string) {
	q.enabled.Store(false)
	q.mu.Lock()
	defer q.mu.Unlock()
	keys := make([]string, 0, len(q.byKey))
	for key := range q.byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "  %4d  %s\n", q.byKey[key], key)
	}
	return q.total, b.String()
}

// countingGenerator hands out distinct invoice numbers, as the real sequence does.
type countingGenerator struct {
	testutil.TestSequenceGenerator
	n atomic.Int64
}

func (g *countingGenerator) GenerateInvoiceNumber(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
	string,
) (string, error) {
	return fmt.Sprintf("INV-%06d", g.n.Add(1)), nil
}

// maxStatementsPerShipment is what one shipment may cost a bulk transfer that
// marks it ready and queues it: one detailed read of the shipment, its billing
// requirements, one narrow status write, the order recompute its status event
// drives, and the queue item. Every reload removed from this path pushed the
// number down; a change that brings one back fails here.
const maxStatementsPerShipment = 26

type bulkTransferHarness struct {
	svc        shipmentports.ShipmentService
	counter    *queryCounter
	db         *keel.DB
	ctx        context.Context
	tenantInfo pagination.TenantInfo
	fixture    *testutil.ShipmentIntegrationFixture
	data       *seedtest.TestData
	shipments  shipmentports.ShipmentRepository
	orders     orderports.OrderRepository
}

func newBulkTransferHarness(t *testing.T) *bulkTransferHarness {
	t.Helper()

	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	counter := &queryCounter{byKey: make(map[string]int)}
	cfg := dbpool.KeelConfig(nil, []keel.Observer{counter})
	cfg.DSN = dsn
	observed, err := keel.Open(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(observed.Close)

	logger := zap.NewNop()
	conn := postgres.NewTestConnection(observed)

	orderRepo := orderrepository.New(orderrepository.Params{DB: conn, Logger: logger})
	allocationRepo := chargeallocationrepository.New(
		chargeallocationrepository.Params{DB: conn, Logger: logger},
	)
	shipmentRepo := shipmentrepository.New(shipmentrepository.Params{
		DB:        conn,
		Logger:    logger,
		Generator: testutil.TestSequenceGenerator{SingleValue: "PRO-BULK"},
		MoveRepository: shipmentmoverepository.New(
			shipmentmoverepository.Params{DB: conn, Logger: logger},
		),
		AdditionalChargeRepository: shipmentadditionalchargerepository.New(
			shipmentadditionalchargerepository.Params{DB: conn, Logger: logger},
		),
		ChargeAllocationRepository: allocationRepo,
		CommodityRepository: shipmentcommodityrepository.New(
			shipmentcommodityrepository.Params{DB: conn, Logger: logger},
		),
		OrderRepository: orderRepo,
	})
	customerRepo := customerrepository.New(customerrepository.Params{DB: conn, Logger: logger})
	userRepo := userrepository.New(userrepository.Params{DB: conn, Logger: logger})
	realtime := &mocks.NoopRealtimeService{}

	derivation := orderderivation.New(orderderivation.Params{
		Logger:       logger,
		ShipmentRepo: shipmentRepo,
		OrderRepo:    orderRepo,
		Realtime:     realtime,
	})
	events := shipmenteventservice.New(shipmenteventservice.Params{
		Logger:    logger,
		Repo:      shipmenteventrepository.New(shipmenteventrepository.Params{DB: conn, Logger: logger}),
		Realtime:  realtime,
		Observers: []shipmenteventports.ShipmentEventObserver{derivation},
	})

	billingQueue := billingqueueservice.New(billingqueueservice.Params{
		Logger:               logger,
		DB:                   conn,
		Repo:                 billingqueuerepository.New(billingqueuerepository.Params{DB: conn, Logger: logger}),
		ShipmentRepo:         shipmentRepo,
		CustomerRepo:         customerRepo,
		UserRepo:             userRepo,
		ChargeAllocationRepo: allocationRepo,
		Generator:            &countingGenerator{},
		AuditService:         &mocks.NoopAuditService{},
		Realtime:             realtime,
		Validator:            billingqueueservice.NewValidator(billingqueueservice.ValidatorParams{DB: conn}),
		OrderDerivation:      derivation,
	})

	svc := New(Params{
		Logger:               logger,
		DB:                   conn,
		Repo:                 shipmentRepo,
		OrderRepo:            orderRepo,
		UserRepo:             userRepo,
		CustomerRepo:         customerRepo,
		ChargeAllocationRepo: allocationRepo,
		DocumentRepo:         documentrepository.New(documentrepository.Params{DB: conn, Logger: logger}),
		BillingRepo:          billingcontrolrepository.New(billingcontrolrepository.Params{DB: conn, Logger: logger}),
		BillingQueueService:  billingQueue,
		Permissions:          permissionportsmock.NewMockPermissionEngine(t),
		AuditService:         &mocks.NoopAuditService{},
		EventService:         events,
		Coordinator:          newStateCoordinator(),
		OrderDerivation:      derivation,
	})

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	ctx = integrationTenantContext(ctx, tenantInfo)

	return &bulkTransferHarness{
		svc:        svc,
		counter:    counter,
		db:         db,
		ctx:        ctx,
		tenantInfo: tenantInfo,
		fixture:    testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenantInfo),
		data:       data,
		shipments:  shipmentRepo,
		orders:     orderRepo,
	}
}

// completedShipment is a delivered leg of its own order with a freight charge,
// the shape a biller transfers every day.
func (h *bulkTransferHarness) completedShipment(t *testing.T, n int) *shipment.Shipment {
	t.Helper()

	ord := &order.Order{
		ID:             pulid.MustNew("ord_"),
		OrganizationID: h.tenantInfo.OrgID,
		BusinessUnitID: h.tenantInfo.BuID,
		CustomerID:     h.fixture.Customer.ID,
		OrderNumber:    fmt.Sprintf("ORD-BULK-%03d", n),
		Status:         order.StatusCompleted,
	}
	require.NoError(t, orderdb.Orders.Create(h.ctx, h.db, ord))

	graph := testutil.CreateShipmentGraph(t, h.ctx, h.db, h.fixture, h.tenantInfo,
		testutil.ShipmentGraphParams{
			BOL:          fmt.Sprintf("BOL-BULK-%03d", n),
			ProNumber:    fmt.Sprintf("PRO-BULK-%03d", n),
			MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusCompleted},
		},
	)

	freight := decimal.NewFromInt(1200)
	_, err := keel.Exec(h.ctx, h.db,
		"UPDATE shipments SET status = $1, order_id = $2, freight_charge_amount = $3, "+
			"total_charge_amount = $3 WHERE id = $4",
		string(shipment.StatusCompleted), ord.ID.String(), freight, graph.Shipment.ID.String())
	require.NoError(t, err)

	return graph.Shipment
}

func (h *bulkTransferHarness) addAccessorial(t *testing.T, shp *shipment.Shipment) *shipment.AdditionalCharge {
	t.Helper()

	accessorial := testutil.MustCreateAccessorialCharge(
		t, h.ctx, h.db, h.tenantInfo, "LUMPER", accessorialcharge.MethodFlat, "", "150",
	)
	charge := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		OrganizationID:      h.tenantInfo.OrgID,
		BusinessUnitID:      h.tenantInfo.BuID,
		ShipmentID:          shp.ID,
		AccessorialChargeID: accessorial.ID,
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(150),
		Unit:                1,
	}
	require.NoError(t, shipmentdb.AdditionalCharges.Create(h.ctx, h.db, charge))

	_, err := keel.Exec(h.ctx, h.db,
		"UPDATE shipments SET other_charge_amount = $1, total_charge_amount = $2 WHERE id = $3",
		decimal.NewFromInt(150), decimal.NewFromInt(1350), shp.ID.String())
	require.NoError(t, err)

	return charge
}

func TestBulkTransferToBillingIntegration_QueriesPerShipment(t *testing.T) {
	h := newBulkTransferHarness(t)

	const count = 5
	ids := make([]pulid.ID, 0, count)
	for i := range count {
		ids = append(ids, h.completedShipment(t, i).ID)
	}
	charged, err := h.shipments.GetByID(h.ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         ids[0],
		TenantInfo: h.tenantInfo,
	})
	require.NoError(t, err)
	charge := h.addAccessorial(t, charged)

	actor := testutil.NewSessionActor(h.data.User.ID, h.tenantInfo.OrgID, h.tenantInfo.BuID)
	h.counter.start()
	resp, transferErr := h.svc.BulkTransferToBilling(h.ctx, &shipmentports.BulkTransferShipmentToBillingRequest{
		ShipmentIDs:                    ids,
		BillType:                       billingqueue.BillTypeInvoice,
		MarkCompletedReadyToInvoice:    true,
		SuppressExceptionNotifications: true,
	}, actor)
	total, breakdown := h.counter.stop()
	require.NoError(t, transferErr)

	for _, result := range resp.Results {
		require.Truef(t, result.Success, "%s: %s", result.ShipmentID, result.Error)
		require.True(t, result.MarkedReadyToInvoice)
		require.NotNil(t, result.Item)
	}
	assert.True(t, resp.Results[0].Item.AllocatedTotalAmount.Equal(decimal.NewFromInt(1350)),
		"the accessorial is billed with the freight")

	perShipment := float64(total) / count
	t.Logf("%d statements for %d shipments (%.1f each)\n%s", total, count, perShipment, breakdown)
	assert.LessOrEqualf(t, perShipment, float64(maxStatementsPerShipment),
		"a bulk transfer spends %.1f statements per shipment:\n%s", perShipment, breakdown)

	kept, err := shipmentdb.AdditionalCharges.Get(h.ctx, h.db, charge.ID)
	require.NoError(t, err)
	assert.True(t, kept.Amount.Equal(charge.Amount), "marking ready leaves the charges as they were")
	assert.Equal(t, charge.Version, kept.Version)

	for _, id := range ids {
		shp, getErr := h.shipments.GetByID(h.ctx, &shipmentports.GetShipmentByIDRequest{
			ID:         id,
			TenantInfo: h.tenantInfo,
		})
		require.NoError(t, getErr)
		assert.Equal(t, shipment.StatusReadyToInvoice, shp.Status)
		assert.NotNil(t, shp.MarkedReadyToBillAt)
		assert.False(t, shp.BillingTransferStatus.IsOutsideBillingQueue(),
			"the shipment reads as transferred")

		ord, ordErr := h.orders.GetByID(h.ctx, orderports.GetOrderByIDRequest{
			ID:         shp.OrderID,
			TenantInfo: h.tenantInfo,
		})
		require.NoError(t, ordErr)
		assert.Equal(t, order.Derive([]shipment.Status{shipment.StatusReadyToInvoice}), ord.Status,
			"the order follows its only leg")
	}
}
