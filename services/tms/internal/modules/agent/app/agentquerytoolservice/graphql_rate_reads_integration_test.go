//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/fuelsurchargerepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/rateagreementrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratematrixrepository"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	rateAgreementsPerTenant = 5
	fuelPricesPerTenant     = 4
)

// perMileTariff prices a lane by its distance at the rate its inputs give.
const perMileTariff = "tariff \"Per Mile\"\n\ninput rate: Money/mi\n\n" +
	"charge \"Per Mile\" (quantity: shipment.distance, rate: rate)\n"

// rateTenant is what one tenant holds for the rate reads: its agreements (the
// last one a draft), the lanes of the first, a fuel index with its prices and
// an active program priced by it.
type rateTenant struct {
	ctx        context.Context
	info       pagination.TenantInfo
	agreements []string
	drafts     []string
	lanes      []string
	index      pulid.ID
	prices     []string
	program    string
}

//nolint:funlen // one tenant's rate records, written straight through the admin connection
func seedRateTenant(t *testing.T, admin *bun.DB, tenant *agentreadtest.Tenant, label string) *rateTenant {
	t.Helper()

	ctx := t.Context()
	bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
	insert := func(model any) {
		t.Helper()
		_, err := admin.NewInsert().Model(model).Exec(ctx)
		require.NoError(t, err)
	}
	out := &rateTenant{
		ctx:  readscope.With(dbscope.WithTenant(ctx, tenant.Scope()), nil),
		info: pagination.TenantInfo{OrgID: org, BuID: bu, UserID: tenant.Actor.UserID},
	}

	party := &customer.Customer{
		ID: pulid.MustNew("cus_"), OrganizationID: org, BusinessUnitID: bu, StateID: tenant.StateID,
		Code: "RC" + label, Name: "Rated " + label, AddressLine1: "1 Lane Way", City: "Atlanta",
		PostalCode: "30303",
	}
	insert(party)
	template := &formulatemplate.FormulaTemplate{
		ID: pulid.MustNew("ft_"), OrganizationID: org, BusinessUnitID: bu, Name: "Per Mile " + label,
		Source: perMileTariff, Status: formulatemplate.StatusActive,
	}
	insert(template)
	for idx := range rateAgreementsPerTenant {
		status := rateagreement.StatusActive
		if idx == rateAgreementsPerTenant-1 {
			status = rateagreement.StatusDraft
		}
		agreement := &rateagreement.RateAgreement{
			ID: pulid.MustNew("rag_"), OrganizationID: org, BusinessUnitID: bu,
			PartyType: rateagreement.PartyTypeCustomer, CustomerID: &party.ID,
			Code: fmt.Sprintf("RA%s%d", label, idx), Name: fmt.Sprintf("Agreement %s%d", label, idx),
			AgreementType: rateagreement.AgreementTypeContract, Status: status, Currency: "USD",
			EffectiveFrom: 100,
			Inputs: tarifftypes.Inputs{
				"rate": rules.QuantityValue(decimal.RequireFromString("2.10"), "$/mi"),
			},
		}
		insert(agreement)
		out.agreements = append(out.agreements, agreement.ID.String())
		if status == rateagreement.StatusDraft {
			out.drafts = append(out.drafts, agreement.ID.String())
		}
		if idx != 0 {
			continue
		}
		for _, destination := range []string{"FL", "TN"} {
			rule := &rateagreement.RateAgreementRule{
				ID: pulid.MustNew("ragr_"), OrganizationID: org, BusinessUnitID: bu,
				RateAgreementID: agreement.ID, PartyType: rateagreement.PartyTypeCustomer,
				PartyID: party.ID, Status: rateagreement.RuleStatusActive,
				OriginScopeType: "State", OriginScopeValue: "GA",
				DestinationScopeType: "State", DestinationScopeValue: destination,
				Direction: rateagreement.DirectionDirectional, FormulaTemplateID: template.ID,
				Inputs: tarifftypes.Inputs{
					"rate": rules.QuantityValue(decimal.RequireFromString("2.00"), "$/mi"),
				},
				EffectiveFrom: 100,
			}
			require.NoError(t, rateagreement.ApplyLanes(ctx, []*rateagreement.RateAgreementRule{rule}))
			insert(rule)
			out.lanes = append(out.lanes, rule.ID.String())
		}
	}

	index := &fuelsurcharge.FuelIndex{
		ID: pulid.MustNew("fidx_"), OrganizationID: org, BusinessUnitID: bu, Name: "Diesel " + label,
		Code: "DSL" + label, Source: fuelsurcharge.IndexSourceCustom, FuelType: fuelsurcharge.FuelTypeDiesel,
		Currency: "USD", IsActive: true,
	}
	insert(index)
	out.index = index.ID
	today := time.Now().UTC()
	for idx := range fuelPricesPerTenant {
		price := &fuelsurcharge.FuelIndexPrice{
			ID: pulid.MustNew("fip_"), OrganizationID: org, BusinessUnitID: bu, FuelIndexID: index.ID,
			PriceDate: today.AddDate(0, 0, -7*idx).Format(fuelsurcharge.PriceDateLayout),
			Price:     decimal.RequireFromString("3.50"), Currency: "USD", IsManual: true,
		}
		insert(price)
		out.prices = append(out.prices, price.ID.String())
	}
	charge := &accessorialcharge.AccessorialCharge{
		ID: pulid.MustNew("acc_"), OrganizationID: org, BusinessUnitID: bu, Code: "FSC" + label,
		Description: "Fuel", Method: accessorialcharge.MethodPerUnit,
		RateUnit: accessorialcharge.RateUnitMile,
	}
	insert(charge)
	program := &fuelsurcharge.FuelSurchargeProgram{
		ID: pulid.MustNew("fsp_"), OrganizationID: org, BusinessUnitID: bu, Name: "Program " + label,
		Code: "PRG" + label, Status: fuelsurcharge.ProgramStatusActive, FuelIndexID: index.ID,
		AccessorialChargeID: charge.ID, Method: fuelsurcharge.ProgramMethodPerMileStep,
		PegPrice:      decimal.NewNullDecimal(decimal.RequireFromString("1.20")),
		Increment:     decimal.NewNullDecimal(decimal.RequireFromString("0.05")),
		IncrementRate: decimal.NewNullDecimal(decimal.RequireFromString("0.01")),
	}
	insert(program)
	out.program = program.ID.String()

	return out
}

func fuelProgramIDs(page map[string]any) []string {
	items, _ := page["items"].([]any)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		program, _ := item.(map[string]any)["program"].(map[string]any)
		ids = append(ids, program["id"].(string))
	}

	return ids
}

//nolint:funlen // the four converted rate reads against one database
const seededMatrixCells = 150

// seedRateMatrix writes a matrix of two axes and more cells than one read
// returns, straight through the admin connection.
func seedRateMatrix(t *testing.T, admin *bun.DB, tenant *agentreadtest.Tenant, label string) string {
	t.Helper()

	ctx := t.Context()
	bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
	matrix := &ratematrix.RateMatrix{
		ID: pulid.MustNew("rmx_"), OrganizationID: org, BusinessUnitID: bu,
		Code: "LTL-" + label, Name: "LTL " + label, Status: domaintypes.StatusActive,
		ValueType: ratematrix.ValueTypeMoney,
	}
	_, err := admin.NewInsert().Model(matrix).Exec(ctx)
	require.NoError(t, err)
	for position, kind := range []ratematrix.DimensionKind{
		ratematrix.DimensionKindZone, ratematrix.DimensionKindWeightBreak,
	} {
		_, err = admin.NewInsert().Model(&ratematrix.RateMatrixDimension{
			ID: pulid.MustNew("rmd_"), OrganizationID: org, BusinessUnitID: bu,
			RateMatrixID: matrix.ID, Position: int16(position), Kind: kind,
			MatchMode: ratematrix.MatchModeExact,
		}).Exec(ctx)
		require.NoError(t, err)
	}
	cells := make([]*ratematrix.RateMatrixCell, 0, seededMatrixCells)
	for idx := range seededMatrixCells {
		cells = append(cells, &ratematrix.RateMatrixCell{
			ID: pulid.MustNew("rmc_"), OrganizationID: org, BusinessUnitID: bu,
			RateMatrixID: matrix.ID, D0Key: fmt.Sprintf("Z%03d", idx),
			D1Min: decimal.NewNullDecimal(decimal.Zero),
			D1Max: decimal.NewNullDecimal(decimal.NewFromInt(500)),
			Value: decimal.RequireFromString("2.15"),
		})
	}
	_, err = admin.NewInsert().Model(&cells).Exec(ctx)
	require.NoError(t, err)

	return matrix.ID.String()
}

func TestConvertedLists_RateReadsReadWhatTheirServicesReadUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	_, resolvers := convertedListCases(conn, admin)
	agreements := rateagreementservice.New(rateagreementservice.Params{
		Logger: logger,
		Repo:   rateagreementrepository.New(rateagreementrepository.Params{DB: conn, Logger: logger}),
	})
	indexes := fuelsurchargerepository.NewIndexRepository(
		fuelsurchargerepository.IndexParams{DB: conn, Logger: logger})
	prices := fuelsurchargerepository.NewPriceRepository(
		fuelsurchargerepository.PriceParams{DB: conn, Logger: logger})
	programs := fuelsurchargerepository.NewProgramRepository(
		fuelsurchargerepository.ProgramParams{DB: conn, Logger: logger})
	fuel := fuelsurchargeservice.New(fuelsurchargeservice.Params{
		Logger: logger, IndexRepo: indexes, PriceRepo: prices, ProgramRepo: programs,
	})
	resolvers.RateAgreementService = agreements
	resolvers.FuelIndexRepo = indexes
	resolvers.FuelIndexPriceRepo = prices
	resolvers.FuelSurchargeProgramRepo = programs
	resolvers.FuelSurchargeService = fuel
	resolvers.RateMatrixService = ratematrixservice.New(ratematrixservice.Params{
		Logger: logger,
		Repo:   ratematrixrepository.New(ratematrixrepository.Params{DB: conn, Logger: logger}),
	})

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: resolvers})
	public := agentreadtest.NewReader(t, agentreadtest.Options{Engine: &publicRole{}, Resolvers: resolvers})
	mineTenant := agentreadtest.SeedTenant(t, admin, "A")
	theirsTenant := agentreadtest.SeedTenant(t, admin, "B")
	mine := seedRateTenant(t, admin, mineTenant, "A")
	theirs := seedRateTenant(t, admin, theirsTenant, "B")
	actor := mineTenant.Actor

	list := func(t *testing.T, name string, engineReader bool) agentports.AgentQueryTool {
		t.Helper()
		var tool agentports.AgentQueryTool
		var err error
		if engineReader {
			tool, err = agentquerytoolservice.ConvertedListTool(name, reader, engine)
		} else {
			tool, err = agentquerytoolservice.ConvertedListTool(name, public, &publicRole{})
		}
		require.NoError(t, err)
		return tool
	}

	t.Run("rate_list_agreements", func(t *testing.T) {
		tool := list(t, "rate_list_agreements", true)

		ids, _ := walk(t, mine.ctx, tool, actor, map[string]any{})
		assert.ElementsMatch(t, mine.agreements, ids, "every page once, and none of the other tenant's")

		drafts := callTool(t, mine.ctx, tool, actor, map[string]any{"status": []any{"Draft"}})
		before, err := agreements.List(mine.ctx, &rateagreementports.ListRateAgreementRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: mine.info, Pagination: pagination.Info{Limit: 50},
			},
			Status: rateagreement.StatusDraft,
		})
		require.NoError(t, err)
		want := make([]string, 0, len(before.Items))
		for _, row := range before.Items {
			want = append(want, row.ID.String())
		}
		assert.ElementsMatch(t, want, itemIDs(drafts))
		assert.ElementsMatch(t, mine.drafts, itemIDs(drafts))

		hidden := callTool(t, mine.ctx, list(t, "rate_list_agreements", false), actor, map[string]any{})
		withheld, _ := hidden["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		items, _ := hidden["items"].([]any)
		for _, item := range items {
			for _, field := range withheld {
				assert.NotContains(t, item, field)
			}
		}
	})

	t.Run("rate_get_agreement", func(t *testing.T) {
		get, err := agentquerytoolservice.ConvertedGetTool("rate_get_agreement", reader, engine)
		require.NoError(t, err)

		for _, id := range mine.agreements {
			record := callTool(t, mine.ctx, get, actor, map[string]any{"rateAgreementId": id})
			assert.Equal(t, id, record["id"])
		}
		card := callTool(t, mine.ctx, get, actor, map[string]any{"rateAgreementId": mine.agreements[0]})
		page, err := agreements.ListLanes(mine.ctx, &rateagreementports.ListRateAgreementLanesRequest{
			TenantInfo: mine.info, RateAgreementID: pulid.ID(mine.agreements[0]), Limit: 50,
		})
		require.NoError(t, err)
		assert.EqualValues(t, page.Count, card["laneCount"])
		lanes, _ := card["lanes"].([]any)
		laneIDs := make([]string, 0, len(lanes))
		for _, lane := range lanes {
			laneIDs = append(laneIDs, lane.(map[string]any)["id"].(string))
		}
		assert.ElementsMatch(t, mine.lanes, laneIDs)

		_, err = get.Query(mine.ctx, &agentports.QueryToolParams{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			Actor:          actor,
			Timezone:       "America/Chicago",
			Params:         map[string]any{"rateAgreementId": theirs.agreements[0]},
			RunID:          pulid.MustNew("ar_"),
		})
		require.Error(t, err, "another tenant's agreement is not found")

		hiddenGet, err := agentquerytoolservice.ConvertedGetTool("rate_get_agreement", public, &publicRole{})
		require.NoError(t, err)
		hidden := callTool(t, mine.ctx, hiddenGet, actor, map[string]any{"rateAgreementId": mine.agreements[0]})
		withheld, _ := hidden["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		for _, field := range withheld {
			assert.NotContains(t, hidden, field)
		}
	})

	t.Run("rate_get_matrix", func(t *testing.T) {
		ours := seedRateMatrix(t, admin, mineTenant, "A")
		other := seedRateMatrix(t, admin, theirsTenant, "B")
		get, err := agentquerytoolservice.ConvertedGetTool("rate_get_matrix", reader, engine)
		require.NoError(t, err)

		first := callTool(t, mine.ctx, get, actor, map[string]any{"rateMatrixId": ours})
		assert.Equal(t, ours, first["id"])
		assert.EqualValues(t, seededMatrixCells, first["cellCount"])
		cells, _ := first["cells"].([]any)
		require.Len(t, cells, 100)
		assert.Equal(t, true, first["cellsHasMore"])
		assert.EqualValues(t, 100, first["nextCellOffset"])
		cell, _ := cells[0].(map[string]any)
		assert.Len(t, cell["keys"], 2, "a cell's keys run along the matrix's two axes")
		assert.Equal(t, []any{"", "0 to 500"}, cell["ranges"])

		rest := callTool(t, mine.ctx, get, actor, map[string]any{
			"rateMatrixId": ours, "cellOffset": first["nextCellOffset"],
		})
		cells, _ = rest["cells"].([]any)
		assert.Len(t, cells, seededMatrixCells-100)
		assert.Equal(t, false, rest["cellsHasMore"])
		assert.NotContains(t, rest, "nextCellOffset")

		_, err = get.Query(mine.ctx, &agentports.QueryToolParams{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			Actor:          actor,
			Timezone:       "America/Chicago",
			Params:         map[string]any{"rateMatrixId": other},
			RunID:          pulid.MustNew("ar_"),
		})
		require.Error(t, err, "another tenant's matrix is not found")

		internal, err := get.Query(mine.ctx, &agentports.QueryToolParams{
			OrganizationID: actor.OrganizationID, BusinessUnitID: actor.BusinessUnitID,
			Actor: actor, Timezone: "America/Chicago", DataAccessCeiling: permission.SensitivityInternal,
			Params: map[string]any{"rateMatrixId": ours}, RunID: pulid.MustNew("ar_"),
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, []any{"cells.value", "cells.minCharge"},
			asMap(t, internal)["withheldByAccess"],
			"a cell's value and minimum stay Restricted, as the repository tool gated them")

		hiddenGet, err := agentquerytoolservice.ConvertedGetTool("rate_get_matrix", public, &publicRole{})
		require.NoError(t, err)
		hidden := callTool(t, mine.ctx, hiddenGet, actor, map[string]any{"rateMatrixId": ours})
		withheld, _ := hidden["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		for _, field := range withheld {
			assert.NotContains(t, hidden, field)
		}
	})

	t.Run("rate_list_fuel_index_prices", func(t *testing.T) {
		tool := list(t, "rate_list_fuel_index_prices", true)

		page := callTool(t, mine.ctx, tool, actor, map[string]any{"fuelIndexId": mine.index.String()})
		before, err := prices.ListByIndex(mine.ctx, &fuelsurchargeports.ListFuelIndexPricesRequest{
			FuelIndexID: mine.index, TenantInfo: mine.info, Limit: 25,
		})
		require.NoError(t, err)
		want := make([]string, 0, len(before))
		for _, row := range before {
			want = append(want, row.ID.String())
		}
		assert.Equal(t, want, itemIDs(page), "newest first, as the repository tool read them")
		assert.ElementsMatch(t, mine.prices, itemIDs(page))

		since := time.Now().UTC().AddDate(0, 0, -8).Format(fuelsurcharge.PriceDateLayout)
		recent := callTool(t, mine.ctx, tool, actor, map[string]any{
			"fuelIndexId": mine.index.String(), "from": since,
		})
		assert.ElementsMatch(t, mine.prices[:2], itemIDs(recent))

		other := callTool(t, mine.ctx, tool, actor, map[string]any{"fuelIndexId": theirs.index.String()})
		assert.Empty(t, itemIDs(other), "another tenant's index reads nothing")

		hidden := callTool(t, mine.ctx, list(t, "rate_list_fuel_index_prices", false), actor,
			map[string]any{"fuelIndexId": mine.index.String()})
		withheld, _ := hidden["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		items, _ := hidden["items"].([]any)
		for _, item := range items {
			for _, field := range withheld {
				assert.NotContains(t, item, field)
			}
		}
	})

	t.Run("rate_list_fuel_surcharge_programs", func(t *testing.T) {
		tool := list(t, "rate_list_fuel_surcharge_programs", true)

		page := callTool(t, mine.ctx, tool, actor, map[string]any{})
		before, err := fuel.ProgramCurrentRates(mine.ctx, mine.info)
		require.NoError(t, err)
		want := make([]string, 0, len(before))
		for _, row := range before {
			want = append(want, row.Program.ID.String())
		}
		assert.ElementsMatch(t, want, fuelProgramIDs(page))
		assert.Equal(t, []string{mine.program}, fuelProgramIDs(page))
		assert.NotContains(t, fuelProgramIDs(page), theirs.program)

		matched := callTool(t, mine.ctx, tool, actor, map[string]any{"query": "prga"})
		assert.Equal(t, []string{mine.program}, fuelProgramIDs(matched))
		missed := callTool(t, mine.ctx, tool, actor, map[string]any{"query": "nothing like it"})
		assert.Empty(t, fuelProgramIDs(missed))
	})
}
