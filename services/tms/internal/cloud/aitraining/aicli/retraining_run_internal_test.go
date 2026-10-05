package aicli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRetrainer struct {
	services.AIRetrainingService
	mu           sync.Mutex
	cycle        *aitraining.RetrainingCycle
	heartbeatErr error
	heartbeats   int
	recorded     *services.RecordAIRetrainingRequest
	failed       *services.FailAIRetrainingRequest
}

func (f *fakeRetrainer) ClaimNext(
	_ context.Context,
	req *services.ClaimAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	if f.cycle == nil {
		return nil, nil
	}
	f.cycle.Status = aitraining.RetrainingStatusTraining
	f.cycle.Trainer = req.Trainer
	f.cycle.Attempts++
	return f.cycle, nil
}

func (f *fakeRetrainer) Heartbeat(
	_ context.Context,
	_ *services.HeartbeatAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heartbeats++
	return f.cycle, f.heartbeatErr
}

func (f *fakeRetrainer) Record(
	_ context.Context,
	req *services.RecordAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	f.recorded = req
	if err := f.cycle.Settle(req.Trainer, req.Result, 1); err != nil {
		return nil, err
	}
	return f.cycle, nil
}

func (f *fakeRetrainer) FailTraining(
	_ context.Context,
	req *services.FailAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	f.failed = req
	return f.cycle, nil
}

type fakeRenderer struct {
	exportID pulid.ID
	mode     aiprovider.StructuredOutputMode
	err      error
}

func (f *fakeRenderer) Render(
	_ context.Context,
	req *services.RenderTrainingDatasetRequest,
) (*aitraining.DatasetManifest, error) {
	f.exportID = req.ExportID
	f.mode = req.StructuredOutputMode
	if f.err != nil {
		return nil, f.err
	}
	for _, name := range []string{aitraining.DatasetEvaluationFile, aitraining.DatasetManifestFile} {
		file, err := req.Sink.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err = file.Write([]byte("{}\n")); err != nil {
			return nil, err
		}
		if err = file.Close(); err != nil {
			return nil, err
		}
	}
	return &aitraining.DatasetManifest{
		PromptSHA256: "prompt-sha",
		Counts:       aitraining.DatasetCounts{Examples: 10, Train: 9, Validation: 1},
	}, nil
}

type fakeScorer struct {
	evaluation  string
	predictions string
}

func (f *fakeScorer) Score(
	_ context.Context,
	req *services.ScoreTrainingPredictionsRequest,
) (*aitraining.ScoreReport, error) {
	evaluation, err := io.ReadAll(req.Evaluation)
	if err != nil {
		return nil, err
	}
	predictions, err := io.ReadAll(req.Predictions)
	if err != nil {
		return nil, err
	}
	f.evaluation, f.predictions = string(evaluation), string(predictions)
	return &aitraining.ScoreReport{
		Examples: 1,
		Model:    aitraining.ScoreSide{Correct: 95, Scored: 100},
		Baseline: aitraining.ScoreSide{Correct: 90, Scored: 100},
	}, nil
}

func writeRun(out string, finalModel string) error {
	if err := os.MkdirAll(filepath.Join(out, runDataDirectory), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, runDataDirectory, "sft.jsonl"), []byte("x"), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, runPredictionsFile), []byte("prediction\n"), 0o600); err != nil {
		return err
	}
	record := `{"format":"run","final_model":null}`
	if finalModel != "" {
		if err := os.MkdirAll(finalModel, 0o700); err != nil {
			return err
		}
		record = `{"format":"run","final_model":"` + finalModel + `"}`
	}
	return os.WriteFile(filepath.Join(out, runRecordFile), []byte(record), 0o600)
}

type runFixture struct {
	run       *retrainingRun
	retrainer *fakeRetrainer
	renderer  *fakeRenderer
	scorer    *fakeScorer
	out       *bytes.Buffer
	cycle     *aitraining.RetrainingCycle
	requests  []pipelineRequest
}

func newRunFixture(t *testing.T) *runFixture {
	t.Helper()

	exportID := pulid.MustNew("aitx_")
	cycle := &aitraining.RetrainingCycle{
		ID:                   pulid.MustNew("airc_"),
		Status:               aitraining.RetrainingStatusReady,
		ExportID:             &exportID,
		StructuredOutputMode: aiprovider.StructuredOutputPrompted,
		MinAccuracyPercent:   85,
	}
	f := &runFixture{
		retrainer: &fakeRetrainer{cycle: cycle},
		renderer:  &fakeRenderer{},
		scorer:    &fakeScorer{},
		out:       &bytes.Buffer{},
		cycle:     cycle,
	}
	f.run = &retrainingRun{
		service:  f.retrainer,
		renderer: f.renderer,
		scorer:   f.scorer,
		pipeline: func(_ context.Context, req pipelineRequest) error {
			f.requests = append(f.requests, req)
			return writeRun(req.Out, filepath.Join(req.Out, "dpo", "model"))
		},
		out:       f.out,
		errOut:    io.Discard,
		trainer:   "gpu-1",
		config:    "/opt/configs/qwen.yaml",
		workDir:   t.TempDir(),
		heartbeat: time.Hour,
	}
	return f
}

func (f *runFixture) base() string {
	return filepath.Join(f.run.workDir, f.cycle.ID.String())
}

func TestRetrainingRunWithNothingToTrain(t *testing.T) {
	t.Parallel()

	f := newRunFixture(t)
	f.retrainer.cycle = nil

	require.NoError(t, f.run.execute(t.Context()))
	assert.Equal(t, "Nothing to train.\n", f.out.String())
	entries, err := os.ReadDir(f.run.workDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRetrainingRunTrainsScoresAndRecords(t *testing.T) {
	t.Parallel()

	f := newRunFixture(t)
	require.NoError(t, os.MkdirAll(filepath.Join(f.base(), retrainingRunDir), 0o700))
	stale := filepath.Join(f.base(), retrainingRunDir, "stale.txt")
	require.NoError(t, os.WriteFile(stale, []byte("old"), 0o600))

	require.NoError(t, f.run.execute(t.Context()))

	assert.Equal(t, *f.cycle.ExportID, f.renderer.exportID)
	assert.Equal(t, aiprovider.StructuredOutputPrompted, f.renderer.mode,
		"the dataset is rendered for the mode the cycle will be served with")
	require.Len(t, f.requests, 1)
	assert.Equal(t, pipelineRequest{
		Config:  "/opt/configs/qwen.yaml",
		Dataset: filepath.Join(f.base(), retrainingDatasetDir),
		Out:     filepath.Join(f.base(), retrainingRunDir),
	}, f.requests[0])
	assert.Equal(t, "{}\n", f.scorer.evaluation)
	assert.Equal(t, "prediction\n", f.scorer.predictions)

	require.NotNil(t, f.retrainer.recorded)
	result := f.retrainer.recorded.Result
	assert.Equal(t, "gpu-1", f.retrainer.recorded.Trainer)
	assert.Equal(t, filepath.Join(f.base(), retrainingRunDir, "dpo", "model"), result.ModelDirectory)
	assert.Equal(t, "prompt-sha", result.PromptSHA256)
	assert.Equal(t, "/opt/configs/qwen.yaml", result.TrainingConfig)
	assert.Nil(t, f.retrainer.failed)
	assert.Equal(t, aitraining.RetrainingStatusPassed, f.cycle.Status)
	assert.Contains(t, f.out.String(), "The model passed the gate")

	assert.NoFileExists(t, stale, "an earlier attempt's files are cleared")
	assert.NoDirExists(t, filepath.Join(f.base(), retrainingDatasetDir), "the dataset is customer data")
	assert.NoDirExists(t, filepath.Join(f.base(), retrainingRunDir, runDataDirectory))
	assert.DirExists(t, result.ModelDirectory, "the trained model is kept")
}

func TestRetrainingRunKeepsDataWhenAsked(t *testing.T) {
	t.Parallel()

	f := newRunFixture(t)
	f.run.keepData = true

	require.NoError(t, f.run.execute(t.Context()))
	assert.DirExists(t, filepath.Join(f.base(), retrainingDatasetDir))
	assert.DirExists(t, filepath.Join(f.base(), retrainingRunDir, runDataDirectory))
}

func TestRetrainingRunRecordsAFailedPipeline(t *testing.T) {
	t.Parallel()

	f := newRunFixture(t)
	errExit := errors.New("exit status 1")
	f.run.pipeline = func(context.Context, pipelineRequest) error { return errExit }

	err := f.run.execute(t.Context())
	require.ErrorIs(t, err, errExit)
	require.NotNil(t, f.retrainer.failed)
	assert.Equal(t, "trenova-finetune run: exit status 1", f.retrainer.failed.Message)
	assert.Nil(t, f.retrainer.recorded)
	assert.NoDirExists(t, filepath.Join(f.base(), retrainingDatasetDir))
}

func TestRetrainingRunRecordsAFailedRender(t *testing.T) {
	t.Parallel()

	f := newRunFixture(t)
	f.renderer.err = errors.New("part checksum mismatch")

	err := f.run.execute(t.Context())
	require.Error(t, err)
	require.NotNil(t, f.retrainer.failed)
	assert.Contains(t, f.retrainer.failed.Message, "part checksum mismatch")
	assert.Empty(t, f.requests, "nothing is trained on a dataset that did not render")
}

func TestRetrainingRunNeedsAFinalModel(t *testing.T) {
	t.Parallel()

	f := newRunFixture(t)
	f.run.pipeline = func(_ context.Context, req pipelineRequest) error {
		return writeRun(req.Out, "")
	}

	err := f.run.execute(t.Context())
	require.ErrorIs(t, err, errNoFinalModel)
	require.NotNil(t, f.retrainer.failed)
	assert.Nil(t, f.retrainer.recorded)
}

func TestRetrainingRunStopsWhenTheLeaseIsLost(t *testing.T) {
	t.Parallel()

	f := newRunFixture(t)
	f.run.heartbeat = time.Millisecond
	f.retrainer.heartbeatErr = aitraining.ErrRetrainingLeaseLost
	stopped := make(chan struct{})
	f.run.pipeline = func(ctx context.Context, _ pipelineRequest) error {
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	}

	err := f.run.execute(t.Context())
	require.ErrorIs(t, err, aitraining.ErrRetrainingLeaseLost)
	<-stopped
	assert.Nil(t, f.retrainer.recorded, "a trainer without the lease records nothing")
	assert.Nil(t, f.retrainer.failed)
	assert.NoDirExists(t, filepath.Join(f.base(), retrainingDatasetDir))
}

func TestRetrainingRunKeepsTrainingThroughATransientHeartbeatError(t *testing.T) {
	t.Parallel()

	f := newRunFixture(t)
	f.run.heartbeat = time.Millisecond
	f.retrainer.heartbeatErr = errors.New("database unavailable")
	f.run.pipeline = func(_ context.Context, req pipelineRequest) error {
		for {
			f.retrainer.mu.Lock()
			beats := f.retrainer.heartbeats
			f.retrainer.mu.Unlock()
			if beats >= 3 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		return writeRun(req.Out, filepath.Join(req.Out, "sft", "model"))
	}

	require.NoError(t, f.run.execute(t.Context()))
	require.NotNil(t, f.retrainer.recorded)
}

func TestHeartbeatInterval(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 30*time.Minute, heartbeatInterval(2*time.Hour))
	assert.Equal(t, time.Minute, heartbeatInterval(2*time.Minute), "never faster than once a minute")
}
