//go:build integration

package airetrievalrepository

import (
	"context"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const expectVectorEnv = "TRENOVA_TEST_EXPECT_VECTOR"

type harness struct {
	ctx    context.Context
	db     *bun.DB
	conn   *postgres.Connection
	repo   repositories.AIRetrievalRepository
	data   *seedtest.TestData
	tenant pagination.TenantInfo
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)

	return &harness{
		ctx:    ctx,
		db:     db,
		conn:   conn,
		repo:   newRepository(conn),
		data:   data,
		tenant: pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID},
	}
}

func newRepository(conn *postgres.Connection) repositories.AIRetrievalRepository {
	return New(Params{
		DB:     conn,
		Probe:  postgres.NewTestCapabilityProbe(conn),
		Logger: zap.NewNop(),
	})
}

func (h *harness) requireVector(t *testing.T) {
	t.Helper()

	availability, err := h.repo.VectorAvailability(h.ctx)
	require.NoError(t, err)
	if availability.Available {
		return
	}

	if os.Getenv(expectVectorEnv) == "true" {
		t.Fatalf("pgvector is expected on this server but reports %s", availability.Reason)
	}

	t.Skipf(
		"pgvector is unavailable (%s); run with %s set to the image built from deploy/Dockerfile.postgres",
		availability.Reason,
		seedtest.PostgresImageEnv,
	)
}

func axisVector(dimensions int, weights map[int]float32) []float32 {
	vector := make([]float32, dimensions)
	for axis, weight := range weights {
		vector[axis] = weight
	}

	return vector
}

func (h *harness) replace(
	t *testing.T,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	sourceID pulid.ID,
	modelKey string,
	dimensions int,
	chunks ...repositories.EmbeddingChunk,
) repositories.ReplaceEmbeddingChunksResult {
	t.Helper()

	result, err := h.repo.ReplaceChunks(h.ctx, &repositories.ReplaceEmbeddingChunksRequest{
		Source: repositories.AIRetrievalSourceRef{
			TenantInfo: tenant,
			SourceType: sourceType,
			SourceID:   sourceID,
		},
		ModelKey:   modelKey,
		Dimensions: dimensions,
		Chunks:     chunks,
	})
	require.NoError(t, err)

	return result
}

func (h *harness) insertMemory(t *testing.T, content string) *agent.Memory {
	t.Helper()

	memory := &agent.Memory{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Kind:           agent.MemoryKindFact,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		Content:        content,
	}
	_, err := h.db.NewInsert().Model(memory).Exec(h.ctx)
	require.NoError(t, err)

	return memory
}

func (h *harness) insertDocument(t *testing.T) *document.Document {
	t.Helper()

	entity := &document.Document{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		FileName:       "rate-confirmation.pdf",
		OriginalName:   "rate-confirmation.pdf",
		FileSize:       128,
		FileType:       "application/pdf",
		StoragePath:    "test/rate-confirmation.pdf",
		Status:         document.StatusActive,
		ResourceID:     pulid.MustNew("shp_").String(),
		ResourceType:   "Shipment",
		UploadedByID:   h.data.User.ID,
	}
	_, err := h.db.NewInsert().Model(entity).Exec(h.ctx)
	require.NoError(t, err)

	return entity
}

func (h *harness) countEmbeddings(t *testing.T, sourceID pulid.ID, modelKey string) int {
	t.Helper()

	cols := buncolgen.EmbeddingColumns
	q := h.db.NewSelect().
		Model((*airetrieval.Embedding)(nil)).
		Where(cols.SourceID.Eq(), sourceID)
	if modelKey != "" {
		q = q.Where(cols.ModelKey.Eq(), modelKey)
	}

	count, err := q.Count(h.ctx)
	require.NoError(t, err)

	return count
}

func (h *harness) countIndexEntries(t *testing.T, sourceID pulid.ID, modelKey string) int {
	t.Helper()

	cols := buncolgen.IndexEntryColumns
	q := h.db.NewSelect().
		Model((*airetrieval.IndexEntry)(nil)).
		Where(cols.SourceID.Eq(), sourceID)
	if modelKey != "" {
		q = q.Where(cols.ModelKey.Eq(), modelKey)
	}

	count, err := q.Count(h.ctx)
	require.NoError(t, err)

	return count
}
