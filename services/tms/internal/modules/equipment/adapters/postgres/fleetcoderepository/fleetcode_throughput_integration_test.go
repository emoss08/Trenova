//go:build integration

package fleetcoderepository

import (
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const throughputEnv = "TRENOVA_DB_THROUGHPUT"

func TestFleetCodeRepository_Throughput(t *testing.T) {
	if os.Getenv(throughputEnv) == "" {
		t.Skip("set " + throughputEnv + "=1 to measure the repository read and write paths")
	}

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	tenantInfo := &pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}

	entity := &fleetcode.FleetCode{
		ID:             pulid.MustNew("fc_"),
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		ManagerID:      tenantInfo.UserID,
		Status:         domaintypes.StatusActive,
		Code:           "BENCH",
		Description:    "Throughput",
	}
	require.NoError(t, fleetcodedb.FleetCodes.Create(data.Ctx(ctx), admin, entity))

	repo := New(Params{DB: dbtest.NewRLSConnection(t, admin, dsn), Logger: zap.NewNop()})
	scoped := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		UserID:         tenantInfo.UserID,
	})
	get := fleetcodeports.GetFleetCodeByIDRequest{ID: entity.ID, TenantInfo: tenantInfo}

	read := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := repo.GetByID(scoped, get); err != nil {
				b.Fatal(err)
			}
		}
	})

	parallelRead := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(4)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := repo.GetByID(scoped, get); err != nil {
					b.Error(err)
					return
				}
			}
		})
	})

	current, err := repo.GetByID(scoped, get)
	require.NoError(t, err)
	write := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			current.Description = pulid.MustNew("d_").String()
			if current, err = repo.Update(scoped, current); err != nil {
				b.Fatal(err)
			}
		}
	})

	t.Logf("GetByID:          %s %s", read, read.MemString())
	t.Logf("GetByID parallel: %s %s", parallelRead, parallelRead.MemString())
	t.Logf("Update:           %s %s", write, write.MemString())
}
