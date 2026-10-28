package agentquerytoolservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/pagetest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// jurisdictionRows is the global jurisdiction list, filtered as the
// repository filters it, recording what each read asked for.
type jurisdictionRows struct {
	iftaports.IFTARepository

	asked []*iftaports.ListJurisdictionsRequest
}

func (r *jurisdictionRows) ListJurisdictions(
	_ context.Context,
	req *iftaports.ListJurisdictionsRequest,
) ([]*ifta.Jurisdiction, error) {
	r.asked = append(r.asked, req)
	all := []*ifta.Jurisdiction{
		{ID: pulid.MustNew("ifj_"), CountryCode: "US", Code: "TX", Name: "Texas",
			IsIftaMember: true, Status: ifta.JurisdictionStatusActive},
		{ID: pulid.MustNew("ifj_"), CountryCode: "CA", Code: "ON", Name: "Ontario",
			IsIftaMember: true, Status: ifta.JurisdictionStatusActive},
		{ID: pulid.MustNew("ifj_"), CountryCode: "US", Code: "ZZ", Name: "Retired",
			IsIftaMember: true, Status: ifta.JurisdictionStatusInactive},
	}
	out := make([]*ifta.Jurisdiction, 0, len(all))
	for _, row := range all {
		if req.CountryCode != "" && row.CountryCode != req.CountryCode {
			continue
		}
		if len(req.Statuses) > 0 && row.Status != req.Statuses[0] {
			continue
		}
		out = append(out, row)
	}

	return out, nil
}

func readJurisdictions(
	t *testing.T,
	repo *jurisdictionRows,
	args map[string]any,
) map[string]any {
	t.Helper()

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: engine,
		Resolvers: agentread.ResolverParams{
			IFTAService: iftaservice.New(iftaservice.Params{Logger: zap.NewNop(), Repo: repo}),
		},
	})
	tool, err := agentquerytoolservice.ConvertedListTool(
		"fuel_list_ifta_jurisdictions",
		reader,
		engine,
	)
	require.NoError(t, err)
	person := agentreadtest.Person()
	data, err := tool.Query(readscope.With(t.Context(), nil), &agenttoolports.QueryToolParams{
		OrganizationID: person.OrganizationID,
		BusinessUnitID: person.BusinessUnitID,
		Actor:          person,
		Timezone:       "America/Chicago",
		Params:         args,
		RunID:          pulid.MustNew("ar_"),
	})
	require.NoError(t, err)

	return asMap(t, data)
}

func jurisdictionCodes(outcome map[string]any) []string {
	items, _ := outcome["items"].([]any)
	codes := make([]string, 0, len(items))
	for _, item := range items {
		codes = append(codes, item.(map[string]any)["code"].(string))
	}

	return codes
}

// The jurisdiction list reads the active ones unless the call asks for the
// retired ones too, as the repository tool always read them, and narrows by
// country on the server.
func TestIFTAJurisdictions_ReadTheActiveOnesByCountry(t *testing.T) {
	t.Parallel()

	repo := &jurisdictionRows{}
	outcome := readJurisdictions(t, repo, map[string]any{"countryCode": "US"})
	assert.Equal(t, []string{"TX"}, jurisdictionCodes(outcome))
	require.Len(t, repo.asked, 1)
	assert.Equal(t, "US", repo.asked[0].CountryCode)
	assert.Equal(
		t,
		[]ifta.JurisdictionStatus{ifta.JurisdictionStatusActive},
		repo.asked[0].Statuses,
	)

	every := readJurisdictions(t, &jurisdictionRows{}, map[string]any{"activeOnly": false})
	assert.ElementsMatch(t, []string{"TX", "ON", "ZZ"}, jurisdictionCodes(every))
}

// purchaseRows answers the purchase list with one purchase and records the
// request the resolver built.
type purchaseRows struct {
	fuelpurchaseports.FuelPurchaseRepository

	asked *fuelpurchaseports.ListFuelPurchasesRequest
}

func (r *purchaseRows) ListPurchases(
	_ context.Context,
	req *fuelpurchaseports.ListFuelPurchasesRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error) {
	r.asked = req

	unit := &tractor.Tractor{ID: pulid.MustNew("tr_"), Code: "TRC-001"}
	texas := &ifta.Jurisdiction{ID: pulid.MustNew("ifj_"), CountryCode: "US", Code: "TX"}

	return pagetest.Of([]*fuelpurchase.FuelPurchase{{
		ID: pulid.MustNew("fpur_"), TractorID: unit.ID, Tractor: unit,
		JurisdictionID: texas.ID, Jurisdiction: texas, FuelType: domaintypes.IFTAFuelTypeDiesel,
		Source: fuelpurchase.PurchaseSourceManual, TaxPaid: true, CurrencyCode: "USD",
		TotalAmountMinor: 38_990,
	}}, 25), nil
}

// The purchase list's typed parameters reach the screen's resolver: a
// record field as a field filter, the screen's own filters on its input,
// and the tractor, jurisdiction and card the row names come with it.
func TestFuelPurchases_TypedFiltersReachTheResolver(t *testing.T) {
	t.Parallel()

	repo := &purchaseRows{}
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: engine,
		Resolvers: agentread.ResolverParams{
			FuelPurchaseService: fuelpurchaseservice.New(fuelpurchaseservice.Params{
				Logger: zap.NewNop(), Repo: repo,
			}),
		},
	})
	tool, err := agentquerytoolservice.ConvertedListTool("fuel_list_purchases", reader, engine)
	require.NoError(t, err)
	jurisdiction := pulid.MustNew("ifj_").String()
	person := agentreadtest.Person()
	data, err := tool.Query(readscope.With(t.Context(), nil), &agenttoolports.QueryToolParams{
		OrganizationID: person.OrganizationID,
		BusinessUnitID: person.BusinessUnitID,
		Actor:          person,
		Timezone:       "America/Chicago",
		Params: map[string]any{
			"taxPaid": true, "jurisdictionId": jurisdiction, "sources": []any{"Manual"},
		},
		RunID: pulid.MustNew("ar_"),
	})
	require.NoError(t, err)

	require.NotNil(t, repo.asked)
	assert.True(t, repo.asked.IncludeTractor)
	assert.True(t, repo.asked.IncludeJurisdiction)
	assert.Equal(t, []fuelpurchase.PurchaseSource{fuelpurchase.PurchaseSourceManual},
		repo.asked.Sources)
	filters := make(map[string]dbtype.Operator)
	for _, filter := range repo.asked.Filter.FieldFilters {
		filters[filter.Field] = filter.Operator
	}
	assert.Equal(t, dbtype.OpEqual, filters["taxPaid"])
	assert.Equal(t, dbtype.OpEqual, filters["jurisdictionId"])
	items, _ := asMap(t, data)["items"].([]any)
	require.Len(t, items, 1)
	row := items[0].(map[string]any)
	assert.Equal(t, "389.90", row["totalAmount"])
	assert.Equal(t, "TRC-001", row["tractor"].(map[string]any)["code"])
	assert.Equal(t, "TX", row["jurisdiction"].(map[string]any)["code"])
}
