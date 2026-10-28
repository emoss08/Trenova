package detectors_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/detector"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/detectors"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

// everyDetectorFinds returns one row per query that crosses every detector's
// threshold, so each detector produces a finding to inspect.
func everyDetectorFinds() *fakeMetrics {
	return &fakeMetrics{
		onTime: []insightports.CustomerOnTimeRow{{
			CustomerID:    pulid.MustNew("cus_"),
			CustomerName:  "Acme Foods",
			CurrentTotal:  100,
			CurrentOnTime: 82,
			PriorTotal:    100,
			PriorOnTime:   95,
		}},
		unbilled: []insightports.UnbilledShipmentRow{{
			CustomerID:     pulid.MustNew("cus_"),
			CustomerName:   "Borden Freight",
			ShipmentCount:  14,
			TotalAmount:    "31240.00",
			OldestDelivery: windowEnd - 21*86400,
		}},
		detention: []insightports.UnbilledDetentionRow{{
			LocationID:        pulid.MustNew("loc_"),
			LocationName:      "Dallas Distribution Center",
			OccurrenceCount:   19,
			UnbilledAmount:    "9840.00",
			TotalDwellMinutes: 5400,
		}},
		emptyMiles: []insightports.CustomerEmptyMilesRow{{
			CustomerID:    pulid.MustNew("cus_"),
			CustomerName:  "Northwind Retail",
			EmptyMiles:    "28000",
			TotalMiles:    "100000",
			MoveCount:     140,
			OrgEmptyMiles: "120000",
			OrgTotalMiles: "1000000",
		}},
		credential: []insightports.ExpiringCredentialRow{{
			CredentialTypeID:   pulid.MustNew("wct_"),
			CredentialTypeName: "DOT Medical Card",
			IsRequired:         true,
			WorkerCount:        11,
			EarliestExpiry:     windowEnd + 6*86400,
		}},
	}
}

// A reader is checked for every kind a detector declares before any of its
// findings reach them, so a finding that names a customer from a detector
// declaring only shipments would show that customer to someone who cannot
// read customers.
func TestEveryDetectorDeclaresEveryKindItsFindingsName(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	for _, d := range detectors.All(everyDetectorFinds()) {
		declared := d.Resources()
		require.NotEmptyf(t, declared, "%s declares no kind of record", d.Key())
		for _, kind := range declared {
			def, ok := registry.Get(kind.String())
			require.Truef(t, ok, "%s declares %q, which is not a registered kind", d.Key(), kind)
			require.Truef(
				t,
				slices.ContainsFunc(
					def.Operations,
					func(op permissioncatalog.OperationDefinition) bool {
						return op.Operation == d.Operation()
					},
				),
				"%s needs %s on %q, which the kind does not offer",
				d.Key(),
				d.Operation(),
				kind,
			)
		}

		findings, err := d.Detect(t.Context(), testParams())
		require.NoError(t, err)
		require.NotEmptyf(t, findings, "%s produced no finding to inspect", d.Key())
		for _, finding := range findings {
			require.NoError(t, finding.Validate(), finding.DedupeKey)
			require.Emptyf(t, detector.Undeclared(d, &finding),
				"%s names kinds it does not declare in Resources", d.Key())
		}
	}
}
