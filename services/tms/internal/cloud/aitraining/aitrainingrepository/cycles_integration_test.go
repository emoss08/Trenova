//go:build integration

package aitrainingrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const cycleWindowEnd = int64(1_790_000_000)

func newCycle(status aitraining.RetrainingStatus) *aitraining.RetrainingCycle {
	cycle := &aitraining.RetrainingCycle{
		Task:                 aicorrection.TaskShipmentDraftExtraction,
		Trigger:              aitraining.RetrainingTriggerScheduled,
		Status:               status,
		RequestedBy:          aitraining.RetrainingRequestedByScheduler,
		CapturedFrom:         cycleWindowEnd - 365*86400,
		CapturedTo:           cycleWindowEnd,
		NewSince:             cycleWindowEnd - 30*86400,
		NewExamples:          1200,
		MinNewExamples:       1000,
		MaxPerOrganization:   aitraining.DefaultMaxPerOrganization,
		ValidationPercent:    aitraining.DefaultValidationPercent,
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		MinAccuracyPercent:   85,
	}
	if status == aitraining.RetrainingStatusSkipped {
		cycle.SkipReason = aitraining.RetrainingSkipNotEnoughExamples
	}
	return cycle
}

func TestRetrainingCycles_OneOpenCycleAtATime(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	params := Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}
	cycles := NewCycles(params)

	open, err := cycles.Create(ctx, newCycle(aitraining.RetrainingStatusExporting))
	require.NoError(t, err)

	_, err = cycles.Create(ctx, newCycle(aitraining.RetrainingStatusExporting))
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err), "a second open cycle is refused with a clear message")

	skipped, err := cycles.Create(ctx, newCycle(aitraining.RetrainingStatusSkipped))
	require.NoError(t, err, "a skipped cycle is recorded while another is open")

	skipped.Status = aitraining.RetrainingStatusReady
	skipped.SkipReason = ""
	_, err = cycles.Update(ctx, skipped)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err), "an update cannot open a second cycle either")

	open.Status = aitraining.RetrainingStatusPassed
	_, err = cycles.Update(ctx, open)
	require.NoError(t, err)

	_, err = cycles.Create(ctx, newCycle(aitraining.RetrainingStatusExporting))
	require.NoError(t, err, "a finished cycle frees the slot")
}

func TestRetrainingCycles_UpdatesAreVersionChecked(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	cycles := NewCycles(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	created, err := cycles.Create(ctx, newCycle(aitraining.RetrainingStatusReady))
	require.NoError(t, err)

	first, err := cycles.GetByID(ctx, created.ID)
	require.NoError(t, err)
	second, err := cycles.GetByID(ctx, created.ID)
	require.NoError(t, err)

	require.NoError(t, first.Claim("gpu-1", cycleWindowEnd, 600))
	updated, err := cycles.Update(ctx, first)
	require.NoError(t, err)
	assert.Equal(t, second.Version+1, updated.Version)

	require.NoError(t, second.Claim("gpu-2", cycleWindowEnd, 600))
	_, err = cycles.Update(ctx, second)
	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err), "two trainers cannot both claim one cycle")

	stored, err := cycles.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "gpu-1", stored.Trainer)
	require.NotNil(t, stored.LeaseExpiresAt)
	assert.Equal(t, cycleWindowEnd+600, *stored.LeaseExpiresAt)
}

func TestRetrainingCycles_ListNewestFirstByStatus(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	cycles := NewCycles(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	statuses := []aitraining.RetrainingStatus{
		aitraining.RetrainingStatusSkipped,
		aitraining.RetrainingStatusPassed,
		aitraining.RetrainingStatusSkipped,
		aitraining.RetrainingStatusExporting,
	}
	created := make([]*aitraining.RetrainingCycle, 0, len(statuses))
	for i, status := range statuses {
		cycle, err := cycles.Create(ctx, newCycle(status))
		require.NoError(t, err)
		cols := buncolgen.RetrainingCycleColumns
		_, err = db.NewUpdate().
			Model((*aitraining.RetrainingCycle)(nil)).
			Set(cols.CreatedAt.Set(), cycleWindowEnd+int64(i)).
			Where(cols.ID.Eq(), cycle.ID).
			Exec(ctx)
		require.NoError(t, err)
		created = append(created, cycle)
	}

	all, err := cycles.List(ctx, repositories.ListRetrainingCyclesRequest{Limit: 10})
	require.NoError(t, err)
	require.Len(t, all, 4)
	assert.Equal(t, created[3].ID, all[0].ID, "newest first")
	assert.Equal(t, created[0].ID, all[3].ID)

	baseline, err := cycles.List(ctx, repositories.ListRetrainingCyclesRequest{
		Statuses: aitraining.BaselineRetrainingStatuses(),
		Limit:    1,
	})
	require.NoError(t, err)
	require.Len(t, baseline, 1)
	assert.Equal(t, created[3].ID, baseline[0].ID)

	skipped, err := cycles.List(ctx, repositories.ListRetrainingCyclesRequest{
		Statuses: []aitraining.RetrainingStatus{aitraining.RetrainingStatusSkipped},
	})
	require.NoError(t, err)
	require.Len(t, skipped, 2)
	assert.Equal(t, created[2].ID, skipped[0].ID)
}

func TestRetrainingCycles_KeepTheirHistoryWhenTheExportIsRemoved(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	params := Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}
	cycles := NewCycles(params)
	exports := NewExports(params)

	export, err := exports.Create(ctx, &aitraining.TrainingExport{
		Task:               aicorrection.TaskShipmentDraftExtraction,
		Status:             aitraining.ExportStatusCompleted,
		Format:             aitraining.ExampleFormat,
		CapturedFrom:       cycleWindowEnd - 86400,
		CapturedTo:         cycleWindowEnd,
		MaxPerOrganization: aitraining.DefaultMaxPerOrganization,
		ValidationPercent:  aitraining.DefaultValidationPercent,
		RequestedBy:        aitraining.RetrainingRequestedByScheduler,
	})
	require.NoError(t, err)

	cycle := newCycle(aitraining.RetrainingStatusReady)
	cycle.StartedExport(export.ID)
	created, err := cycles.Create(ctx, cycle)
	require.NoError(t, err)

	_, err = db.NewDelete().
		Model((*aitraining.TrainingExport)(nil)).
		Where(buncolgen.TrainingExportColumns.ID.Eq(), export.ID).
		Exec(ctx)
	require.NoError(t, err)

	stored, err := cycles.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, stored.ExportID, "the cycle stays; only the link to the export clears")
}

func TestRetrainingCycles_TheDatabaseHoldsTheRules(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	cycles := NewCycles(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	noReason := newCycle(aitraining.RetrainingStatusSkipped)
	noReason.SkipReason = ""
	_, err := cycles.Create(ctx, noReason)
	require.Error(t, err, "a skipped cycle must say why")

	training := newCycle(aitraining.RetrainingStatusTraining)
	_, err = cycles.Create(ctx, training)
	require.Error(t, err, "a training cycle must name its trainer and lease")

	window := newCycle(aitraining.RetrainingStatusPassed)
	window.NewSince = window.CapturedTo + 1
	_, err = cycles.Create(ctx, window)
	require.Error(t, err, "new corrections are counted inside the window")

	_, err = cycles.GetByID(ctx, pulid.MustNew("airc_"))
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
