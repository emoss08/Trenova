//go:build integration

package fuelsurchargerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge/fuelsurchargedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/ratingdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fuelFixture struct {
	index   *fuelsurcharge.FuelIndex
	program *fuelsurcharge.FuelSurchargeProgram
}

func insertFuelFixture(t *testing.T, rdb *ratingdb.Database, tenant ratingdb.Tenant, code string) fuelFixture {
	t.Helper()

	index := &fuelsurcharge.FuelIndex{
		OrganizationID: tenant.Info.OrgID,
		BusinessUnitID: tenant.Info.BuID,
		Name:           code + " Index",
		Code:           code + "IDX",
		Source:         fuelsurcharge.IndexSourceCustom,
		FuelType:       fuelsurcharge.FuelTypeDiesel,
		Currency:       "USD",
		IsActive:       true,
	}
	require.NoError(t, fuelsurchargedb.FuelIndexes.Create(tenant.Ctx, rdb.Admin, index))

	charge := &accessorialcharge.AccessorialCharge{
		OrganizationID: tenant.Info.OrgID,
		BusinessUnitID: tenant.Info.BuID,
		Code:           code + "FSC",
		Description:    "Fuel surcharge",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromInt(0),
	}
	require.NoError(t, accessorialchargedb.AccessorialCharges.Create(tenant.Ctx, rdb.Admin, charge))

	program := &fuelsurcharge.FuelSurchargeProgram{
		OrganizationID:       tenant.Info.OrgID,
		BusinessUnitID:       tenant.Info.BuID,
		Name:                 code + " Program",
		Code:                 code + "FSP",
		FuelIndexID:          index.ID,
		AccessorialChargeID:  charge.ID,
		Method:               fuelsurcharge.ProgramMethodPerMileStep,
		PegPrice:             decimal.NewNullDecimal(decimal.RequireFromString("1.20")),
		Increment:            decimal.NewNullDecimal(decimal.RequireFromString("0.05")),
		IncrementRate:        decimal.NewNullDecimal(decimal.RequireFromString("0.01")),
		RatePrecision:        4,
		DateBasis:            fuelsurcharge.DateBasisPickupDate,
		PriceEffectiveDay:    3,
		MissingPriceFallback: fuelsurcharge.FallbackUseLatestAvailable,
		Status:               fuelsurcharge.ProgramStatusActive,
	}
	require.NoError(t, fuelsurchargedb.FuelSurchargePrograms.Create(tenant.Ctx, rdb.Admin, program))

	return fuelFixture{index: index, program: program}
}

func TestGetByIDsReadsOnlyTheTenantsIndexesAndPrograms(t *testing.T) {
	t.Parallel()

	rdb := ratingdb.NewDatabase(t)
	first := insertFuelFixture(t, rdb, rdb.A, "ONE")
	second := insertFuelFixture(t, rdb, rdb.A, "TWO")
	foreign := insertFuelFixture(t, rdb, rdb.B, "OTH")

	indexes := NewIndexRepository(IndexParams{DB: rdb.Conn, Logger: zap.NewNop()})
	programs := NewProgramRepository(ProgramParams{DB: rdb.Conn, Logger: zap.NewNop()})

	gotIndexes, err := indexes.GetByIDs(rdb.A.Ctx, &fuelsurchargeports.GetFuelIndexesByIDsRequest{
		TenantInfo: rdb.A.Info,
		FuelIndexIDs: []pulid.ID{
			second.index.ID,
			foreign.index.ID,
			first.index.ID,
			pulid.MustNew("fidx_"),
		},
	})
	require.NoError(t, err)
	indexCodes := make(map[pulid.ID]string, len(gotIndexes))
	for _, index := range gotIndexes {
		indexCodes[index.ID] = index.Code
	}
	assert.Equal(
		t,
		map[pulid.ID]string{first.index.ID: "ONEIDX", second.index.ID: "TWOIDX"},
		indexCodes,
	)

	gotPrograms, err := programs.GetByIDs(
		rdb.A.Ctx,
		&fuelsurchargeports.GetFuelSurchargeProgramsByIDsRequest{
			TenantInfo: rdb.A.Info,
			ProgramIDs: []pulid.ID{
				second.program.ID,
				foreign.program.ID,
				first.program.ID,
				pulid.MustNew("fsp_"),
			},
		},
	)
	require.NoError(t, err)
	programCodes := make(map[pulid.ID]string, len(gotPrograms))
	for _, program := range gotPrograms {
		programCodes[program.ID] = program.Code
		assert.Empty(t, program.TableRows, "a batch read loads no table rows")
	}
	assert.Equal(
		t,
		map[pulid.ID]string{first.program.ID: "ONEFSP", second.program.ID: "TWOFSP"},
		programCodes,
	)

	noIndexes, err := indexes.GetByIDs(
		rdb.A.Ctx,
		&fuelsurchargeports.GetFuelIndexesByIDsRequest{TenantInfo: rdb.A.Info},
	)
	require.NoError(t, err)
	assert.Empty(t, noIndexes)

	noPrograms, err := programs.GetByIDs(
		rdb.A.Ctx,
		&fuelsurchargeports.GetFuelSurchargeProgramsByIDsRequest{TenantInfo: rdb.A.Info},
	)
	require.NoError(t, err)
	assert.Empty(t, noPrograms)
}
