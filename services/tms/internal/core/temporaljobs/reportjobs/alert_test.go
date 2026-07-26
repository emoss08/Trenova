package reportjobs

import (
	"context"
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// stubScheduleRepo records the alert state the activity persists. Only Update
// is exercised; the rest satisfies the port.
type stubScheduleRepo struct {
	updates []bool
	err     error
}

func (s *stubScheduleRepo) Update(
	_ context.Context,
	entity *report.ReportSchedule,
) (*report.ReportSchedule, error) {
	s.updates = append(s.updates, entity.AlertFiring)
	return entity, s.err
}

func (s *stubScheduleRepo) Create(
	_ context.Context, e *report.ReportSchedule,
) (*report.ReportSchedule, error) {
	return e, nil
}

func (s *stubScheduleRepo) GetByID(
	_ context.Context, _ *repositories.GetReportScheduleRequest,
) (*report.ReportSchedule, error) {
	return nil, nil //nolint:nilnil // unused by these tests
}

func (s *stubScheduleRepo) List(
	_ context.Context, _ *repositories.ListReportSchedulesRequest,
) ([]*report.ReportSchedule, error) {
	return nil, nil
}

func (s *stubScheduleRepo) ListAllEnabled(_ context.Context) ([]*report.ReportSchedule, error) {
	return nil, nil
}

func (s *stubScheduleRepo) ListDue(
	_ context.Context, _ int64, _ int,
) ([]*report.ReportSchedule, error) {
	return nil, nil
}

func (s *stubScheduleRepo) Delete(
	_ context.Context, _ *repositories.GetReportScheduleRequest,
) error {
	return nil
}

func alertActivities(repo *stubScheduleRepo) *Activities {
	return &Activities{scheduleRepo: repo, l: zap.NewNop()}
}

func alertSchedule(alert *report.ScheduleAlert, firing bool) *report.ReportSchedule {
	schedule := deliveryTestSchedule(&report.ScheduleDelivery{
		EmailRecipients: []string{"ops@example.com"},
	})
	schedule.Alert = alert
	schedule.AlertFiring = firing
	return schedule
}

func alertRun(rowCount int64) *report.ReportRun {
	run := deliveryTestRun()
	run.RowCount = rowCount
	return run
}

// Every schedule that predates alerts must keep delivering unconditionally.
func TestNoAlertAlwaysDelivers(t *testing.T) {
	repo := &stubScheduleRepo{}

	deliver, reason := alertActivities(repo).evaluateAlert(
		t.Context(), alertRun(0), alertSchedule(nil, false), nil,
	)

	assert.True(t, deliver)
	assert.Empty(t, reason)
	assert.Empty(t, repo.updates, "a schedule without an alert has no state to write")
}

// The exception-report pattern: a report that should come back empty only
// matters when it does not.
func TestAlertHoldsBackAnEmptyResult(t *testing.T) {
	repo := &stubScheduleRepo{}
	alert := &report.ScheduleAlert{Operator: dbtype.OpGreaterThan, Threshold: 0}

	deliver, reason := alertActivities(repo).evaluateAlert(
		t.Context(), alertRun(0), alertSchedule(alert, false), nil,
	)
	assert.False(t, deliver)
	assert.Equal(t, alertSkipConditionUnmet, reason)

	deliver, reason = alertActivities(repo).evaluateAlert(
		t.Context(), alertRun(3), alertSchedule(alert, false), nil,
	)
	assert.True(t, deliver)
	assert.Empty(t, reason)
}

// A daily exception report that stays broken for a week should mail once.
func TestSuppressionMailsOnlyTheTransition(t *testing.T) {
	repo := &stubScheduleRepo{}
	alert := &report.ScheduleAlert{
		Operator:            dbtype.OpGreaterThan,
		Threshold:           0,
		SuppressWhileFiring: true,
	}

	// Day one: the condition trips, so it is delivered and recorded as firing.
	schedule := alertSchedule(alert, false)
	deliver, _ := alertActivities(repo).evaluateAlert(t.Context(), alertRun(5), schedule, nil)
	assert.True(t, deliver)
	assert.True(t, schedule.AlertFiring)
	assert.Equal(t, []bool{true}, repo.updates)

	// Day two: still broken, already reported — held back, and no rewrite of a
	// state that did not change.
	repo.updates = nil
	deliver, reason := alertActivities(repo).evaluateAlert(t.Context(), alertRun(5), schedule, nil)
	assert.False(t, deliver)
	assert.Equal(t, alertSkipAlreadyFiring, reason)
	assert.Empty(t, repo.updates)
}

// Once the condition clears, the alert has to re-arm — otherwise it fires once
// in the schedule's lifetime and then goes quiet forever.
func TestAlertReArmsAfterClearing(t *testing.T) {
	repo := &stubScheduleRepo{}
	alert := &report.ScheduleAlert{
		Operator:            dbtype.OpGreaterThan,
		Threshold:           0,
		SuppressWhileFiring: true,
	}
	schedule := alertSchedule(alert, true)

	// Cleared: not delivered, but the flag comes down.
	deliver, _ := alertActivities(repo).evaluateAlert(t.Context(), alertRun(0), schedule, nil)
	assert.False(t, deliver)
	assert.False(t, schedule.AlertFiring)
	assert.Equal(t, []bool{false}, repo.updates)

	// Trips again: delivered.
	deliver, _ = alertActivities(repo).evaluateAlert(t.Context(), alertRun(2), schedule, nil)
	assert.True(t, deliver)
	assert.True(t, schedule.AlertFiring)
}

// Without suppression every matching run is delivered, which is what someone
// watching a threshold day to day expects.
func TestUnsuppressedAlertDeliversEveryMatch(t *testing.T) {
	repo := &stubScheduleRepo{}
	alert := &report.ScheduleAlert{Operator: dbtype.OpGreaterThanOrEqual, Threshold: 10}
	schedule := alertSchedule(alert, true)

	for range 3 {
		deliver, _ := alertActivities(repo).evaluateAlert(t.Context(), alertRun(25), schedule, nil)
		assert.True(t, deliver)
	}
}

// A write failure must not silently disable suppression or drop the alert.
func TestAlertStillDeliversWhenStateCannotBePersisted(t *testing.T) {
	repo := &stubScheduleRepo{err: assert.AnError}
	alert := &report.ScheduleAlert{Operator: dbtype.OpGreaterThan, Threshold: 0}

	deliver, reason := alertActivities(repo).evaluateAlert(
		t.Context(), alertRun(4), alertSchedule(alert, false), nil,
	)

	assert.True(t, deliver)
	assert.Empty(t, reason)
}

func TestAlertMatchesEveryOperator(t *testing.T) {
	cases := []struct {
		op    dbtype.Operator
		rows  int64
		match bool
	}{
		{dbtype.OpGreaterThan, 11, true},
		{dbtype.OpGreaterThan, 10, false},
		{dbtype.OpGreaterThanOrEqual, 10, true},
		{dbtype.OpLessThan, 9, true},
		{dbtype.OpLessThanOrEqual, 10, true},
		{dbtype.OpEqual, 10, true},
		{dbtype.OpNotEqual, 11, true},
		{dbtype.OpNotEqual, 10, false},
	}

	for _, tc := range cases {
		alert := &report.ScheduleAlert{Operator: tc.op, Threshold: 10}
		assert.Equal(t, tc.match, alert.Matches(tc.rows), string(tc.op))
	}

	// An operator that never passed validation must not fall open.
	unknown := &report.ScheduleAlert{Operator: dbtype.OpContains, Threshold: 0}
	assert.False(t, unknown.Matches(5))

	// A nil alert means "no condition", so everything matches.
	assert.True(t, (*report.ScheduleAlert)(nil).Matches(0))
}

func TestAlertValidation(t *testing.T) {
	multiErr := errortypes.NewMultiError()
	(&report.ScheduleAlert{Operator: dbtype.OpContains}).Validate(multiErr)
	require.True(t, multiErr.HasErrors())

	multiErr = errortypes.NewMultiError()
	(&report.ScheduleAlert{Operator: dbtype.OpGreaterThan, Threshold: -1}).Validate(multiErr)
	require.True(t, multiErr.HasErrors())

	multiErr = errortypes.NewMultiError()
	(&report.ScheduleAlert{Operator: dbtype.OpGreaterThan, Threshold: 0}).Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func measureDigest(columnID string, total any) *services.ReportDigest {
	return &services.ReportDigest{
		Columns: []services.ReportResultColumn{
			textColumn("Customer"),
			{ID: columnID, Label: "Revenue", Type: reportcatalog.FieldDecimal},
		},
		Rows:   []services.ReportRow{{"ACME", total}},
		Totals: services.ReportRow{nil, total},
	}
}

// The gap this closes: "revenue fell below target" is a number no row count
// can express.
func TestMeasureAlertReadsTheGrandTotal(t *testing.T) {
	repo := &stubScheduleRepo{}
	threshold := 100_000.0
	alert := &report.ScheduleAlert{
		Operator: dbtype.OpLessThan,
		ColumnID: "c_revenue",
		Value:    &threshold,
	}

	// Below target — the condition holds, so it delivers.
	deliver, _ := alertActivities(repo).evaluateAlert(
		t.Context(), alertRun(3), alertSchedule(alert, false),
		measureDigest("c_revenue", decimal.NewFromInt(80_000)),
	)
	assert.True(t, deliver)

	// Above target — nothing to say.
	deliver, reason := alertActivities(repo).evaluateAlert(
		t.Context(), alertRun(3), alertSchedule(alert, false),
		measureDigest("c_revenue", decimal.NewFromInt(120_000)),
	)
	assert.False(t, deliver)
	assert.Equal(t, alertSkipConditionUnmet, reason)
}

// A measure alert must ignore the row count entirely — a report with rows can
// still be within target, and one with few rows can still be far outside it.
func TestMeasureAlertIgnoresRowCount(t *testing.T) {
	repo := &stubScheduleRepo{}
	threshold := 10.0
	alert := &report.ScheduleAlert{
		Operator: dbtype.OpGreaterThan,
		// A row-count threshold that would match is deliberately set, to prove
		// it is not what gets compared.
		Threshold: 0,
		ColumnID:  "c_revenue",
		Value:     &threshold,
	}

	deliver, _ := alertActivities(repo).evaluateAlert(
		t.Context(), alertRun(500), alertSchedule(alert, false),
		measureDigest("c_revenue", decimal.NewFromInt(5)),
	)

	assert.False(t, deliver, "500 rows but the total is under threshold")
}

// A measure that vanished from the report is not zero. Treating it as zero
// would fire "revenue below target" on a report that merely lost the column.
func TestMeasureAlertDoesNotFireOnAMissingTotal(t *testing.T) {
	repo := &stubScheduleRepo{}
	threshold := 100.0
	alert := &report.ScheduleAlert{
		Operator: dbtype.OpLessThan,
		ColumnID: "c_revenue",
		Value:    &threshold,
	}

	for name, digest := range map[string]*services.ReportDigest{
		"no digest at all": nil,
		"column absent":    measureDigest("c_other", decimal.NewFromInt(5)),
		"totals empty": {
			Columns: []services.ReportResultColumn{textColumn("Customer")},
			Rows:    []services.ReportRow{{"ACME"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			deliver, _ := alertActivities(repo).evaluateAlert(
				t.Context(), alertRun(3), alertSchedule(alert, false), digest,
			)
			assert.False(t, deliver)
		})
	}
}

func TestMeasureAlertValidation(t *testing.T) {
	infinite := math.Inf(1)

	missingValue := &report.ScheduleAlert{Operator: dbtype.OpLessThan, ColumnID: "c_revenue"}
	multiErr := errortypes.NewMultiError()
	missingValue.Validate(multiErr)
	assert.Contains(t, multiErr.Error(), "needs a threshold value")

	notFinite := &report.ScheduleAlert{
		Operator: dbtype.OpLessThan, ColumnID: "c_revenue", Value: &infinite,
	}
	multiErr = errortypes.NewMultiError()
	notFinite.Validate(multiErr)
	assert.Contains(t, multiErr.Error(), "finite number")
}

// A measure alert needs the totals row, so the run has to capture one even
// when nobody asked for an inline table.
func TestMeasureAlertOptsIntoCapture(t *testing.T) {
	measure := &report.ScheduleAlert{Operator: dbtype.OpLessThan, ColumnID: "c_revenue"}
	rowCount := &report.ScheduleAlert{Operator: dbtype.OpGreaterThan, Threshold: 0}

	assert.True(t, measure.TargetsMeasure())
	assert.False(t, rowCount.TargetsMeasure())
	assert.False(t, (*report.ScheduleAlert)(nil).TargetsMeasure())
}
