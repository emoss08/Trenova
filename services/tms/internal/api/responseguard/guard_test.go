package responseguard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerinjury"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routeOptions struct {
	install    bool
	selfScoped bool
	references bool
}

// serve writes value through helpers.JSON as a signed-in caller the engine
// denies the given resources, and returns the decoded body and the header.
func serve(
	t *testing.T,
	denied []permission.Resource,
	value any,
	opts routeOptions,
) (map[string]any, string) {
	t.Helper()

	raw, header := serveRaw(t, denied, value, opts)
	body := map[string]any{}
	require.NoError(t, sonic.Unmarshal(raw, &body))

	return body, header
}

func serveRaw(
	t *testing.T,
	denied []permission.Resource,
	value any,
	opts routeOptions,
) ([]byte, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	names := make([]string, 0, len(denied))
	for _, resource := range denied {
		names = append(names, resource.String())
	}
	engine := &mocks.DenyResourcesPermissionEngine{Denied: names}
	guard := NewGuard(GuardParams{
		PermissionEngine: engine,
		ReferenceViews:   refview.NewPort(refview.Default(), engine),
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(
			c,
			pulid.MustNew("usr_"),
			pulid.MustNew("bu_"),
			pulid.MustNew("org_"),
		)
		c.Next()
	})
	if opts.install {
		router.Use(guard.Install())
	}
	handlers := make([]gin.HandlerFunc, 0, 3)
	if opts.selfScoped {
		handlers = append(handlers, SelfScoped())
	}
	if opts.references {
		handlers = append(handlers, References())
	}
	handlers = append(handlers, func(c *gin.Context) { helpers.JSON(c, http.StatusOK, value) })
	router.GET("/", handlers...)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	return rec.Body.Bytes(), rec.Header().Get(WithheldHeader)
}

var guarded = routeOptions{install: true}

func driver() *worker.Worker {
	return &worker.Worker{
		ID:                    pulid.MustNew("wrk_"),
		FirstName:             "Dana",
		LastName:              "Reyes",
		WholeName:             "Dana Reyes",
		Status:                "Active",
		AddressLine1:          "12 Home Street",
		City:                  "Columbus",
		PostalCode:            "43004",
		PhoneNumber:           "614-555-0100",
		Email:                 "dana@example.com",
		EmergencyContactName:  "Sam Reyes",
		EmergencyContactPhone: "614-555-0199",
		Profile: &worker.WorkerProfile{
			ID:            pulid.MustNew("wp_"),
			LicenseNumber: "D1234567",
		},
	}
}

func client() *customer.Customer {
	return &customer.Customer{
		ID:           pulid.MustNew("cus_"),
		Code:         "ACME",
		Name:         "Acme Foods",
		AddressLine1: "1 Market Street",
		City:         "Dayton",
		ExternalID:   "ERP-77",
		DOTNumber:    "1234567",
	}
}

func dispatchedShipment() *shipment.Shipment {
	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "PRO-1",
		BOL:       "BOL-1",
		Customer:  client(),
		Moves: []*shipment.ShipmentMove{{
			ID: pulid.MustNew("smv_"),
			Stops: []*shipment.Stop{{
				ID:          pulid.MustNew("stp_"),
				AddressLine: "1 Dock Road",
				Location: &location.Location{
					ID:           pulid.MustNew("loc_"),
					Code:         "DOCK1",
					Name:         "North Dock",
					AddressLine1: "1 Dock Road",
					Description:  "Gate code 4411",
				},
			}},
			Assignment: &shipment.Assignment{
				ID:            pulid.MustNew("a_"),
				PrimaryWorker: driver(),
				Tractor: &tractor.Tractor{
					ID:   pulid.MustNew("tr_"),
					Code: "T-100",
					Vin:  "1HGCM82633A004352",
				},
			},
		}},
	}
}

func object(t *testing.T, value any, path ...any) map[string]any {
	t.Helper()
	current := value
	for _, step := range path {
		switch key := step.(type) {
		case string:
			m, ok := current.(map[string]any)
			require.Truef(t, ok, "%v is not an object", step)
			current = m[key]
		case int:
			list, ok := current.([]any)
			require.Truef(t, ok, "%v is not a list", step)
			require.Greater(t, len(list), key)
			current = list[key]
		}
	}
	out, ok := current.(map[string]any)
	require.Truef(t, ok, "%v is not an object", path)

	return out
}

func TestShipmentShowsItsDriversNameAndWithholdsTheRestWithoutWorkerRead(t *testing.T) {
	t.Parallel()

	body, header := serve(
		t,
		[]permission.Resource{permission.ResourceWorker},
		dispatchedShipment(),
		guarded,
	)

	primary := object(t, body, "moves", 0, "assignment", "primaryWorker")
	assert.Equal(t, "Dana", primary["firstName"])
	assert.Equal(t, "Reyes", primary["lastName"])
	assert.Equal(t, "Dana Reyes", primary["wholeName"])
	assert.NotEmpty(t, primary["id"])
	for _, field := range []string{
		"addressLine1", "city", "postalCode", "phoneNumber", "email",
		"emergencyContactName", "emergencyContactPhone", "profile",
	} {
		value, present := primary[field]
		assert.Truef(t, present, "%s should be answered, as null", field)
		assert.Nilf(t, value, "%s was not withheld", field)
	}
	assert.Equal(t, "moves.assignment.primaryWorker", header)

	assert.Equal(t, "PRO-1", body["proNumber"], "the shipment is the route's own record")
	assert.Equal(t, "ERP-77", object(t, body, "customer")["externalId"],
		"the caller reads customers")
	assert.Equal(t, "1HGCM82633A004352",
		object(t, body, "moves", 0, "assignment", "tractor")["vin"])
}

func TestShipmentShowsTheWholeDriverToAWorkerReader(t *testing.T) {
	t.Parallel()

	body, header := serve(t, nil, dispatchedShipment(), guarded)

	primary := object(t, body, "moves", 0, "assignment", "primaryWorker")
	assert.Equal(t, "12 Home Street", primary["addressLine1"])
	assert.Equal(t, "614-555-0199", primary["emergencyContactPhone"])
	assert.Equal(t, "D1234567", object(t, primary, "profile")["licenseNumber"])
	assert.Empty(t, header)
}

func TestShipmentWithholdsEveryLinkedKindTheCallerCannotRead(t *testing.T) {
	t.Parallel()

	body, header := serve(t, []permission.Resource{
		permission.ResourceCustomer,
		permission.ResourceLocation,
		permission.ResourceTractor,
		permission.ResourceShipmentMove,
		permission.ResourceShipmentStop,
	}, dispatchedShipment(), guarded)

	cus := object(t, body, "customer")
	assert.Equal(t, "Acme Foods", cus["name"])
	assert.Equal(t, "1 Market Street", cus["addressLine1"])
	assert.Equal(t, "ERP-77", cus["externalId"], "a customer's external id is in its view")
	assert.Nil(t, cus["dotNumber"])

	stop := object(t, body, "moves", 0, "stops", 0)
	assert.Equal(t, "1 Dock Road", stop["addressLine"],
		"a shipment's stop is part of the shipment, whatever the stop resource says")
	loc := object(t, stop, "location")
	assert.Equal(t, "North Dock", loc["name"])
	assert.Nil(t, loc["description"])

	tr := object(t, body, "moves", 0, "assignment", "tractor")
	assert.Equal(t, "T-100", tr["code"])
	assert.Nil(t, tr["vin"])

	assert.Equal(t,
		"customer, moves.assignment.tractor, moves.stops.location",
		header)
}

func TestGuardLeavesTheHandlersValueUntouched(t *testing.T) {
	t.Parallel()

	entity := dispatchedShipment()
	serve(
		t,
		[]permission.Resource{permission.ResourceWorker, permission.ResourceCustomer},
		entity,
		guarded,
	)

	assert.Equal(t, "12 Home Street", entity.Moves[0].Assignment.PrimaryWorker.AddressLine1)
	assert.Equal(t, "ERP-77", entity.Customer.ExternalID)
}

// A record rebuilt as a map must encode as encoding/json encodes the struct:
// every key the struct writes, under the same name, with the same value.
func TestARebuiltRecordEncodesItsVisibleFieldsAsTheStructDoes(t *testing.T) {
	t.Parallel()

	entity := dispatchedShipment()
	original, err := sonic.Marshal(entity)
	require.NoError(t, err)
	want := map[string]any{}
	require.NoError(t, sonic.Unmarshal(original, &want))

	got, _ := serve(t, []permission.Resource{permission.ResourceWorker}, entity, guarded)

	for key, value := range want {
		if key == "moves" {
			continue
		}
		assert.Equalf(t, value, got[key], "root key %s", key)
	}
	for key := range got {
		_, ok := want[key]
		assert.Truef(t, ok, "root key %s is not one the struct writes", key)
	}
	wantMove := object(t, want, "moves", 0)
	gotMove := object(t, got, "moves", 0)
	assert.Equal(t, wantMove["stops"], gotMove["stops"])
	wantWorker := object(t, wantMove, "assignment", "primaryWorker")
	gotWorker := object(t, gotMove, "assignment", "primaryWorker")
	for key := range wantWorker {
		_, ok := gotWorker[key]
		assert.Truef(t, ok, "the withheld record dropped key %s", key)
	}
	for key := range gotWorker {
		_, ok := wantWorker[key]
		assert.Truef(t, ok, "the withheld record wrote key %s the struct leaves out", key)
	}
}

func TestInvoiceShowsOnlyTheViewsOfItsCustomerAndFactoringCompany(t *testing.T) {
	t.Parallel()

	inv := &invoice.Invoice{
		ID:       pulid.MustNew("inv_"),
		Number:   "INV-1",
		Customer: client(),
		ShipperCustomer: &customer.Customer{
			ID: pulid.MustNew("cus_"), Name: "Shipper Co", DOTNumber: "7654321",
		},
		FactoringCompany: &factoringcompany.FactoringCompany{
			ID:          pulid.MustNew("fc_"),
			Code:        "FAC",
			Name:        "Fast Factor",
			Email:       "remit@factor.example",
			PhoneNumber: "800-555-0101",
		},
	}

	body, header := serve(t, []permission.Resource{
		permission.ResourceCustomer, permission.ResourceFactoringCompany,
	}, inv, guarded)

	assert.Equal(t, "INV-1", body["number"])
	assert.Equal(t, "Acme Foods", object(t, body, "customer")["name"])
	assert.Nil(t, object(t, body, "customer")["dotNumber"])
	assert.Equal(t, "Shipper Co", object(t, body, "shipperCustomer")["name"])
	assert.Nil(t, object(t, body, "shipperCustomer")["dotNumber"])
	factor := object(t, body, "factoringCompany")
	assert.Equal(t, "FAC", factor["code"])
	assert.Equal(t, "Fast Factor", factor["name"])
	assert.Nil(t, factor["email"])
	assert.Nil(t, factor["phoneNumber"])
	assert.Equal(t, "customer, factoringCompany, shipperCustomer", header)
}

func TestBillingQueueItemShowsTheViewsOfItsShipmentCustomerAndBiller(t *testing.T) {
	t.Parallel()

	weight := int64(42000)
	item := &billingqueue.BillingQueueItem{
		ID: pulid.MustNew("bqi_"),
		Shipment: &shipment.Shipment{
			ID: pulid.MustNew("shp_"), ProNumber: "PRO-9", Weight: &weight, Customer: client(),
		},
		BillToCustomer: client(),
		AssignedBiller: &tenant.User{
			ID: pulid.MustNew("usr_"), Name: "Billie", EmailAddress: "billie@example.com",
		},
	}

	body, header := serve(t, []permission.Resource{
		permission.ResourceShipment, permission.ResourceCustomer, permission.ResourceUser,
	}, item, guarded)

	shp := object(t, body, "shipment")
	assert.Equal(t, "PRO-9", shp["proNumber"])
	assert.Nil(t, shp["weight"])
	assert.Nil(t, shp["customer"], "the shipment's customer is outside the shipment's view")
	assert.Equal(t, "Acme Foods", object(t, body, "billToCustomer")["name"])
	assert.Nil(t, object(t, body, "billToCustomer")["dotNumber"])
	biller := object(t, body, "assignedBiller")
	assert.Equal(t, "Billie", biller["name"])
	assert.Nil(t, biller["emailAddress"])
	assert.Equal(t, "assignedBiller, billToCustomer, shipment", header)
}

func TestEveryRecordOfAListIsARootRecord(t *testing.T) {
	t.Parallel()

	page := &pagination.ListResult[*shipment.Shipment]{
		Items:      []*shipment.Shipment{dispatchedShipment(), dispatchedShipment()},
		Total:      2,
		TotalExact: true,
	}

	body, header := serve(t, []permission.Resource{permission.ResourceCustomer}, page, guarded)

	items, ok := body["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	for i := range items {
		row := object(t, body, "items", i)
		assert.Equal(t, "PRO-1", row["proNumber"])
		assert.Equal(t, "Acme Foods", object(t, row, "customer")["name"])
		assert.Nil(t, object(t, row, "customer")["dotNumber"])
	}
	assert.InDelta(t, 2, body["total"], 0)
	assert.Equal(t, "items.customer", header)
}

func TestAProtectedFieldIsWithheldFromANestedRecordAsFromARoot(t *testing.T) {
	t.Parallel()

	body, _ := serve(t, []permission.Resource{permission.ResourceProtectedPersonal},
		dispatchedShipment(), guarded)
	profile := object(t, body, "moves", 0, "assignment", "primaryWorker", "profile")
	assert.Empty(t, profile["licenseNumber"])

	raw, _ := serveRaw(t, []permission.Resource{permission.ResourceProtectedPersonal},
		[]*worker.Worker{driver()}, guarded)
	var workers []map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &workers))
	require.Len(t, workers, 1)
	assert.Empty(t, object(t, workers[0], "profile")["licenseNumber"])
	assert.Equal(t, "12 Home Street", workers[0]["addressLine1"])
}

func TestASelfScopedRouteShowsEveryRecordWhole(t *testing.T) {
	t.Parallel()

	body, header := serve(t, []permission.Resource{permission.ResourceWorker},
		dispatchedShipment(), routeOptions{install: true, selfScoped: true})

	primary := object(t, body, "moves", 0, "assignment", "primaryWorker")
	assert.Equal(t, "12 Home Street", primary["addressLine1"])
	assert.Empty(t, header)
}

func TestAReferenceRouteShowsOnlyTheViewOfAKindTheCallerCannotRead(t *testing.T) {
	t.Parallel()

	body, header := serve(t, []permission.Resource{permission.ResourceWorker},
		driver(), routeOptions{install: true, references: true})
	assert.Equal(t, "Dana", body["firstName"])
	assert.Nil(t, body["addressLine1"])
	assert.Nil(t, body["phoneNumber"])
	assert.Empty(t, header, "the record is the response itself")

	readable, _ := serve(t, nil, driver(), routeOptions{install: true, references: true})
	assert.Equal(t, "12 Home Street", readable["addressLine1"])
}

func TestAResponseWithoutAGuardIsWrittenAsIs(t *testing.T) {
	t.Parallel()

	body, header := serve(t, []permission.Resource{permission.ResourceWorker},
		dispatchedShipment(), routeOptions{})

	primary := object(t, body, "moves", 0, "assignment", "primaryWorker")
	assert.Equal(t, "12 Home Street", primary["addressLine1"])
	assert.Empty(t, header)
}

func TestARecordInAMapIsGuarded(t *testing.T) {
	t.Parallel()

	body, header := serve(t, []permission.Resource{permission.ResourceWorker},
		gin.H{"shipment": dispatchedShipment(), "count": 1}, guarded)

	primary := object(t, body, "shipment", "moves", 0, "assignment", "primaryWorker")
	assert.Equal(t, "Dana", primary["firstName"])
	assert.Nil(t, primary["addressLine1"])
	assert.InDelta(t, 1, body["count"], 0)
	assert.Equal(t, "shipment.moves.assignment.primaryWorker", header)
}

func TestARecordInsideAResultStructIsLinked(t *testing.T) {
	t.Parallel()

	type result struct {
		Worker *worker.Worker `json:"worker"`
		Note   string         `json:"note"`
	}

	body, _ := serve(t, []permission.Resource{permission.ResourceWorker},
		result{Worker: driver(), Note: "ok"}, guarded)

	assert.Equal(t, "ok", body["note"])
	assert.Equal(t, "Dana", object(t, body, "worker")["firstName"])
	assert.Nil(t, object(t, body, "worker")["addressLine1"])
}

func TestAPartOfALinkedRecordShowsOnlyItsIDWithoutTheLinkedKindsRead(t *testing.T) {
	t.Parallel()

	inv := &invoice.Invoice{ID: pulid.MustNew("inv_"), Shipment: dispatchedShipment()}
	walker := &walker{
		ctx:      t.Context(),
		decider:  denyAll{},
		planner:  newPlanner(newKindTable(refview.Default().Registry())),
		registry: refview.Default().Registry(),
	}
	linked := walker.recordOf(permission.ResourceShipment, rootRecord, false)
	require.True(t, linked.linked)

	move := walker.recordOf(permission.ResourceShipmentMove, linked, false)
	assert.Equal(t, &record{kind: permission.ResourceShipment, part: true}, move)
	assert.Equal(t, gatePart, walker.gate(permission.ResourceShipmentMove, move))
	assert.Same(t, linked, walker.recordOf(permission.ResourceShipment, linked, false),
		"a record of the linked record's own kind is gated by the same view")

	body, _ := serve(t, []permission.Resource{permission.ResourceShipment}, inv, guarded)
	shp := object(t, body, "shipment")
	assert.Equal(t, "PRO-1", shp["proNumber"])
	assert.Nil(t, shp["moves"], "a linked shipment's moves are outside its view")
}

func TestALeaveCaseAWorkerCarriesNeedsLeaveRead(t *testing.T) {
	t.Parallel()

	registry := refview.Default().Registry()
	walker := &walker{
		ctx:      t.Context(),
		decider:  denyAll{},
		planner:  newPlanner(newKindTable(registry)),
		registry: registry,
	}
	kind, ok := newKindTable(registry).kindOf(reflect.TypeFor[workerleave.WorkerLeaveCase]())
	require.True(t, ok)
	require.Equal(t, permission.ResourceWorkerLeave, kind)

	root := walker.recordOf(permission.ResourceWorker, nil, false)
	leave := walker.recordOf(permission.ResourceWorkerLeave, root, false)
	assert.Equal(t, &record{kind: permission.ResourceWorkerLeave, linked: true}, leave,
		"a leave case under its worker is a linked record, not part of the worker")
	assert.Equal(t, gateView, walker.gate(permission.ResourceWorkerLeave, leave))
	assert.Same(t, root, walker.recordOf(permission.ResourceWorkerCredential, root, false),
		"a credential is still part of its worker")

	type result struct {
		Worker     *worker.Worker                 `json:"worker"`
		LeaveCases []*workerleave.WorkerLeaveCase `json:"leaveCases"`
	}
	value := result{
		Worker: driver(),
		LeaveCases: []*workerleave.WorkerLeaveCase{{
			ID:                  pulid.MustNew("wlc_"),
			LeaveType:           worker.LeaveTypeMedical,
			Reason:              "Surgery",
			Notes:               "Certified by the surgeon",
			CertificationStatus: workerleave.CertificationReceived,
		}},
	}

	body, header := serve(t, []permission.Resource{permission.ResourceWorkerLeave}, value, guarded)
	assert.Equal(t, "dana@example.com", object(t, body, "worker")["email"])
	leaveCase := object(t, body, "leaveCases", 0)
	assert.NotEmpty(t, leaveCase["id"])
	for _, field := range []string{"leaveType", "reason", "notes", "certificationStatus"} {
		value, present := leaveCase[field]
		assert.Truef(t, present, "%s should be answered, as null", field)
		assert.Nilf(t, value, "%s needs worker_leave read", field)
	}
	assert.Equal(t, "leaveCases", header)

	body, _ = serve(t, []permission.Resource{permission.ResourceWorker}, value, guarded)
	assert.Equal(t, "Surgery", object(t, body, "leaveCases", 0)["reason"],
		"worker_leave read shows the case without worker read")
}

type denyAll struct{}

func (denyAll) reads(context.Context, permission.Resource) bool { return false }

func (denyAll) visible(context.Context, permission.Resource, string) bool { return false }

func (denyAll) groups() permission.FieldGroupSet { return 0 }

func TestADrugTestAWorkerCarriesNeedsDOTTestRead(t *testing.T) {
	t.Parallel()

	registry := refview.Default().Registry()
	kinds := newKindTable(registry)
	for typ, want := range map[reflect.Type]permission.Resource{
		reflect.TypeFor[workerdot.WorkerDOTTest]():                  permission.ResourceWorkerDOTTest,
		reflect.TypeFor[workerdot.WorkerDOTViolation]():             permission.ResourceWorkerDOTTest,
		reflect.TypeFor[workerinjury.WorkerInjury]():                permission.ResourceWorkerInjury,
		reflect.TypeFor[timesheet.TimeClockEntry]():                 permission.ResourceTimesheet,
		reflect.TypeFor[profilechange.WorkerProfileChangeRequest](): permission.ResourceProfileChangeRequest,
	} {
		kind, ok := kinds.kindOf(typ)
		require.True(t, ok, typ.String())
		assert.Equal(t, want, kind, typ.String())
	}

	type result struct {
		Worker *worker.Worker             `json:"worker"`
		Tests  []*workerdot.WorkerDOTTest `json:"tests"`
	}
	value := result{
		Worker: driver(),
		Tests: []*workerdot.WorkerDOTTest{{
			ID:            pulid.MustNew("wdot_"),
			Result:        workerdot.DOTResultNegative,
			CollectorName: "Pat Collector",
			MROName:       "Dr. Reviewer",
		}},
	}

	body, header := serve(
		t,
		[]permission.Resource{permission.ResourceWorkerDOTTest},
		value,
		guarded,
	)
	assert.Equal(t, "dana@example.com", object(t, body, "worker")["email"],
		"the worker reader still sees the worker")
	test := object(t, body, "tests", 0)
	assert.NotEmpty(t, test["id"])
	for _, field := range []string{"result", "collectorName", "mroName"} {
		value, present := test[field]
		assert.Truef(t, present, "%s should be answered, as null", field)
		assert.Nilf(t, value, "%s needs worker_dot_test read (49 CFR 40.321)", field)
	}
	assert.Equal(t, "tests", header)

	body, _ = serve(t, []permission.Resource{permission.ResourceWorker}, value, guarded)
	assert.Equal(t, "Negative", object(t, body, "tests", 0)["result"],
		"worker_dot_test read shows the result")
}
