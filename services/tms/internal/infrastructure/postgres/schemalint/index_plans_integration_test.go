//go:build integration

package schemalint

import (
	"testing"

	"github.com/emoss08/keel"
	keelsnap "github.com/emoss08/keel/keeltest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchlearningrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket/dispatchmarketdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice/dispatchplanpricedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"github.com/emoss08/trenova/internal/modules/places/domain/storedmileage"
	"github.com/emoss08/trenova/internal/modules/places/domain/storedmileage/storedmileagedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestHotPathQueriesReachTheirIndexesUnderAGenericPlan(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenant := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: pulid.ID("org_plan"),
		BusinessUnitID: pulid.ID("bu_plan"),
	})
	system := dbscope.WithSystem(ctx, "index plan test")

	u := tenantdb.Users
	sm := storedmileagedb.StoredMileages
	m := inboundmessagedb.InboundMessages
	ap := iamdb.AccessPolicies
	tr := tractordb.Tractors
	ac := accessorialchargedb.AccessorialCharges
	sp := shipmentdb.Shipments
	pp := dispatchplanpricedb.DispatchPlanPrices
	mv := dispatchmarketdb.MarketValues
	dc := dispatchcontroldb.DispatchControls

	cases := []struct {
		name  string
		ctx   bool
		query keelsnap.Explainer
		index string
	}{
		{
			"user by email",
			false,
			u.AllTenants().Where(u.EmailAddress.Lower().Eq("a@b.c")),
			"idx_users_email_address",
		},
		{
			"user by username",
			false,
			u.AllTenants().Where(u.Username.Lower().Eq("dispatch")),
			"idx_users_username",
		},
		{"stored mileage lookup", true, sm.Where(
			keel.DynOf(sm.Status).Eq(keel.DynConst(storedmileage.StatusActive)),
			sm.RouteHash.Eq("h"),
			sm.DistanceUnits.Eq("Miles"),
			sm.RoutingType.Eq("Practical"),
			sm.Method.Eq(
				"Provider",
			),
			sm.DistanceProfileID.Eq(pulid.ID("dp_1")),
			sm.HazmatSignature.Eq(""),
		), "uq_stored_mileages_active_policy"},
		{"inbound waiting queue", true, m.Where(
			m.Status.In(
				inboundmessage.StatusInReview,
				inboundmessage.StatusQuarantined,
			),
			m.Parked.IsNull(),
		).Order(m.DueAt.Asc().NullsLast(), m.ReceivedAt.Desc(), m.ID.Desc()).Limit(50),
			"idx_inbound_messages_waiting_queue"},
		{"enabled access policies", true, ap.Where(
			ap.Resource.Eq("shipment"), ap.Operation.Eq("read"), ap.Enabled.IsTrue(),
		), "idx_access_policies_org_resource"},
		{"tractors by code", true, tr.Where(tr.Code.Lower().In("t-100")), "idx_tractors_code"},
		{
			"tractors by plate",
			true,
			tr.Where(tr.LicensePlateNumber.Lower().In("abc123")),
			"idx_tractors_license_plate",
		},
		{
			"accessorial charge code",
			true,
			ac.Where(ac.Code.Lower().Eq("det")),
			"uq_accessorial_charges_code",
		},
		{"shipment by external reference", true, sp.Pick(sp.ID).Where(
			sp.CustomerID.Eq(pulid.ID("cus_1")),
			sp.ExternalReference.Lower().Eq("po-1"),
			sp.Status.Ne(shipmentvocab.StatusCanceled),
		), "uq_shipments_external_reference"},
		{
			"dispatch learning decisions after the cursor",
			true,
			dispatchlearningrepository.DecisionEventsQuery(dispatchlearningports.DecisionCursor{
				At: 1_700_000_000,
				ID: pulid.ID("sev_1"),
			}, 500),
			"idx_shipment_events_user_driver_assigned",
		},
		{
			"plan prices by move",
			true,
			pp.Where(pp.MoveID.In(pulid.ID("smv_1"), pulid.ID("smv_2"))).Limit(2),
			"uq_dispatch_plan_prices_move",
		},
		{
			"market values by cell",
			true,
			mv.Where(mv.Cell.In(int64(1), int64(2))).Limit(84),
			"uq_dispatch_market_values_cell_slot",
		},
		{
			"organizations that learn their dispatch style",
			false,
			dc.AllTenants().
				Pick(dc.OrganizationID, dc.BusinessUnitID).
				Where(dc.LearnDispatchStyle.IsTrue()).
				Order(dc.OrganizationID.Asc(), dc.BusinessUnitID.Asc()).
				Limit(100),
			"idx_dispatch_controls_learn_dispatch_style",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			scope := system
			if c.ctx {
				scope = tenant
			}
			keelsnap.UsesIndex(t, scope, db, c.query, c.index)
		})
	}
}
