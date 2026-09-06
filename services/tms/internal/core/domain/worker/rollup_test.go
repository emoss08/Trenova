package worker_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func unixPtr(v int64) *int64 { return &v }

func TestRollupFrom_UsesEverySummary(t *testing.T) {
	rollup := worker.RollupFrom(
		&worker.WorkerCredentialSummary{ComplianceStatus: worker.ComplianceStatusNonCompliant},
		&worker.WorkerTrainingSummary{
			Items: []*worker.TrainingSummaryItem{
				{Required: true, Health: worker.TrainingHealthCurrent},
				{Required: true, Health: worker.TrainingHealthExpired},
			},
		},
		&worker.SafetyScorecard{Rating: worker.SafetyRatingWatch, Score: 71},
	)

	assert.Equal(t, worker.ComplianceStatusNonCompliant, rollup.ComplianceStatus)
	assert.Equal(t, worker.TrainingHealthExpired, rollup.TrainingHealth)
	assert.Equal(t, worker.SafetyRatingWatch, rollup.SafetyRating)
	assert.Equal(t, int16(71), rollup.SafetyScore)
}

// A tenant with no courses configured has no training summary to speak of.
// That must read as "nothing to report", never as a problem.
func TestRollupFrom_NilSummariesMeanNothingToReport(t *testing.T) {
	rollup := worker.RollupFrom(nil, nil, nil)

	assert.Equal(t, worker.ComplianceStatusPending, rollup.ComplianceStatus)
	assert.Equal(t, worker.TrainingHealthCurrent, rollup.TrainingHealth)
	assert.Equal(t, worker.SafetyRatingExcellent, rollup.SafetyRating)
	assert.Equal(t, int16(100), rollup.SafetyScore)
	assert.Nil(t, rollup.NextCredentialExpiry)
	assert.Nil(t, rollup.NextTrainingDue)
}

// Nine current courses and one expired certification is not "current". The
// roster has one column to say how a worker stands, and it has to say the
// worst thing.
func TestWorstHealth_ReportsTheWorstRequiredCourse(t *testing.T) {
	summary := &worker.WorkerTrainingSummary{
		Items: []*worker.TrainingSummaryItem{
			{Required: true, Health: worker.TrainingHealthCurrent},
			{Required: true, Health: worker.TrainingHealthDueSoon},
			{Required: true, Health: worker.TrainingHealthMissing},
			{Required: true, Health: worker.TrainingHealthCurrent},
		},
	}

	assert.Equal(t, worker.TrainingHealthMissing, summary.WorstHealth())
}

// An optional course in a bad state must not drag the worker's roll-up down;
// nobody should be chased over training the role does not require.
func TestWorstHealth_IgnoresOptionalCourses(t *testing.T) {
	summary := &worker.WorkerTrainingSummary{
		Items: []*worker.TrainingSummaryItem{
			{Required: true, Health: worker.TrainingHealthCurrent},
			{Required: false, Health: worker.TrainingHealthExpired},
		},
	}

	assert.Equal(t, worker.TrainingHealthCurrent, summary.WorstHealth())
}

func TestWorstHealth_EmptyAndNilAreCurrent(t *testing.T) {
	assert.Equal(t, worker.TrainingHealthCurrent, (*worker.WorkerTrainingSummary)(nil).WorstHealth())
	assert.Equal(
		t,
		worker.TrainingHealthCurrent,
		(&worker.WorkerTrainingSummary{}).WorstHealth(),
	)
}

func TestNextExpiry_EarliestRequiredCredential(t *testing.T) {
	summary := &worker.WorkerCredentialSummary{
		Items: []*worker.CredentialSummaryItem{
			{Required: true, Credential: &worker.WorkerCredential{ExpiresAt: unixPtr(3_000)}},
			{Required: true, Credential: &worker.WorkerCredential{ExpiresAt: unixPtr(1_500)}},
			{Required: true, Credential: &worker.WorkerCredential{ExpiresAt: unixPtr(9_000)}},
		},
	}

	require.NotNil(t, summary.NextExpiry())
	assert.Equal(t, int64(1_500), *summary.NextExpiry())
}

// An optional credential expiring tomorrow must not become the date the roster
// counts down to for a required one expiring next year.
func TestNextExpiry_IgnoresOptionalAndMissingDates(t *testing.T) {
	summary := &worker.WorkerCredentialSummary{
		Items: []*worker.CredentialSummaryItem{
			{Required: false, Credential: &worker.WorkerCredential{ExpiresAt: unixPtr(10)}},
			{Required: true, Credential: &worker.WorkerCredential{ExpiresAt: nil}},
			{Required: true, Credential: nil},
			{Required: true, Credential: &worker.WorkerCredential{ExpiresAt: unixPtr(5_000)}},
		},
	}

	require.NotNil(t, summary.NextExpiry())
	assert.Equal(t, int64(5_000), *summary.NextExpiry())
}

func TestNextExpiry_NothingExpiring(t *testing.T) {
	summary := &worker.WorkerCredentialSummary{
		Items: []*worker.CredentialSummaryItem{
			{Required: true, Credential: &worker.WorkerCredential{ExpiresAt: nil}},
		},
	}

	assert.Nil(t, summary.NextExpiry())
}

// Whichever comes first: a due date on something still open, or an expiry on
// something already current.
func TestNextDue_TakesTheSoonerOfDueAndExpiry(t *testing.T) {
	summary := &worker.WorkerTrainingSummary{
		Items: []*worker.TrainingSummaryItem{
			{
				Required: true,
				Record:   &worker.WorkerTrainingRecord{DueAt: unixPtr(8_000)},
			},
			{
				Required: true,
				Record:   &worker.WorkerTrainingRecord{ExpiresAt: unixPtr(4_000)},
			},
		},
	}

	require.NotNil(t, summary.NextDue())
	assert.Equal(t, int64(4_000), *summary.NextDue())
}

// A refresh that changes nothing must not write, so updated_at keeps meaning
// "something about this worker actually moved".
func TestProfileRollup_Equal(t *testing.T) {
	base := worker.ProfileRollup{
		ComplianceStatus:     worker.ComplianceStatusCompliant,
		TrainingHealth:       worker.TrainingHealthCurrent,
		SafetyRating:         worker.SafetyRatingGood,
		SafetyScore:          88,
		NextCredentialExpiry: unixPtr(1_000),
	}

	same := base
	assert.True(t, base.Equal(same))

	scoreMoved := base
	scoreMoved.SafetyScore = 87
	assert.False(t, base.Equal(scoreMoved))

	expiryCleared := base
	expiryCleared.NextCredentialExpiry = nil
	assert.False(t, base.Equal(expiryCleared))

	expiryMoved := base
	expiryMoved.NextCredentialExpiry = unixPtr(2_000)
	assert.False(t, base.Equal(expiryMoved))
}

// The roster is worthless if it can display the roll-ups but not filter on
// them. Declaring the profile queryable is what makes `profile.<field>` reach
// the joined table, so this pins the SQL rather than merely asserting the
// query is non-nil.
func TestWorkerSearchConfig_ProfileIsFilterableAndSortable(t *testing.T) {
	querybuilder.ClearCaches()
	entity := &worker.Worker{}
	config := querybuilder.GetFieldConfiguration(entity)

	for _, field := range []string{
		"profile.complianceStatus",
		"profile.trainingHealth",
		"profile.safetyRating",
		"profile.nextCredentialExpiry",
	} {
		assert.True(t, config.FilterableFields[field], "filterable: %s", field)
		assert.True(t, config.SortableFields[field], "sortable: %s", field)
	}
}

func TestWorkerSearchConfig_ProfileFilterJoinsAndQualifies(t *testing.T) {
	querybuilder.ClearCaches()
	entity := &worker.Worker{}
	db := bun.NewDB(nil, pgdialect.New())

	query := db.NewSelect().
		Model((*worker.Worker)(nil)).
		ModelTableExpr("workers AS wrk")
	qb := querybuilder.NewWithPostgresSearch(
		query,
		"wrk",
		querybuilder.GetFieldConfiguration(entity),
		entity,
	)
	qb.WithTraversalSupport(true)
	qb.ApplyFilters([]domaintypes.FieldFilter{
		{
			Field:    "profile.trainingHealth",
			Operator: dbtype.OpEqual,
			Value:    string(worker.TrainingHealthOverdue),
		},
	})

	sql := qb.GetQuery().String()
	assert.Contains(t, sql, "worker_profiles")
	assert.Contains(t, sql, "wprof")
	assert.Contains(t, sql, "training_health")
	// The join has to hang off the worker row, not float free.
	assert.True(
		t,
		strings.Contains(sql, "wprof.worker_id = wrk.id"),
		"expected the profile join to be keyed on the worker, got: %s",
		sql,
	)
}
