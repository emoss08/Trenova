//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/partiesloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/placesloaders"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/operationsquerytools"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/orderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/recurringshipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicefailurereasoncoderepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicefailurerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/tenderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentioneffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/servicefailurereasoncodeservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/servicefailureservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment/recurringshipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationrepository"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// operationsWorld writes the operations records the converted operations
// reads are held to: shipments with their moves and stops, and the orders,
// series, failures, detention clocks and tenders that hang off them. It
// writes through the admin connection, as agentreadtest.SeedTenant does, and
// every read goes through the RLS connection.
type operationsWorld struct {
	t     *testing.T
	admin *keel.DB

	mu       sync.Mutex
	fixtures map[pulid.ID]*testutil.ShipmentIntegrationFixture
	sources  map[pulid.ID]*testutil.ShipmentGraph
}

func newOperationsWorld(t *testing.T, admin *keel.DB) *operationsWorld {
	t.Helper()

	return &operationsWorld{
		t:        t,
		admin:    admin,
		fixtures: make(map[pulid.ID]*testutil.ShipmentIntegrationFixture),
		sources:  make(map[pulid.ID]*testutil.ShipmentGraph),
	}
}

func tenantInfoOf(tenant *agentreadtest.Tenant) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  tenant.Actor.OrganizationID,
		BuID:   tenant.Actor.BusinessUnitID,
		UserID: tenant.Actor.UserID,
	}
}

// fixture is the tenant's customer, types and location category, written
// once.
func (w *operationsWorld) fixture(
	tenant *agentreadtest.Tenant,
) *testutil.ShipmentIntegrationFixture {
	w.mu.Lock()
	defer w.mu.Unlock()

	org := tenant.Actor.OrganizationID
	if fixture, ok := w.fixtures[org]; ok {
		return fixture
	}
	fixture := testutil.SeedShipmentIntegrationFixture(w.t, w.t.Context(), w.admin,
		&seedtest.TestData{State: &usstate.UsState{ID: tenant.StateID}}, tenantInfoOf(tenant))
	w.fixtures[org] = fixture

	return fixture
}

// shipment writes one shipment of the tenant's, with one move of a pickup
// and a delivery.
func (w *operationsWorld) shipment(
	tenant *agentreadtest.Tenant,
	label string,
) *testutil.ShipmentGraph {
	return testutil.CreateShipmentGraph(w.t, w.t.Context(), w.admin, w.fixture(tenant),
		tenantInfoOf(tenant), testutil.ShipmentGraphParams{
			BOL:          "BOL" + label + tenant.Actor.OrganizationID.String()[4:10],
			ProNumber:    "PRO" + label + tenant.Actor.OrganizationID.String()[4:10],
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
		})
}

// source is the tenant's one shipment a recurring series copies.
func (w *operationsWorld) source(tenant *agentreadtest.Tenant) *testutil.ShipmentGraph {
	org := tenant.Actor.OrganizationID
	w.mu.Lock()
	graph, ok := w.sources[org]
	w.mu.Unlock()
	if ok {
		return graph
	}
	graph = w.shipment(tenant, "SRC")
	w.mu.Lock()
	w.sources[org] = graph
	w.mu.Unlock()

	return graph
}

func (w *operationsWorld) writeOrder(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	owner pulid.ID,
	marked bool,
) (pulid.ID, error) {
	status := order.StatusDraft
	if marked {
		status = order.StatusCanceled
	}
	row := &order.Order{
		ID:             pulid.MustNew("ord_"),
		BusinessUnitID: tenant.Actor.BusinessUnitID,
		OrganizationID: tenant.Actor.OrganizationID,
		CustomerID:     w.fixture(tenant).Customer.ID,
		OwnerID:        owner,
		Status:         status,
		OrderNumber:    "ORD" + label,
		CurrencyCode:   "USD",
		TotalAmount:    decimal.NewNullDecimal(decimal.NewFromInt(1200)),
	}
	ctx = tenant.Ctx(ctx)
	if err := orderdb.Orders.Create(ctx, w.admin, row); err != nil {
		return pulid.Nil, err
	}
	graph := w.shipment(tenant, "O"+label)
	shipments := shipmentdb.Shipments
	if _, err := shipments.Where(shipments.ID.Eq(graph.Shipment.ID)).
		Update(ctx, w.admin, shipments.OrderID.To(row.ID)); err != nil {
		return pulid.Nil, err
	}

	return row.ID, nil
}

func (w *operationsWorld) writeSeries(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	marked bool,
) (pulid.ID, error) {
	status := recurringshipment.StatusActive
	if marked {
		status = recurringshipment.StatusPaused
	}
	row := &recurringshipment.RecurringShipment{
		ID:               pulid.MustNew("rsh_"),
		BusinessUnitID:   tenant.Actor.BusinessUnitID,
		OrganizationID:   tenant.Actor.OrganizationID,
		SourceShipmentID: w.source(tenant).Shipment.ID,
		CustomerID:       w.fixture(tenant).Customer.ID,
		Name:             "Series " + label,
		Status:           status,
		CronExpression:   "0 6 * * 1",
		Timezone:         "America/Chicago",
		LeadTimeDays:     1,
		ExceptionPolicy:  recurringshipment.ExceptionPolicySkip,
		AutoGenerate:     true,
	}

	return row.ID, recurringshipmentdb.RecurringShipments.Create(tenant.Ctx(ctx), w.admin, row)
}

func (w *operationsWorld) writeReasonCode(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	marked bool,
) (pulid.ID, error) {
	row := &servicefailure.ReasonCode{
		ID:             pulid.MustNew("sfrc_"),
		BusinessUnitID: tenant.Actor.BusinessUnitID,
		OrganizationID: tenant.Actor.OrganizationID,
		Code:           "RC" + label,
		Label:          "Reason " + label,
		Category:       servicefailure.ReasonCategoryWeather,
		AppliesTo:      servicefailure.ReasonCodeAppliesToBoth,
		Active:         !marked,
		SortOrder:      100,
	}

	return row.ID, servicefailuredb.ReasonCodes.Create(tenant.Ctx(ctx), w.admin, row)
}

func (w *operationsWorld) writeFailure(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	marked bool,
) (pulid.ID, error) {
	status := servicefailure.StatusOpen
	if marked {
		status = servicefailure.StatusResolved
	}
	graph := w.shipment(tenant, "F"+label)
	move := graph.Moves[0]
	stop := move.Stops[1]
	now := timeutils.NowUnix()
	row := &servicefailure.ServiceFailure{
		ID:              pulid.MustNew("sf_"),
		BusinessUnitID:  tenant.Actor.BusinessUnitID,
		OrganizationID:  tenant.Actor.OrganizationID,
		ShipmentID:      graph.Shipment.ID,
		ShipmentMoveID:  move.ID,
		StopID:          stop.ID,
		Number:          "SF" + label,
		Type:            servicefailure.TypeLateDelivery,
		Source:          servicefailure.SourceDetected,
		Status:          status,
		StopType:        stop.Type,
		ScheduledCutoff: now - 7200,
		ActualArrival:   now - 3600,
		LateMinutes:     60,
		DetectedAt:      now - 3500,
	}
	if marked {
		resolved := now - 600
		row.ResolvedAt = &resolved
		row.ResolvedByID = &tenant.Actor.UserID
	}

	return row.ID, servicefailuredb.ServiceFailures.Create(tenant.Ctx(ctx), w.admin, row)
}

// writeOccurrence writes a detention clock still running at the shipment's
// delivery, or, when held, a stopped charge waiting on approval.
func (w *operationsWorld) writeOccurrence(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	graph *testutil.ShipmentGraph,
	held bool,
) (pulid.ID, error) {
	move := graph.Moves[0]
	stop := move.Stops[1]
	now := timeutils.NowUnix()
	arrived := now - 4*3600
	row := &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		BusinessUnitID:     tenant.Actor.BusinessUnitID,
		OrganizationID:     tenant.Actor.OrganizationID,
		ShipmentID:         graph.Shipment.ID,
		ShipmentMoveID:     move.ID,
		StopID:             stop.ID,
		CustomerID:         graph.Shipment.CustomerID,
		LocationID:         stop.LocationID,
		StopType:           stop.Type,
		ScheduleType:       stop.ScheduleType,
		ArrivedAt:          &arrived,
		ClockStartAt:       arrived,
		FreeTimeExpiresAt:  arrived + 2*3600,
		IsOpen:             !held,
		FreeMinutesGranted: 120,
		RawDwellMinutes:    240,
		BillableMinutes:    120,
		RoundedMinutes:     120,
		BillableAmount:     decimal.NewFromInt(150),
		Currency:           "USD",
		Status:             detention.OccurrenceStatusAccruing,
		NotificationStatus: detention.NotificationStatusNotRequired,
	}
	if held {
		stopped := now - 3600
		row.DepartedAt = &stopped
		row.ClockStopAt = &stopped
		row.Status = detention.OccurrenceStatusPending
		row.RequiresApproval = true
	}

	return row.ID, detentiondb.DetentionOccurrences.Create(tenant.Ctx(ctx), w.admin, row)
}

// writeTender writes a live tender on the move with a declined offer and one
// awaiting the carrier's answer.
func (w *operationsWorld) insert(ctx context.Context, move *shipment.ShipmentMove) error {
	return shipmentdb.ShipmentMoves.Create(ctx, w.admin, move)
}

func (w *operationsWorld) writeTender(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	shipmentID, moveID pulid.ID,
	label string,
) (pulid.ID, error) {
	bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
	ctx = tenant.Ctx(ctx)
	hauler := &carrier.Carrier{
		ID: pulid.MustNew("car_"), BusinessUnitID: bu, OrganizationID: org,
		Status: carrier.StatusActive, Code: "C" + label, Name: "Carrier " + label,
	}
	if err := carrierdb.Carriers.Create(ctx, w.admin, hauler); err != nil {
		return pulid.Nil, err
	}
	row := &tender.Tender{
		ID: pulid.MustNew("tnd_"), BusinessUnitID: bu, OrganizationID: org,
		ShipmentID: shipmentID, ShipmentMoveID: moveID,
		Mode: tender.ModeWaterfall, Status: tender.StatusActive, CurrentRank: 2,
	}
	if err := tenderdb.Tenders.Create(ctx, w.admin, row); err != nil {
		return pulid.Nil, err
	}
	offers := []*tender.TenderOffer{
		{
			ID: pulid.MustNew("tof_"), BusinessUnitID: bu, OrganizationID: org,
			TenderID: row.ID, CarrierID: hauler.ID, Rank: 1,
			RateMethod: shipment.CarrierRateMethodFlat, Rate: decimal.NewFromInt(900),
			OfferTTLSeconds: 900, Channel: tender.ChannelEmail,
			Status: tender.OfferStatusDeclined, DeclineReason: "No trucks in the lane",
		},
		{
			ID: pulid.MustNew("tof_"), BusinessUnitID: bu, OrganizationID: org,
			TenderID: row.ID, CarrierID: hauler.ID, Rank: 2,
			RateMethod: shipment.CarrierRateMethodFlat, Rate: decimal.NewFromInt(950),
			OfferTTLSeconds: 900, Channel: tender.ChannelEmail, Status: tender.OfferStatusSent,
		},
	}

	return row.ID, tenderdb.TenderOffers.CreateMany(ctx, w.admin, offers)
}

// operationsServices are the operations resolvers' services over the RLS
// connection, added to the services the other converted reads reach.
func operationsServices(conn *postgres.Connection, services *agentread.ResolverParams) {
	logger := zap.NewNop()
	orders := orderrepository.New(orderrepository.Params{DB: conn, Logger: logger})
	reasons := servicefailurereasoncoderepository.New(
		servicefailurereasoncoderepository.Params{DB: conn, Logger: logger})
	failures := servicefailurerepository.New(
		servicefailurerepository.Params{DB: conn, Logger: logger},
	)

	services.OrderService = orderservice.New(orderservice.Params{Logger: logger, Repo: orders})
	services.RecurringShipmentService = recurringshipmentservice.New(
		recurringshipmentservice.Params{
			Logger: logger,
			Repo: recurringshipmentrepository.New(
				recurringshipmentrepository.Params{DB: conn, Logger: logger}),
		},
	)
	services.ServiceFailureReasonCodeSvc = servicefailurereasoncodeservice.New(
		servicefailurereasoncodeservice.Params{Logger: logger, Repo: reasons})
	services.ServiceFailureSvc = servicefailureservice.New(servicefailureservice.Params{
		Logger: logger, Repo: failures, ReasonCodeRepo: reasons,
	})
	services.DetentionReview = detentionreview.New(detentionreview.Params{
		OccurrenceRepo: detentionrepository.NewOccurrenceRepository(
			detentionrepository.OccurrenceParams{DB: conn, Logger: logger}),
		Effects: detentioneffects.New(detentioneffects.Params{Logger: logger}),
	})
	services.TenderRepo = tenderrepository.New(tenderrepository.Params{DB: conn, Logger: logger})
}

// operationsListCases are the converted order, recurring shipment, reason
// code and service failure lists, each with the filter that selects its
// marked rows.
func operationsListCases(world *operationsWorld, conn *postgres.Connection) []referenceCase {
	logger := zap.NewNop()
	engine := &mocks.DenyResourcesPermissionEngine{}

	return []referenceCase{
		{
			tool: "order_list", marked: canceled, oldMarked: oldCanceled,
			old: operationsquerytools.NewListOrdersTool(
				orderrepository.New(orderrepository.Params{DB: conn, Logger: logger}), engine),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				return world.writeOrder(ctx, tenant, label, pulid.Nil, marked)
			},
		},
		{
			tool: "shipment_list_recurring", marked: paused, oldMarked: oldPaused,
			old: operationsquerytools.NewListRecurringShipmentsTool(
				recurringshipmentrepository.New(
					recurringshipmentrepository.Params{DB: conn, Logger: logger}),
			),
			write: world.writeSeries,
		},
		{
			tool:   "exception_list_service_failure_reason_codes",
			marked: map[string]any{"active": false}, readsActiveByDefault: true,
			write: world.writeReasonCode,
		},
		{
			tool:   "exception_list_service_failures",
			marked: map[string]any{"status": []any{"Resolved"}},
			write:  world.writeFailure,
		},
	}
}

var (
	canceled    = map[string]any{"status": []any{"Canceled"}}
	oldCanceled = map[string]any{"filters": []any{map[string]any{
		"field": "status", "operator": "in", "values": []any{"Canceled"},
	}}}
	paused    = map[string]any{"status": []any{"Paused"}}
	oldPaused = map[string]any{"filters": []any{map[string]any{
		"field": "status", "operator": "in", "values": []any{"Paused"},
	}}}
)

// operationsEnv is the converted lists' environment with the operations
// services, over two tenants of a fresh database.
func operationsEnv(
	t *testing.T,
) (*convertedListEnv, *operationsWorld, *postgres.Connection, agentread.ResolverParams) {
	t.Helper()

	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	_, services := convertedListCases(conn, admin)
	operationsServices(conn, &services)
	engine := &mocks.DenyResourcesPermissionEngine{}
	customers := partiesloaders.NewCustomerByIDLoaderFactory(
		partiesloaders.CustomerByIDLoaderFactoryParams{
			CustomerService: services.CustomerService,
		},
	)
	places := placesloaders.NewLocationByIDLoaderFactory(
		placesloaders.LocationByIDLoaderFactoryParams{
			LocationRepo: locationrepository.New(
				locationrepository.Params{DB: conn, Logger: zap.NewNop()},
			),
		},
	)

	return &convertedListEnv{
		attach: func(ctx context.Context, tenant *agentreadtest.Tenant) context.Context {
			return places.Attach(customers.Attach(ctx, tenantInfoOf(tenant)), tenantInfoOf(tenant))
		},
		engine: engine,
		reader: agentreadtest.NewReader(
			t,
			agentreadtest.Options{Engine: engine, Resolvers: services},
		),
		public: agentreadtest.NewReader(
			t,
			agentreadtest.Options{Engine: &publicRole{}, Resolvers: services},
		),
		mine:   agentreadtest.SeedTenant(t, admin, "A"),
		theirs: agentreadtest.SeedTenant(t, admin, "B"),
	}, newOperationsWorld(t, admin), conn, services
}

func TestConvertedLists_OperationsReadWhatTheRepositoryToolsReadUnderRowLevelSecurity(
	t *testing.T,
) {
	env, world, conn, _ := operationsEnv(t)

	cases := operationsListCases(world, conn)
	for idx := range cases {
		c := &cases[idx]
		t.Run(c.tool, func(t *testing.T) { checkConvertedList(t, c, env) })
	}
}

// A person whose order grant reaches only their own orders reads, through
// the GraphQL list, exactly the orders the repository list showed them: the
// read_scope policy narrows the screen's statement as it narrowed the tool's.
func TestConvertedLists_OrderListIsNarrowedToTheOwnerOnBothPaths(t *testing.T) {
	env, world, conn, _ := operationsEnv(t)
	tenant := env.mine
	tenantCtx := dbscope.WithTenant(t.Context(), tenant.Scope())

	own := make([]string, 0, 2)
	for idx := range 5 {
		owner := pulid.Nil
		if idx < 2 {
			owner = tenant.Actor.UserID
		}
		id, err := world.writeOrder(tenantCtx, tenant, fmt.Sprintf("N%d", idx), owner, false)
		require.NoError(t, err)
		if idx < 2 {
			own = append(own, id.String())
		}
	}

	scoped := readscope.With(tenantCtx, func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{"order": {Scope: readscope.ScopeOwn}}, nil
	})
	converted, err := agentquerytoolservice.ConvertedListTool("order_list", env.reader, env.engine)
	require.NoError(t, err)
	old := operationsquerytools.NewListOrdersTool(
		orderrepository.New(orderrepository.Params{DB: conn, Logger: zap.NewNop()}), env.engine)

	before, _ := walk(t, scoped, old, tenant.Actor, nil)
	after, _ := walk(t, scoped, converted, tenant.Actor, nil)
	assert.ElementsMatch(t, own, before, "the repository list reads only the person's own")
	assert.ElementsMatch(t, before, after, "the GraphQL list reads what the repository list read")

	everyone, _ := walk(t, readscope.With(tenantCtx, nil), converted, tenant.Actor, nil)
	assert.Len(t, everyone, 5, "without a narrow grant every order is read")
}

func TestConvertedGets_OperationsReadOneRecordOfTheTenantUnderRowLevelSecurity(t *testing.T) {
	env, world, _, _ := operationsEnv(t)

	gets := map[string]*referenceCase{
		"order_get": {
			tool: "order_get",
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				return world.writeOrder(ctx, tenant, "G"+label, pulid.Nil, marked)
			},
		},
		"exception_get_service_failure": {
			tool: "exception_get_service_failure",
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				return world.writeFailure(ctx, tenant, "G"+label, marked)
			},
		},
		"detention_get_occurrence": {
			tool: "detention_get_occurrence",
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				return world.writeOccurrence(ctx, tenant, world.shipment(tenant, "D"+label), marked)
			},
			withheldAtInternal: []string{"occurrence.driverPayAmount", "occurrence.netMargin"},
		},
	}
	occurrenceID := func(record map[string]any) any {
		occurrence, _ := record["occurrence"].(map[string]any)
		return occurrence["id"]
	}
	for name, c := range gets {
		idOf := recordID
		if name == "detention_get_occurrence" {
			idOf = occurrenceID
		}
		t.Run(name, func(t *testing.T) { checkConvertedGet(t, name, c, env, idOf) })
	}

	t.Run("an order carries every leg it groups", func(t *testing.T) {
		get, err := agentquerytoolservice.ConvertedGetTool("order_get", env.reader, env.engine)
		require.NoError(t, err)
		ctx := readscope.With(dbscope.WithTenant(t.Context(), env.mine.Scope()), nil)
		id, err := world.writeOrder(ctx, env.mine, "LEGS", pulid.Nil, false)
		require.NoError(t, err)

		record := callTool(t, ctx, get, env.mine.Actor, map[string]any{"orderId": id.String()})
		legs, _ := record["legs"].([]any)
		require.Len(t, legs, 1)
		assert.Contains(t, legs[0], "proNumber")
	})
}

// The detention board and a shipment's tenders are unpaged lists: each reads
// every row of ours and none of theirs, narrows as its arguments say, reads
// what the service the repository tool read returned, and withholds at
// Public what that role may not see.
func TestConvertedLists_OperationsUnpagedListsReadTheTenantsRowsUnderRowLevelSecurity(
	t *testing.T,
) {
	env, world, _, services := operationsEnv(t)
	actor := env.mine.Actor
	ours := readscope.With(dbscope.WithTenant(t.Context(), env.mine.Scope()), nil)
	theirs := readscope.With(dbscope.WithTenant(t.Context(), env.theirs.Scope()), nil)

	t.Run("detention_list_occurrences", func(t *testing.T) {
		tool, err := agentquerytoolservice.ConvertedListTool("detention_list_occurrences",
			env.reader, env.engine)
		require.NoError(t, err)

		var running, held []string
		var oneShipment pulid.ID
		for idx := range 3 {
			graph := world.shipment(env.mine, fmt.Sprintf("DL%d", idx))
			id, writeErr := world.writeOccurrence(ours, env.mine, graph, idx == 2)
			require.NoError(t, writeErr)
			if idx == 2 {
				held = append(held, id.String())
			} else {
				running = append(running, id.String())
			}
			if idx == 0 {
				oneShipment = graph.Shipment.ID
			}
		}
		other, err := world.writeOccurrence(
			theirs,
			env.theirs,
			world.shipment(env.theirs, "DL9"),
			false,
		)
		require.NoError(t, err)

		occurrences := func(page map[string]any) []string {
			items, _ := page["items"].([]any)
			ids := make([]string, 0, len(items))
			for _, item := range items {
				occurrence, _ := item.(map[string]any)["occurrence"].(map[string]any)
				ids = append(ids, occurrence["id"].(string))
			}
			return ids
		}

		all := occurrences(callTool(t, ours, tool, actor, map[string]any{}))
		assert.ElementsMatch(t, slices.Concat(running, held), all,
			"the running clocks and the charges held for approval, by default")
		assert.NotContains(t, all, other.String())

		tenant := tenantInfoOf(env.mine)
		open, err := services.DetentionReview.ListDesk(ours, tenant)
		require.NoError(t, err)
		waiting, err := services.DetentionReview.ListAwaitingApproval(ours, tenant)
		require.NoError(t, err)
		before := make([]string, 0, len(open)+len(waiting))
		for _, entry := range slices.Concat(open, waiting) {
			before = append(before, entry.Occurrence.ID.String())
		}
		assert.ElementsMatch(t, before, all, "what the repository tool read")

		assert.ElementsMatch(t, running, occurrences(callTool(t, ours, tool, actor,
			map[string]any{"includeAwaitingApproval": false})))
		assert.ElementsMatch(t, held, occurrences(callTool(t, ours, tool, actor,
			map[string]any{"urgency": "AwaitingApproval"})))
		assert.ElementsMatch(t, running[:1], occurrences(callTool(t, ours, tool, actor,
			map[string]any{"shipmentId": oneShipment.String()})))

		hidden, err := agentquerytoolservice.ConvertedListTool("detention_list_occurrences",
			env.public, &publicRole{})
		require.NoError(t, err)
		page := callTool(t, ours, hidden, actor, map[string]any{})
		withheld, _ := page["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		for _, item := range page["items"].([]any) {
			for _, field := range withheld {
				assert.NotContains(t, item, field)
			}
		}
	})

	t.Run("dispatch_list_tenders", func(t *testing.T) {
		tool, err := agentquerytoolservice.ConvertedListTool("dispatch_list_tenders",
			env.reader, env.engine)
		require.NoError(t, err)

		graph := world.shipment(env.mine, "TL")
		second := &shipment.ShipmentMove{
			ID: pulid.MustNew("sm_"), BusinessUnitID: actor.BusinessUnitID,
			OrganizationID: actor.OrganizationID, ShipmentID: graph.Shipment.ID,
			Status: shipment.MoveStatusNew, Loaded: true, Sequence: 1,
		}
		require.NoError(t, world.insert(ours, second))
		first, err := world.writeTender(ours, env.mine, graph.Shipment.ID, graph.Moves[0].ID, "T1")
		require.NoError(t, err)
		other, err := world.writeTender(ours, env.mine, graph.Shipment.ID, second.ID, "T2")
		require.NoError(t, err)
		elsewhere := world.shipment(env.theirs, "TL")
		_, err = world.writeTender(theirs, env.theirs, elsewhere.Shipment.ID,
			elsewhere.Moves[0].ID, "T3")
		require.NoError(t, err)

		page := callTool(
			t,
			ours,
			tool,
			actor,
			map[string]any{"shipmentId": graph.Shipment.ID.String()},
		)
		assert.ElementsMatch(t, []string{first.String(), other.String()}, itemIDs(page))

		listed, err := services.TenderRepo.ListByShipment(
			ours,
			tenderports.ListTendersByShipmentRequest{
				TenantInfo: tenantInfoOf(env.mine), ShipmentID: graph.Shipment.ID,
			},
		)
		require.NoError(t, err)
		before := make([]string, 0, len(listed))
		for _, entity := range listed {
			before = append(before, entity.ID.String())
		}
		assert.ElementsMatch(t, before, itemIDs(page), "what the repository tool read")

		narrowed := callTool(t, ours, tool, actor, map[string]any{
			"shipmentId":     graph.Shipment.ID.String(),
			"shipmentMoveId": graph.Moves[0].ID.String(),
		})
		assert.Equal(t, []string{first.String()}, itemIDs(narrowed))
		row := narrowed["items"].([]any)[0].(map[string]any)
		assert.Equal(t, true, row["live"])
		offers := row["offers"].([]any)
		require.Len(t, offers, 2)
		answerable := []any{
			offers[0].(map[string]any)["answerable"],
			offers[1].(map[string]any)["answerable"],
		}
		assert.ElementsMatch(t, []any{false, true}, answerable)
		assert.NotContains(t, fmt.Sprint(narrowed), "No trucks in the lane",
			"a carrier's own words are never read")

		mine := callTool(t, theirs, tool, env.theirs.Actor,
			map[string]any{"shipmentId": graph.Shipment.ID.String()})
		assert.Empty(t, itemIDs(mine), "another tenant reads none of our tenders")

		hidden, err := agentquerytoolservice.ConvertedListTool("dispatch_list_tenders",
			env.public, &publicRole{})
		require.NoError(t, err)
		page = callTool(
			t,
			ours,
			hidden,
			actor,
			map[string]any{"shipmentId": graph.Shipment.ID.String()},
		)
		withheld, _ := page["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		for _, item := range page["items"].([]any) {
			assert.Contains(t, item, "id")
			for _, field := range withheld {
				assert.NotContains(t, item, field)
			}
		}
	})
}
