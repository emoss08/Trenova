package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func personReachingEach(
	sensitivity permission.FieldSensitivity,
	resources ...permission.Resource,
) *fakePermissions {
	readable := make(map[string]*serviceports.ResourcePermissionDetail, len(resources))
	for _, resource := range resources {
		readable[resource.String()] = &serviceports.ResourcePermissionDetail{
			Resource:       resource.String(),
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: sensitivity,
		}
	}

	return &fakePermissions{readable: readable}
}

type nestedPeopleFixture struct {
	managerID string
	workerID  string
	orgID     string
	buID      string
	userID    string
	carrierID string
}

func nestedPeopleRecord(fixture *nestedPeopleFixture) map[string]any {
	fixture.managerID = pulid.MustNew(tenant.UserIDPrefix).String()
	fixture.workerID = pulid.MustNew(agent.SubjectWorker.IDPrefix()).String()
	fixture.orgID = pulid.MustNew(tenant.OrganizationIDPrefix).String()
	fixture.buID = pulid.MustNew(tenant.BusinessUnitIDPrefix).String()
	fixture.userID = pulid.MustNew(tenant.UserIDPrefix).String()
	fixture.carrierID = pulid.MustNew(shipment.CarrierAssignmentIDPrefix).String()

	return map[string]any{
		"id":   pulid.MustNew("tr_").String(),
		"code": "T-104",
		"primaryWorker": map[string]any{
			"id":          fixture.workerID,
			"firstName":   "Dana",
			"lastName":    "Whitfield",
			"email":       "dana.whitfield@example.com",
			"phoneNumber": "815-555-0142",
			"manager": map[string]any{
				"id":           fixture.managerID,
				"name":         "Morgan Reyes",
				"emailAddress": "morgan.reyes@example.com",
				"username":     "mreyes",
			},
			"profile": map[string]any{
				"licenseNumber": "W123-4567-8901",
				"endorsement":   "X",
			},
		},
		"organization": map[string]any{
			"id":           fixture.orgID,
			"name":         "Lakeshore Logistics",
			"addressLine1": "900 Harbor Rd",
		},
		"businessUnit": map[string]any{
			"id":          fixture.buID,
			"name":        "Midwest",
			"phoneNumber": "312-555-0100",
		},
		"events": []any{map[string]any{
			"id": pulid.MustNew("evt_").String(),
			"createdBy": map[string]any{
				"id":           fixture.userID,
				"name":         "Avery Chen",
				"emailAddress": "avery.chen@example.com",
			},
		}},
		"carrierAssignment": map[string]any{
			"id":                  fixture.carrierID,
			"status":              "Confirmed",
			"externalDriverName":  "Pat Carrier",
			"externalDriverPhone": "312-555-0111",
		},
	}
}

func redact(
	t *testing.T,
	permissions *fakePermissions,
	params *serviceports.QueryToolParams,
	record map[string]any,
) map[string]any {
	t.Helper()

	gated, err := newFieldAccess(permissions).redactor(t.Context(), params).withhold(record)
	require.NoError(t, err)

	return encodedDocument(t, gated)
}

func TestRedactor_AUserNestedInAWorkerIsReducedRatherThanFieldGated(t *testing.T) {
	t.Parallel()

	fixture := &nestedPeopleFixture{}
	document := redact(t,
		personReachingEach(permission.SensitivityRestricted,
			permission.ResourceWorker, permission.ResourceShipmentMove),
		chatParams(nil, ""),
		nestedPeopleRecord(fixture),
	)

	worker := objectAt(t, document, "primaryWorker")
	assert.Equal(t, "dana.whitfield@example.com", worker["email"],
		"the worker's own contact follows the worker ceiling")
	assert.Equal(t, map[string]any{"id": fixture.managerID, "name": "Morgan Reyes"},
		worker["manager"], "a manager is a user, not a field of the worker")
	assert.Equal(t, map[string]any{"endorsement": "X"}, worker["profile"],
		"a nested object without a rule of its own belongs to the worker")
	assert.NotContains(t, document, "withheldByAccess")
}

func TestRedactor_ReducesUsersAndTenantsWhereverTheySit(t *testing.T) {
	t.Parallel()

	fixture := &nestedPeopleFixture{}
	document := redact(t, &fakePermissions{allowed: true},
		agentParams(nil, permission.SensitivityRestricted), nestedPeopleRecord(fixture))

	assert.Equal(t, map[string]any{"id": fixture.orgID, "name": "Lakeshore Logistics"},
		document["organization"])
	assert.Equal(t, map[string]any{"id": fixture.buID, "name": "Midwest"},
		document["businessUnit"])
	assert.Equal(t, map[string]any{"id": fixture.userID, "name": "Avery Chen"},
		objectAt(t, document, "events", 0)["createdBy"])
	assert.Equal(t, "T-104", document["code"], "the record itself is untouched")
}

func TestRedactor_GatesWorkersAndCarrierAssignmentsBelowTheCeiling(t *testing.T) {
	t.Parallel()

	fixture := &nestedPeopleFixture{}
	document := redact(t, &fakePermissions{allowed: true},
		agentParams(nil, permission.SensitivityInternal), nestedPeopleRecord(fixture))

	worker := objectAt(t, document, "primaryWorker")
	assert.Equal(t, "Dana", worker["firstName"])
	assert.NotContains(t, worker, "email")
	assert.NotContains(t, worker, "phoneNumber")
	assert.NotContains(t, worker, "manager", "the manager is Restricted on the worker")

	carrier := objectAt(t, document, "carrierAssignment")
	assert.Equal(t, "Pat Carrier", carrier["externalDriverName"])
	assert.NotContains(t, carrier, "externalDriverPhone")

	assert.Equal(t, []any{
		"shipmentMove.externalDriverPhone",
		"worker.email",
		"worker.manager",
		"worker.phoneNumber",
		"worker.profile",
	}, document["withheldByAccess"])
}

func TestRedactor_ShowsTheExternalDriverPhoneAtRestricted(t *testing.T) {
	t.Parallel()

	fixture := &nestedPeopleFixture{}
	document := redact(t, &fakePermissions{allowed: true},
		agentParams(nil, permission.SensitivityRestricted), nestedPeopleRecord(fixture))

	carrier := objectAt(t, document, "carrierAssignment")
	assert.Equal(t, "312-555-0111", carrier["externalDriverPhone"])
	assert.NotContains(t, document, "withheldByAccess")
}

func TestNestedRules_ComeFromTheDomainsPrefixes(t *testing.T) {
	t.Parallel()

	prefixes := make(map[string]bool, len(nestedRules))
	for _, rule := range nestedRules {
		assert.False(t, prefixes[rule.idPrefix], "%s is ruled twice", rule.idPrefix)
		prefixes[rule.idPrefix] = true
	}

	for _, prefix := range []string{
		tenant.UserIDPrefix,
		tenant.OrganizationIDPrefix,
		tenant.BusinessUnitIDPrefix,
		agent.SubjectWorker.IDPrefix(),
		shipment.CarrierAssignmentIDPrefix,
	} {
		assert.True(t, prefixes[prefix], "%s has no rule", prefix)
	}
}
