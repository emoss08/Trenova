package agentquerytoolservice

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/partyquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func carrierCardAnswer(id string) map[string]*graphqlreadports.GraphQLReadResult {
	return map[string]*graphqlreadports.GraphQLReadResult{
		"DeskCarrierCard": {
			Data: map[string]any{"carrier": map[string]any{
				"id": id, "name": "Red River Freight", "dbaName": nil, "__typename": "Carrier",
				"insurancePolicies": []any{map[string]any{
					"id": "carins_1", "policyNumber": "CG-1182", "coverageAmount": "100000",
				}},
				"contacts": []any{},
			}},
			Withheld: []string{"carrier.phone", "carrier.insurancePolicies.coverageAmount"},
		},
	}
}

func TestGraphQLGet_ReadsTheCardByTheIDItsParameterNames(t *testing.T) {
	t.Parallel()

	carrierID := pulid.MustNew("car_").String()
	reader := &querytooltest.KeyedReader{Answers: carrierCardAnswer(carrierID)}
	spec := partyquerytools.CarrierGetSpec()
	tool, err := gettool.NewGraphQL(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{"carrierId": carrierID}),
	)
	require.NoError(t, err)

	sent := reader.Sent("DeskCarrierCard")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{"id": carrierID}, sent.Variables)
	record, ok := data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Red River Freight", record["name"])
	assert.NotContains(t, record, "dbaName", "a null is never carried")
	assert.NotContains(t, record, "__typename")
	assert.NotContains(t, record, "contacts", "an empty list is never carried")
	policies, _ := record["insurancePolicies"].([]any)
	require.Len(t, policies, 1)
	assert.Equal(t, []string{"phone", "insurancePolicies.coverageAmount"},
		record[graphqlread.OutcomeWithheld])
	assert.Equal(t, fieldgate.WithheldNote, record[graphqlread.OutcomeWithheldNote])
}

// A pro number or a paraphrased id pasted where an id belongs is refused
// before anything is read, rather than read and reported missing.
func TestGraphQLGet_RefusesWhatIsNotAnID(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{}
	spec := partyquerytools.CarrierGetSpec()
	tool, err := gettool.NewGraphQL(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	_, err = tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{"carrierId": "PRO-48812"}),
	)
	require.Error(t, err)
	assert.Empty(t, reader.Requests)
}

func TestGraphQLGet_AReadTheServerRefusesIsARefusal(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskLocationCard": {Errors: []graphqlreadports.GraphQLReadError{{
			Code: graphqlreadports.GraphQLCodeForbidden, Message: "not allowed",
		}}},
	}}
	spec := partyquerytools.LocationGetSpec()
	tool, err := gettool.NewGraphQL(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	_, err = tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		partyquerytools.LocationIDParam: pulid.MustNew("loc_").String(),
	}))
	var refused *graphqlreadports.ReadRefusedError
	require.ErrorAs(t, err, &refused)
	assert.Equal(t, permission.ResourceLocation, refused.Resource)
}

func TestGraphQLGet_SpecsMatchWhatTheGeneratorProved(t *testing.T) {
	t.Parallel()

	for _, spec := range graphQLGetSpecs() {
		read, ok := graphqlread.Reads[spec.Name]
		require.True(t, ok, spec.Name)
		assert.Equal(t, graphqlread.VerbGet, read.Verb, spec.Name)
		if read.SelfScoped {
			assert.Equal(t, agenttooltrust.ToolScopeSelf, spec.Scope, spec.Name)
			continue
		}
		assert.Equal(t, string(spec.Resource), read.Resource, spec.Name)
	}

	spec := partyquerytools.CarrierGetSpec()
	spec.Name = "carrier_list"
	_, err := gettool.NewGraphQL(&spec, &querytooltest.KeyedReader{}, nil)
	require.ErrorContains(t, err, "not a get")
}

// What each summary promises, the card's operation selects.
func TestGraphQLGet_TheCardSelectsWhatTheSummaryPromises(t *testing.T) {
	t.Parallel()

	selected := func(tool string) []string {
		var names []string
		for _, field := range graphqlread.Reads[tool].Fields {
			if field.Selected {
				names = append(names, field.Name)
			}
		}
		return names
	}
	for tool, want := range map[string][]string{
		"carrier_get":           {"insurancePolicies", "contacts", "complianceStatus", "dotNumber", "mcNumber"},
		"customer_get":          {"billingProfile", "addressLine1", "dotNumber"},
		"location_get":          {"latitude", "longitude", "geofenceType", "timezone", "locationCategory"},
		"equipment_get_tractor": {"primaryWorker", "fleetCode", "equipmentType", "leaseEndDate"},
		"equipment_get_trailer": {"fleetCode", "equipmentType", "maxLoadWeight", "leaseEndDate"},
		"carrier_get_intelligence_event": {
			"severity", "priorValue", "currentValue", "acknowledgedAt", "resolvedAt",
		},
		"edi_get_partner":      {"partner", "customer", "readiness", "transports"},
		"edi_get_inbound_file": {"file", "messages", "raw"},
		"edi_get_message":      {"message", "decoded", "diagnostics", "raw"},
		"edi_get_transfer":     {"transfer", "unresolvedMappings", "canAccept"},
		"edi_get_ack_health": {
			"ackStatusCounts", "partners", "overdueByPartner", "recentDeadLettered",
		},
	} {
		names := selected(tool)
		for _, field := range want {
			assert.True(t, slices.Contains(names, field), "%s selects %s", tool, field)
		}
	}
}
