package agentrunrepository

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

type queryLog struct {
	queries []string
}

func (l *queryLog) Match(expected, actual string) error {
	if !strings.Contains(actual, expected) {
		return fmt.Errorf("query %q does not contain %q", actual, expected)
	}
	if !slices.Contains(l.queries, actual) {
		l.queries = append(l.queries, actual)
	}

	return nil
}

func newRunRepository(t *testing.T) (*repository, sqlmock.Sqlmock, *queryLog) {
	t.Helper()

	log := &queryLog{}
	db, sqlMock, err := sqlmock.New(sqlmock.QueryMatcherOption(log))
	require.NoError(t, err)
	sqlMock.MatchExpectationsInOrder(false)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		require.NoError(t, sqlMock.ExpectationsWereMet())
		_ = bunDB.Close()
	})

	return &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}, sqlMock, log
}

func runTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestRunReads_LeaveTheTranscriptBehind(t *testing.T) {
	t.Parallel()

	repo, sqlMock, log := newRunRepository(t)
	tenant := runTenant()
	runID := pulid.MustNew("arun_")

	sqlMock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(runID.String()))
	_, err := repo.GetByID(t.Context(), repositories.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &tenant,
	})
	require.NoError(t, err)

	sqlMock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(runID.String()))
	_, err = repo.ListByIDs(t.Context(), repositories.ListAgentRunsByIDsRequest{
		IDs:        []pulid.ID{runID},
		TenantInfo: tenant,
	})
	require.NoError(t, err)

	sqlMock.ExpectQuery(`"ar"."summary"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(runID.String()))
	sqlMock.ExpectQuery("count(").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	_, err = repo.List(t.Context(), &repositories.ListAgentRunRequest{
		Filter: &pagination.QueryOptions{TenantInfo: tenant},
	})
	require.NoError(t, err)

	require.Len(t, log.queries, 4)
	for _, query := range log.queries {
		assert.NotContains(t, query, "transcript", query)
	}
	assert.Contains(t, log.queries[0], `"ar"."summary"`, "the run itself is read")
	assert.Contains(t, log.queries[1], `"ar"."summary"`, "the run itself is read")
}

func TestListTranscriptsByIDs_ReadsOnlyTheTranscriptOfTheTenantsRuns(t *testing.T) {
	t.Parallel()

	repo, sqlMock, log := newRunRepository(t)
	tenant := runTenant()
	runID := pulid.MustNew("arun_")

	sqlMock.ExpectQuery("").WillReturnRows(
		sqlmock.NewRows([]string{"id", "business_unit_id", "organization_id", "transcript"}).
			AddRow(
				runID.String(),
				tenant.BuID.String(),
				tenant.OrgID.String(),
				`{"messages":[{"role":"Assistant","content":"Done.","createdAt":7}],"omittedMessages":3,"omittedAt":1}`,
			),
	)

	runs, err := repo.ListTranscriptsByIDs(t.Context(), repositories.ListAgentRunsByIDsRequest{
		IDs:        []pulid.ID{runID},
		TenantInfo: tenant,
	})
	require.NoError(t, err)

	require.Len(t, log.queries, 1)
	query := log.queries[0]
	assert.Contains(t, query, `"ar"."transcript"`)
	assert.NotContains(t, query, `"ar"."summary"`, "nothing but the transcript and its keys")
	assert.Contains(t, query, `"ar"."organization_id"`)
	assert.Contains(t, query, `"ar"."business_unit_id"`)

	require.Len(t, runs, 1)
	assert.Equal(t, runID, runs[0].ID)
	require.NotNil(t, runs[0].Transcript)
	assert.Equal(t, &agent.RunTranscript{
		Messages:        []agent.TranscriptMessage{{Role: "Assistant", Content: "Done.", CreatedAt: 7}},
		OmittedMessages: 3,
		OmittedAt:       1,
	}, runs[0].Transcript)
}

func TestListTranscriptsByIDs_AsksNothingForNoRuns(t *testing.T) {
	t.Parallel()

	repo, _, log := newRunRepository(t)

	runs, err := repo.ListTranscriptsByIDs(t.Context(), repositories.ListAgentRunsByIDsRequest{
		TenantInfo: runTenant(),
	})
	require.NoError(t, err)
	assert.Empty(t, runs)
	assert.Empty(t, log.queries)
}

func TestListConnection_WithoutAProjectionLeavesTheTranscriptBehind(t *testing.T) {
	t.Parallel()

	repo, sqlMock, log := newRunRepository(t)
	tenant := runTenant()

	sqlMock.ExpectQuery(`"ar"."summary"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(pulid.MustNew("arun_").String()))
	_, err := repo.ListConnection(t.Context(), &repositories.ListAgentRunConnectionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: 10},
		},
	})
	require.NoError(t, err)

	require.Len(t, log.queries, 1)
	assert.NotContains(t, log.queries[0], "transcript")
}
