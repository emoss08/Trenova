package agentruneventrepository

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

type queryLog struct {
	mu      sync.Mutex
	queries []string
}

func (l *queryLog) Match(expected, actual string) error {
	if !strings.Contains(actual, expected) {
		return fmt.Errorf("query %q does not contain %q", actual, expected)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !slices.Contains(l.queries, actual) {
		l.queries = append(l.queries, actual)
	}

	return nil
}

func readEventsEveryWay(t *testing.T, reader agentports.RunReader) []string {
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
	repo := &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	owner := string(agentports.RunStepOwnerAgentRun)

	sqlMock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	_, err = repo.List(t.Context(), agentports.ListAgentRunEventsRequest{
		TenantInfo: tenant,
		OwnerKind:  owner,
		OwnerID:    pulid.MustNew("ar_"),
		Reader:     reader,
	})
	require.NoError(t, err)

	sqlMock.ExpectQuery("count(").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	sqlMock.ExpectQuery(`"are"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	_, err = repo.ListConnection(t.Context(), &agentports.ListAgentRunEventConnectionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: 10},
		},
		Cursor:    pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		OwnerKind: owner,
		Reader:    reader,
	})
	require.NoError(t, err)

	require.Len(t, log.queries, 3, "a by-run read, a count and a page")

	return log.queries
}

func TestRunEventReads_KeepAConversationRunsEventsToItsOwnerForAPerson(t *testing.T) {
	t.Parallel()

	person := pulid.MustNew("usr_")
	for _, query := range readEventsEveryWay(t, agentports.PersonRunRead(person)) {
		assert.Contains(t, query, `(are.owner_kind <> 'AgentRun')`, query)
		assert.Contains(t, query, `EXISTS (SELECT 1 FROM "agent_runs" AS "ar"`, query)
		assert.Contains(t, query, `(ar.id = are.owner_id)`, query)
		assert.Contains(t, query, `(ar.organization_id = are.organization_id)`, query)
		assert.Contains(t, query, `(ar.business_unit_id = are.business_unit_id)`, query)
		assert.Contains(t, query, `(ar.subject_type <> 'AssistantThread')`, query)
		assert.Contains(t, query, `(athr.user_id = '`+person.String()+`')`, query)
	}
}

func TestRunEventReads_ShowNoConversationRunsEventsToAReaderWhoIsNoPerson(t *testing.T) {
	t.Parallel()

	for _, query := range readEventsEveryWay(t, agentports.PersonRunRead(pulid.Nil)) {
		assert.Contains(t, query, `(ar.subject_type <> 'AssistantThread')`, query)
		assert.NotContains(t, query, "assistant_threads", query)
	}
}

func TestRunEventReads_LeaveAnUnattendedReadAsItWas(t *testing.T) {
	t.Parallel()

	for _, query := range readEventsEveryWay(t, agentports.UnattendedRunRead()) {
		assert.NotContains(t, query, "agent_runs", query)
		assert.NotContains(t, query, "assistant_threads", query)
	}
}
