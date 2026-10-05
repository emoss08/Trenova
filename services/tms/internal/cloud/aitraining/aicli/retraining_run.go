package aicli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	runRecordFile         = "run.json"
	runPredictionsFile    = "predictions.jsonl"
	runDataDirectory      = "data"
	retrainingDatasetDir  = "dataset"
	retrainingRunDir      = "run"
	heartbeatsPerLease    = 4
	minHeartbeatInterval  = time.Minute
	pipelineStopGrace     = 30 * time.Second
	finetuneCommand       = "trenova-finetune"
	defaultUVCommand      = "uv"
	defaultFinetuneFolder = "ml/extraction-finetune"
)

var (
	errNoFinalModel      = errors.New("the fine-tuning run recorded no final model")
	errWorkDirIsRequired = errors.New("--work-dir is required")
)

type pipelineRequest struct {
	Config  string
	Dataset string
	Out     string
}

type pipelineRunner func(ctx context.Context, req pipelineRequest) error

type retrainingRun struct {
	service   services.AIRetrainingService
	renderer  services.AITrainingDatasetRenderer
	scorer    services.AITrainingScorer
	pipeline  pipelineRunner
	out       io.Writer
	errOut    io.Writer
	trainer   string
	config    string
	workDir   string
	keepData  bool
	heartbeat time.Duration
}

type runRecord struct {
	FinalModel *string `json:"final_model"`
}

func (r *retrainingRun) execute(ctx context.Context) error {
	if r.workDir == "" {
		return errWorkDirIsRequired
	}

	cycle, err := r.service.ClaimNext(ctx, &services.ClaimAIRetrainingRequest{Trainer: r.trainer})
	if err != nil {
		return err
	}
	if cycle == nil {
		fmt.Fprintln(r.out, "Nothing to train.")
		return nil
	}
	fmt.Fprintf(r.out, "Claimed retraining cycle %s (export %s, attempt %d)\n",
		cycle.ID, exportLabel(cycle), cycle.Attempts)

	base := filepath.Join(r.workDir, cycle.ID.String())
	datasetDir := filepath.Join(base, retrainingDatasetDir)
	runDir := filepath.Join(base, retrainingRunDir)

	runCtx, cancel := context.WithCancelCause(ctx)
	var wg sync.WaitGroup
	wg.Go(func() { r.keepLease(runCtx, cancel, cycle) })

	result, trainErr := r.train(runCtx, cycle, base, datasetDir, runDir)
	cause := context.Cause(runCtx)
	cancel(nil)
	wg.Wait()

	if !r.keepData {
		r.removeData(datasetDir, filepath.Join(runDir, runDataDirectory))
	}

	if cause != nil && errors.Is(cause, aitraining.ErrRetrainingLeaseLost) {
		return fmt.Errorf("stopped training cycle %s: %w", cycle.ID, cause)
	}
	if trainErr != nil {
		if _, failErr := r.service.FailTraining(ctx, &services.FailAIRetrainingRequest{
			CycleID: cycle.ID,
			Trainer: r.trainer,
			Message: trainErr.Error(),
		}); failErr != nil {
			return errors.Join(trainErr, fmt.Errorf("record the failure: %w", failErr))
		}
		return trainErr
	}

	recorded, err := r.service.Record(ctx, &services.RecordAIRetrainingRequest{
		CycleID: cycle.ID,
		Trainer: r.trainer,
		Result:  result,
	})
	if err != nil {
		return fmt.Errorf("record the result of cycle %s: %w", cycle.ID, err)
	}

	printRetrainingOutcome(r.out, recorded)

	return nil
}

func (r *retrainingRun) train(
	ctx context.Context,
	cycle *aitraining.RetrainingCycle,
	base, datasetDir, runDir string,
) (*aitraining.RetrainingResult, error) {
	if cycle.ExportID == nil {
		return nil, errors.New("the cycle has no training export")
	}
	if err := os.RemoveAll(base); err != nil {
		return nil, fmt.Errorf("clear %s from an earlier attempt: %w", base, err)
	}

	sink, err := newDirectorySink(datasetDir)
	if err != nil {
		return nil, err
	}
	manifest, err := r.renderer.Render(ctx, &services.RenderTrainingDatasetRequest{
		ExportID:             *cycle.ExportID,
		StructuredOutputMode: cycle.StructuredOutputMode,
		Sink:                 sink,
	})
	if err != nil {
		sink.abort()
		return nil, fmt.Errorf("render export %s: %w", *cycle.ExportID, err)
	}
	if err = sink.commit(); err != nil {
		sink.abort()
		return nil, err
	}
	fmt.Fprintf(r.out, "Rendered %d example(s) (train %d, validation %d) to %s\n",
		manifest.Counts.Examples, manifest.Counts.Train, manifest.Counts.Validation, datasetDir)

	if err = r.pipeline(ctx, pipelineRequest{Config: r.config, Dataset: datasetDir, Out: runDir}); err != nil {
		return nil, fmt.Errorf("%s run: %w", finetuneCommand, err)
	}

	model, err := finalModel(runDir)
	if err != nil {
		return nil, err
	}
	report, err := r.score(ctx, datasetDir, runDir)
	if err != nil {
		return nil, err
	}

	return &aitraining.RetrainingResult{
		TrainingConfig: r.config,
		RunDirectory:   runDir,
		ModelDirectory: model,
		PromptSHA256:   manifest.PromptSHA256,
		Report:         report,
	}, nil
}

func (r *retrainingRun) score(
	ctx context.Context,
	datasetDir, runDir string,
) (*aitraining.ScoreReport, error) {
	evaluationPath := filepath.Join(datasetDir, aitraining.DatasetEvaluationFile)
	evaluation, err := os.Open(evaluationPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", evaluationPath, err)
	}
	defer evaluation.Close()

	predictionsPath := filepath.Join(runDir, runPredictionsFile)
	predictions, err := os.Open(predictionsPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", predictionsPath, err)
	}
	defer predictions.Close()

	report, err := r.scorer.Score(ctx, &services.ScoreTrainingPredictionsRequest{
		Evaluation:  evaluation,
		Predictions: predictions,
	})
	if err != nil {
		return nil, fmt.Errorf("score the predictions: %w", err)
	}

	return report, nil
}

func finalModel(runDir string) (string, error) {
	path := filepath.Join(runDir, runRecordFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	var record runRecord
	if err = sonic.Unmarshal(raw, &record); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	if record.FinalModel == nil || *record.FinalModel == "" {
		return "", fmt.Errorf("%w (%s)", errNoFinalModel, path)
	}

	return *record.FinalModel, nil
}

func (r *retrainingRun) keepLease(
	ctx context.Context,
	stop context.CancelCauseFunc,
	cycle *aitraining.RetrainingCycle,
) {
	ticker := time.NewTicker(r.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := r.service.Heartbeat(ctx, &services.HeartbeatAIRetrainingRequest{
				CycleID: cycle.ID,
				Trainer: r.trainer,
			})
			switch {
			case err == nil:
			case errors.Is(err, aitraining.ErrRetrainingLeaseLost), errortypes.IsNotFoundError(err):
				stop(fmt.Errorf("%w: %w", aitraining.ErrRetrainingLeaseLost, err))
				return
			case ctx.Err() == nil:
				fmt.Fprintf(r.errOut, "Could not extend the lease on cycle %s: %v\n", cycle.ID, err)
			}
		}
	}
}

func (r *retrainingRun) removeData(paths ...string) {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintf(r.errOut, "Could not remove %s: %v\n", path, err)
		}
	}
}

func heartbeatInterval(lease time.Duration) time.Duration {
	return max(lease/heartbeatsPerLease, minHeartbeatInterval)
}

func uvPipeline(uv, finetuneDir string, stdout, stderr io.Writer) pipelineRunner {
	return func(ctx context.Context, req pipelineRequest) error {
		cmd := exec.CommandContext(ctx, uv,
			"run", "--directory", finetuneDir,
			finetuneCommand, "run",
			"--config", req.Config,
			"--dataset", req.Dataset,
			"--out", req.Out,
		)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
		cmd.WaitDelay = pipelineStopGrace

		return cmd.Run()
	}
}

func exportLabel(cycle *aitraining.RetrainingCycle) string {
	if cycle.ExportID == nil {
		return "none"
	}

	return cycle.ExportID.String()
}
