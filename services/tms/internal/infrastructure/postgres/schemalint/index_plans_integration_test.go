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
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
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

	cases := []struct {
		name  string
		ctx   bool
		query keelsnap.Explainer
		index string
	}{
		{"user by email", false, u.AllTenants().Where(u.EmailAddress.Lower().Eq("a@b.c")), "idx_users_email_address"},
		{"user by username", false, u.AllTenants().Where(u.Username.Lower().Eq("dispatch")), "idx_users_username"},
		{"stored mileage lookup", true, sm.Where(
			keel.DynOf(sm.Status).Eq(keel.DynConst(storedmileage.StatusActive)),
			sm.RouteHash.Eq("h"), sm.DistanceUnits.Eq("Miles"), sm.RoutingType.Eq("Practical"),
			sm.Method.Eq("Provider"), sm.DistanceProfileID.Eq(pulid.ID("dp_1")), sm.HazmatSignature.Eq(""),
		), "uq_stored_mileages_active_policy"},
		{"inbound waiting queue", true, m.Where(
			m.Status.In(inboundmessage.StatusInReview, inboundmessage.StatusQuarantined), m.Parked.IsNull(),
		).Order(m.DueAt.Asc().NullsLast(), m.ReceivedAt.Desc(), m.ID.Desc()).Limit(50),
			"idx_inbound_messages_waiting_queue"},
		{"enabled access policies", true, ap.Where(
			ap.Resource.Eq("shipment"), ap.Operation.Eq("read"), ap.Enabled.IsTrue(),
		), "idx_access_policies_org_resource"},
		{"tractors by code", true, tr.Where(tr.Code.Lower().In("t-100")), "idx_tractors_code"},
		{"tractors by plate", true, tr.Where(tr.LicensePlateNumber.Lower().In("abc123")), "idx_tractors_license_plate"},
		{"accessorial charge code", true, ac.Where(ac.Code.Lower().Eq("det")), "uq_accessorial_charges_code"},
		{"shipment by external reference", true, sp.Pick(sp.ID).Where(
			sp.CustomerID.Eq(pulid.ID("cus_1")),
			sp.ExternalReference.Lower().Eq("po-1"),
			sp.Status.Ne(shipment.StatusCanceled),
		), "uq_shipments_external_reference"},
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
