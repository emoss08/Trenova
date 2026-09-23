package billingtransferrepository

import (
	"strings"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
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

// capturingMatcher records every statement bun renders so a test can assert on
// the SQL itself. The bug this guards is in the SET clause, not in the bind
// values: a `nullzero` column renders inline rather than as a bind parameter.
type capturingMatcher struct {
	mu         sync.Mutex
	statements []string
}

func (m *capturingMatcher) Match(expectedSQL, actualSQL string) error {
	m.mu.Lock()
	m.statements = append(m.statements, actualSQL)
	m.mu.Unlock()

	return sqlmock.QueryMatcherRegexp.Match(expectedSQL, actualSQL)
}

func (m *capturingMatcher) find(t *testing.T, needle string) string {
	t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, statement := range m.statements {
		if strings.Contains(statement, needle) {
			return statement
		}
	}

	t.Fatalf("no statement containing %q; saw %v", needle, m.statements)

	return ""
}

func newTestRepo(t *testing.T) (*repository, sqlmock.Sqlmock, *capturingMatcher) {
	t.Helper()

	matcher := &capturingMatcher{}
	db, dbMock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		dbMock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	return &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}, dbMock, matcher
}

// expectRecordFlow sets up the three statements one flush makes: the item
// update, the aggregate over the run's items, and the counter write-back.
func expectRecordFlow(dbMock sqlmock.Sqlmock) {
	dbMock.ExpectBegin()
	dbMock.ExpectExec(`UPDATE "billing_transfer_run_items"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectQuery(`SELECT COUNT\(\*\)`).WillReturnRows(
		sqlmock.NewRows([]string{
			"processed", "transferred", "not_transferred", "skipped", "marked_ready", "retryable",
		}).AddRow(1, 1, 0, 0, 0, 0),
	)
	dbMock.ExpectQuery(`UPDATE "billing_transfer_runs"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "status", "total_count", "processed_count"}).
			AddRow("btr_1", string(billingtransfer.RunStatusRunning), 1, 1),
	)
	dbMock.ExpectCommit()
}

func recordOne(
	t *testing.T,
	repo *repository,
	outcome repositories.BillingTransferItemOutcome,
) {
	t.Helper()

	_, err := repo.RecordItemOutcomes(
		t.Context(),
		&repositories.RecordBillingTransferOutcomesRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			RunID:    pulid.MustNew("btr_"),
			Outcomes: []repositories.BillingTransferItemOutcome{outcome},
		},
	)
	require.NoError(t, err)
}

// billing_queue_status is a Postgres enum and failure_code carries a check
// constraint, so an unset value has to reach the database as NULL. Sending the
// Go zero value instead writes "", which Postgres rejects with
// `invalid input value for enum billing_queue_status` — and because the whole
// flush is one transaction, that fails every shipment in the batch, not just
// the one with the empty column.
func TestRecordItemOutcomesWritesUnsetColumnsAsNull(t *testing.T) {
	t.Parallel()

	repo, dbMock, matcher := newTestRepo(t)
	expectRecordFlow(dbMock)

	// A shipment that transferred cleanly: no failure code, no error message.
	recordOne(t, repo, repositories.BillingTransferItemOutcome{
		ShipmentID: pulid.MustNew("shp_"),
		Status:     billingtransfer.ItemStatusTransferred,
	})

	// bun renders a `nullzero` column as "= DEFAULT" on an update, and none of
	// these columns declares a DEFAULT clause, so Postgres resolves that to NULL.
	// What matters is that none of them is written as an empty string.
	update := matcher.find(t, `UPDATE "billing_transfer_run_items"`)
	assert.Contains(t, update, `"failure_code" = DEFAULT`)
	assert.Contains(t, update, `"billing_queue_status" = DEFAULT`)
	assert.Contains(t, update, `"billing_queue_item_id" = DEFAULT`)
	assert.Contains(t, update, `"error_message" = DEFAULT`)
	assert.NotContains(t, update, `= ''`, "an empty string is not how an unset column is written")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// The mirror of the above: a shipment that did not transfer carries a failure
// code but no queue item, so the enum is the column that must be NULL while the
// reason must survive.
func TestRecordItemOutcomesKeepsTheFailureReasonWithoutAQueueStatus(t *testing.T) {
	t.Parallel()

	repo, dbMock, matcher := newTestRepo(t)
	expectRecordFlow(dbMock)

	recordOne(t, repo, repositories.BillingTransferItemOutcome{
		ShipmentID:   pulid.MustNew("shp_"),
		ProNumber:    "PRO-1",
		Status:       billingtransfer.ItemStatusNotTransferred,
		FailureCode:  billingtransfer.FailureRequirementsUnmet,
		ErrorMessage: "Billing requirements must be resolved first",
	})

	update := matcher.find(t, `UPDATE "billing_transfer_run_items"`)
	assert.Contains(t, update, `"billing_queue_status" = DEFAULT`)
	assert.Contains(t, update, `"failure_code" = 'RequirementsUnmet'`)
	assert.NotContains(t, update, `"billing_queue_status" = ''`)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// NULLing everything would be exactly as wrong as NULLing nothing: a transferred
// shipment's queue item is the link the report shows.
func TestRecordItemOutcomesWritesTheQueueItemWhenThereIsOne(t *testing.T) {
	t.Parallel()

	repo, dbMock, matcher := newTestRepo(t)
	expectRecordFlow(dbMock)

	recordOne(t, repo, repositories.BillingTransferItemOutcome{
		ShipmentID:         pulid.MustNew("shp_"),
		ProNumber:          "PRO-1",
		Status:             billingtransfer.ItemStatusTransferred,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		BillingQueueNumber: "INV-1",
		BillingQueueStatus: billingqueue.StatusReadyForReview,
	})

	update := matcher.find(t, `UPDATE "billing_transfer_run_items"`)
	assert.Contains(t, update, `"billing_queue_status" = 'ReadyForReview'`)
	assert.Contains(t, update, `"billing_queue_number" = 'INV-1'`)
	assert.NotContains(t, update, `"billing_queue_status" = DEFAULT`)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// The update must touch only the columns an outcome owns. Writing through the
// model without restricting the column list would also rewrite the row's
// identity and its place in the run.
func TestRecordItemOutcomesLeavesTheRowsIdentityAlone(t *testing.T) {
	t.Parallel()

	repo, dbMock, matcher := newTestRepo(t)
	expectRecordFlow(dbMock)

	recordOne(t, repo, repositories.BillingTransferItemOutcome{
		ShipmentID: pulid.MustNew("shp_"),
		Status:     billingtransfer.ItemStatusTransferred,
	})

	update := matcher.find(t, `UPDATE "billing_transfer_run_items"`)
	setClause := update[strings.Index(update, "SET"):strings.Index(update, "WHERE")]
	assert.NotContains(t, setClause, `"id" =`)
	assert.NotContains(t, setClause, `"sequence" =`)
	assert.NotContains(t, setClause, `"run_id" =`)
	assert.NotContains(t, setClause, `"created_at" =`)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}
