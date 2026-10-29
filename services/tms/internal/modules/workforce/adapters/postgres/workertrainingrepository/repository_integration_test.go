//go:build integration

package workertrainingrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining/workertrainingdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workertrainingports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedTrainingCourse(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *workertraining.TrainingCourse {
	t.Helper()

	entity := &workertraining.TrainingCourse{Code: code, Name: code + " course"}
	require.NoError(t, workertrainingdb.TrainingCourses.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetCoursesByIDsReturnsOnlyTheTenantsCourses(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "TC")
	first := seedTrainingCourse(t, ctx, db, tenantA, "DEFENSIVE")
	second := seedTrainingCourse(t, ctx, db, tenantA, "HOS")
	foreign := seedTrainingCourse(t, ctx, db, tenantB, "FORKLIFT")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetCoursesByIDs(
		tenantA.Ctx(ctx),
		&workertrainingports.GetTrainingCoursesByIDsRequest{
			TenantInfo: tenantA.Info(),
			CourseIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("trnc_")},
		},
	)
	require.NoError(t, err)

	keys := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		keys[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "DEFENSIVE", second.ID: "HOS"}, keys)

	none, err := repo.GetCoursesByIDs(
		tenantA.Ctx(ctx),
		&workertrainingports.GetTrainingCoursesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
