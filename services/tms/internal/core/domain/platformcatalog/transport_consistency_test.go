package platformcatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type transportPair struct {
	routePrefix string
	source      GraphQLSource
}

func domainTransportPairs() []transportPair {
	return []transportPair{
		{routePrefix: "/api/v1/accessorial-charges/", source: "accessorial_charge.graphqls"},
		{routePrefix: "/api/v1/account-types/", source: "account_type.graphqls"},
		{routePrefix: "/api/v1/agent-runs/", source: "agent.graphqls"},
		{routePrefix: "/api/v1/api-keys/", source: "api_key.graphqls"},
		{routePrefix: "/api/v1/billing-queue/", source: "billing_queue.graphqls"},
		{routePrefix: "/api/v1/carriers/", source: "carrier.graphqls"},
		{routePrefix: "/api/v1/commodities/", source: "commodity.graphqls"},
		{routePrefix: "/api/v1/customers/", source: "customer.graphqls"},
		{routePrefix: "/api/v1/detention/", source: "detention.graphqls"},
		{routePrefix: "/api/v1/distance-overrides/", source: "distance_override.graphqls"},
		{routePrefix: "/api/v1/distance-profiles/", source: "distance_profile.graphqls"},
		{routePrefix: "/api/v1/document-packet-rules/", source: "document_packet_rule.graphqls"},
		{routePrefix: "/api/v1/document-templates/", source: "document_template.graphqls"},
		{routePrefix: "/api/v1/document-types/", source: "document_type.graphqls"},
		{routePrefix: "/api/v1/edi/", source: "edi.graphqls"},
		{routePrefix: "/api/v1/email-profiles/", source: "email_profile.graphqls"},
		{routePrefix: "/api/v1/equipment-manufacturers/", source: "equipment_manufacturer.graphqls"},
		{routePrefix: "/api/v1/equipment-types/", source: "equipment_type.graphqls"},
		{routePrefix: "/api/v1/fiscal-periods/", source: "fiscal_period.graphqls"},
		{routePrefix: "/api/v1/fiscal-years/", source: "fiscal_year.graphqls"},
		{routePrefix: "/api/v1/fleet-codes/", source: "fleet_code.graphqls"},
		{routePrefix: "/api/v1/formula-templates/", source: "formula_template.graphqls"},
		{routePrefix: "/api/v1/hazardous-materials/", source: "hazardous_material.graphqls"},
		{
			routePrefix: "/api/v1/hazmat-segregation-rules/",
			source:      "hazmat_segregation_rule.graphqls",
		},
		{routePrefix: "/api/v1/hold-reasons/", source: "hold_reason.graphqls"},
		{routePrefix: "/api/v1/jurisdiction-rules/", source: "jurisdiction_rule.graphqls"},
		{routePrefix: "/api/v1/location-categories/", source: "location_category.graphqls"},
		{routePrefix: "/api/v1/locations/", source: "location.graphqls"},
		{routePrefix: "/api/v1/orders/", source: "order.graphqls"},
		{routePrefix: "/api/v1/portal/", source: "driver_portal.graphqls"},
		{routePrefix: "/api/v1/rate-agreements/", source: "rate.graphqls"},
		{routePrefix: "/api/v1/recurring-shipments/", source: "recurring_shipment.graphqls"},
		{routePrefix: "/api/v1/reports/", source: "report.graphqls"},
		{routePrefix: "/api/v1/roles/", source: "role.graphqls"},
		{
			routePrefix: "/api/v1/service-failure-reason-codes/",
			source:      "service_failure_reason_code.graphqls",
		},
		{routePrefix: "/api/v1/service-failures/", source: "service_failure.graphqls"},
		{routePrefix: "/api/v1/service-types/", source: "service_type.graphqls"},
		{routePrefix: "/api/v1/shipment-types/", source: "shipment_type.graphqls"},
		{routePrefix: "/api/v1/shipments/", source: "shipment.graphqls"},
		{routePrefix: "/api/v1/stored-mileages/", source: "stored_mileage.graphqls"},
		{routePrefix: "/api/v1/tca/", source: "table_change_alert.graphqls"},
		{routePrefix: "/api/v1/tenders/", source: "tender.graphqls"},
		{routePrefix: "/api/v1/tractors/", source: "tractor.graphqls"},
		{routePrefix: "/api/v1/trailers/", source: "trailer.graphqls"},
		{routePrefix: "/api/v1/users/", source: "user.graphqls"},
		{routePrefix: "/api/v1/workers/", source: "worker.graphqls"},
	}
}

func TestDomainsResolveToTheSameFeatureOnBothTransports(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)
	prefixes := protectedProductRoutePrefixes()

	for _, pair := range domainTransportPairs() {
		t.Run(pair.routePrefix, func(t *testing.T) {
			t.Parallel()

			restFeature, ok := longestPrefixOwner(prefixes, pair.routePrefix)
			require.Truef(t, ok, "no REST route prefix owns %s", pair.routePrefix)

			policy := registry.PolicyForGraphQLRootField(
				GraphQLOperationQuery,
				"unmappedRootField",
				pair.source,
			)
			require.Equalf(
				t,
				RouteAccessClassProduct,
				policy.AccessClass,
				"GraphQL source %s is not owned by a feature", pair.source,
			)

			require.Equalf(
				t,
				restFeature,
				policy.FeatureKey,
				"domain is sold as %q over REST (%s) but %q over GraphQL (%s)",
				restFeature, pair.routePrefix, policy.FeatureKey, pair.source,
			)
		})
	}
}

func TestTransportPairsReferenceKnownSources(t *testing.T) {
	t.Parallel()

	owned := make(map[GraphQLSource]struct{})
	for _, feature := range NewStaticProvider().Features() {
		for _, source := range feature.GraphQLSources {
			owned[source] = struct{}{}
		}
	}

	for _, pair := range domainTransportPairs() {
		t.Run(string(pair.source), func(t *testing.T) {
			_, ok := owned[pair.source]
			require.Truef(t, ok, "GraphQL source %s is not owned by any feature", pair.source)
		})
	}
}
