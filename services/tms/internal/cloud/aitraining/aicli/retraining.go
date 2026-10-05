package aicli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/spf13/cobra"
)

var (
	retrainRequestedBy string
	retrainNote        string
	retrainListLimit   int
	retrainConfig      string
	retrainWorkDir     string
	retrainFinetuneDir string
	retrainTrainer     string
	retrainUV          string
	retrainKeepData    bool
)

var retrainingCmd = &cobra.Command{
	Use:   "retraining",
	Short: "Scheduled retraining of the document extraction model",
	Long: `Scheduled retraining of the document extraction model.

Every Monday the worker decides whether enough new corrections have been
confirmed, by consenting organizations, since the last retraining. When they
have, it starts a training export over the lookback window and records a
retraining cycle. A GPU machine running "trenova ai retraining run" on a timer
claims the cycle once its export finishes, renders the datasets, trains with
trenova-finetune, scores the model against production on the validation set and
records whether it passed the gate. The policy is the aiRetraining section of
the configuration.`,
}

var retrainingStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a retraining now, whatever the schedule would decide",
	Long: `Start a retraining now.

The minimum interval and the minimum of new corrections are not applied, and the
schedule does not need to be enabled. It still waits for an open cycle or a
running training export to finish.

Examples:
  trenova ai retraining start --requested-by "Jordan Lee" --note "new customer layouts"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		requestedBy := strings.TrimSpace(retrainRequestedBy)
		if requestedBy == "" {
			if current, err := user.Current(); err == nil {
				requestedBy = current.Username
			}
		}

		return withRetrainer(cmd.Context(), func(ctx context.Context, retrainer services.AIRetrainingService) error {
			cycle, err := retrainer.Plan(ctx, &services.PlanAIRetrainingRequest{
				Manual:      true,
				RequestedBy: requestedBy,
				Note:        retrainNote,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Started retraining cycle %s (export %s)\n", cycle.ID, exportLabel(cycle))
			fmt.Fprintf(cmd.OutOrStdout(), "Follow it with: trenova ai retraining status %s\n", cycle.ID)
			return nil
		})
	},
}

var retrainingListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent retraining cycles, skipped ones included",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return withRetrainer(cmd.Context(), func(ctx context.Context, retrainer services.AIRetrainingService) error {
			if _, err := retrainer.Reconcile(ctx); err != nil {
				return err
			}
			cycles, err := retrainer.List(ctx, retrainListLimit)
			if err != nil {
				return err
			}
			printCycles(cmd.OutOrStdout(), cycles)
			return nil
		})
	},
}

var retrainingStatusCmd = &cobra.Command{
	Use:   "status <cycle-id>",
	Short: "Show one retraining cycle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := cycleID(args[0])
		if err != nil {
			return err
		}

		return withRetrainer(cmd.Context(), func(ctx context.Context, retrainer services.AIRetrainingService) error {
			if _, reconcileErr := retrainer.Reconcile(ctx); reconcileErr != nil {
				return reconcileErr
			}
			cycle, getErr := retrainer.Get(ctx, id)
			if getErr != nil {
				return getErr
			}
			printCycle(cmd.OutOrStdout(), cycle)
			return nil
		})
	},
}

var retrainingCancelCmd = &cobra.Command{
	Use:   "cancel <cycle-id>",
	Short: "Cancel an exporting, ready or training cycle",
	Long: `Cancel an exporting, ready or training cycle.

A cycle still exporting cancels its export too. A trainer working on the cycle
stops at its next heartbeat and records nothing.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := cycleID(args[0])
		if err != nil {
			return err
		}

		return withRetrainer(cmd.Context(), func(ctx context.Context, retrainer services.AIRetrainingService) error {
			canceled, cancelErr := retrainer.Cancel(ctx, id)
			if cancelErr != nil {
				return cancelErr
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Canceled retraining cycle %s\n", canceled.ID)
			return nil
		})
	},
}

var retrainingRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Claim the next retraining cycle, train it and record its score (GPU machine)",
	Long: `Claim the retraining cycle whose export has finished, and train it.

Run this on the GPU machine from a timer (hourly is enough); it exits at once
when there is nothing to train. For a claimed cycle it:

  1. renders the export into <work-dir>/<cycle-id>/dataset, honouring every
     consent withdrawal made since the export ran;
  2. runs "uv run --directory <finetune-dir> trenova-finetune run" with
     --config into <work-dir>/<cycle-id>/run;
  3. scores the predictions against the production model with the same
     scorer as "trenova ai fine-tune score", and records whether the model
     passed the cycle's gate.

The lease on the cycle is extended while training runs. A trainer that loses
it (the cycle was canceled, or another trainer took over an expired lease)
stops the pipeline and records nothing. A cycle whose trainer disappeared is
claimed again once its lease expires, and starts from a fresh dataset.

The rendered dataset and the run's built training data are anonymized customer
data; both are deleted when the run ends unless --keep-data is given. The
trained model is kept.

Examples:
  trenova ai retraining run --config ml/extraction-finetune/configs/qwen2.5-7b-instruct.yaml \
    --work-dir /data/retraining`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		paths, err := runPaths()
		if err != nil {
			return err
		}
		trainer := strings.TrimSpace(retrainTrainer)
		if trainer == "" {
			if trainer, err = os.Hostname(); err != nil {
				return fmt.Errorf("name this trainer with --trainer: %w", err)
			}
		}

		var (
			retrainer services.AIRetrainingService
			renderer  services.AITrainingDatasetRenderer
		)
		return withCommandApp(cmd.Context(), []any{&retrainer, &renderer}, func(ctx context.Context) error {
			run := &retrainingRun{
				service:   retrainer,
				renderer:  renderer,
				scorer:    newScorer(),
				pipeline:  uvPipeline(retrainUV, paths.finetuneDir, cmd.OutOrStdout(), cmd.ErrOrStderr()),
				out:       cmd.OutOrStdout(),
				errOut:    cmd.ErrOrStderr(),
				trainer:   trainer,
				config:    paths.config,
				workDir:   paths.workDir,
				keepData:  retrainKeepData,
				heartbeat: heartbeatInterval(retrainer.LeaseDuration()),
			}
			return run.execute(ctx)
		})
	},
}

type retrainingPaths struct {
	config      string
	workDir     string
	finetuneDir string
}

func runPaths() (retrainingPaths, error) {
	if strings.TrimSpace(retrainConfig) == "" {
		return retrainingPaths{}, errors.New("--config is required")
	}
	if strings.TrimSpace(retrainWorkDir) == "" {
		return retrainingPaths{}, errWorkDirIsRequired
	}

	config, err := existingPath(retrainConfig, false)
	if err != nil {
		return retrainingPaths{}, err
	}
	finetuneDir, err := existingPath(retrainFinetuneDir, true)
	if err != nil {
		return retrainingPaths{}, err
	}
	workDir, err := filepath.Abs(strings.TrimSpace(retrainWorkDir))
	if err != nil {
		return retrainingPaths{}, fmt.Errorf("resolve %s: %w", retrainWorkDir, err)
	}
	if err = os.MkdirAll(workDir, datasetDirMode); err != nil {
		return retrainingPaths{}, fmt.Errorf("create %s: %w", workDir, err)
	}

	return retrainingPaths{config: config, workDir: workDir, finetuneDir: finetuneDir}, nil
}

func existingPath(raw string, directory bool) (string, error) {
	path, err := filepath.Abs(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", raw, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("check %s: %w", path, err)
	}
	if info.IsDir() != directory {
		kind := "a file"
		if directory {
			kind = "a directory"
		}
		return "", fmt.Errorf("%s is not %s", path, kind)
	}

	return path, nil
}

func withRetrainer(
	parent context.Context,
	run func(ctx context.Context, retrainer services.AIRetrainingService) error,
) error {
	var retrainer services.AIRetrainingService
	return withCommandApp(parent, []any{&retrainer}, func(ctx context.Context) error {
		return run(ctx, retrainer)
	})
}

func printCycles(out io.Writer, cycles []*aitraining.RetrainingCycle) {
	if len(cycles) == 0 {
		fmt.Fprintln(out, "No retraining cycles yet.")
		return
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tTRIGGER\tNEW\tEXPORT\tMODEL\tBASELINE\tCREATED")
	for _, c := range cycles {
		status := c.Status.String()
		if c.SkipReason != "" {
			status += " (" + c.SkipReason.String() + ")"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d/%d\t%s\t%s\t%s\t%s\n",
			c.ID, status, c.Trigger, c.NewExamples, c.MinNewExamples, exportLabel(c),
			accuracyLabel(c.ModelCorrect, c.ModelScored, c.ModelAccuracy()),
			accuracyLabel(c.BaselineCorrect, c.BaselineScored, c.BaselineAccuracy()),
			formatUnix(c.CreatedAt))
	}
	_ = w.Flush()
}

func printCycle(out io.Writer, c *aitraining.RetrainingCycle) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Cycle\t%s\n", c.ID)
	fmt.Fprintf(w, "Status\t%s\n", c.Status)
	if c.SkipReason != "" {
		fmt.Fprintf(w, "Skipped\t%s\n", c.SkipReason.Message())
	}
	fmt.Fprintf(w, "Trigger\t%s\n", c.Trigger)
	fmt.Fprintf(w, "Requested by\t%s\n", c.RequestedBy)
	if c.Note != "" {
		fmt.Fprintf(w, "Note\t%s\n", c.Note)
	}
	fmt.Fprintf(w, "Window\t%s to %s\n", formatUnix(c.CapturedFrom), formatUnix(c.CapturedTo))
	fmt.Fprintf(w, "New corrections\t%d since %s (%d needed)\n",
		c.NewExamples, formatUnix(c.NewSince), c.MinNewExamples)
	if c.DriftingProviders > 0 {
		fmt.Fprintf(w, "Drifting providers\t%d\n", c.DriftingProviders)
	}
	fmt.Fprintf(w, "Export\t%s (up to %d per organization, %d%% validation)\n",
		exportLabel(c), c.MaxPerOrganization, c.ValidationPercent)
	fmt.Fprintf(w, "Structured output\t%s\n", c.StructuredOutputMode)
	fmt.Fprintf(w, "Gate\tat least %d%%, at most %d points below production\n",
		c.MinAccuracyPercent, c.MaxRegressionPoints)
	if c.Trainer != "" {
		fmt.Fprintf(w, "Trainer\t%s (attempt %d)\n", c.Trainer, c.Attempts)
	}
	if c.LeaseExpiresAt != nil {
		fmt.Fprintf(w, "Lease until\t%s\n", formatUnix(*c.LeaseExpiresAt))
	}
	if c.TrainingConfig != "" {
		fmt.Fprintf(w, "Config\t%s\n", c.TrainingConfig)
	}
	if c.ModelDirectory != "" {
		fmt.Fprintf(w, "Model\t%s\n", c.ModelDirectory)
	}
	if c.ModelScored > 0 || c.BaselineScored > 0 {
		fmt.Fprintf(w, "Validation examples\t%d\n", c.Examples)
		fmt.Fprintf(w, "Model accuracy\t%s\n", accuracyLabel(c.ModelCorrect, c.ModelScored, c.ModelAccuracy()))
		fmt.Fprintf(w, "Production accuracy\t%s\n",
			accuracyLabel(c.BaselineCorrect, c.BaselineScored, c.BaselineAccuracy()))
	}
	if c.GateMessage != "" {
		fmt.Fprintf(w, "Gate result\t%s\n", c.GateMessage)
	}
	if c.FailureMessage != "" {
		fmt.Fprintf(w, "Failure\t%s\n", c.FailureMessage)
	}
	fmt.Fprintf(w, "Created\t%s\n", formatUnix(c.CreatedAt))
	if c.FinishedAt != nil {
		fmt.Fprintf(w, "Finished\t%s\n", formatUnix(*c.FinishedAt))
	}
	_ = w.Flush()
}

func printRetrainingOutcome(out io.Writer, c *aitraining.RetrainingCycle) {
	fmt.Fprintln(out)
	printCycle(out, c)
	fmt.Fprintln(out)
	if c.Status != aitraining.RetrainingStatusPassed {
		fmt.Fprintln(out, "The model did not pass the gate; production is unchanged.")
		return
	}
	fmt.Fprintln(out, "The model passed the gate. Next: serve it with trenova-finetune serve,")
	fmt.Fprintln(out, "register it as an AI provider after the current one, run an evaluation on")
	fmt.Fprintln(out, "the golden set, then shadow and roll it out from AI Control.")
}

func accuracyLabel(correct, scored int, accuracy float64) string {
	if scored == 0 {
		return "-"
	}

	return fmt.Sprintf("%s (%d/%d)", formatPercent(accuracy), correct, scored)
}

func init() {
	retrainingStartCmd.Flags().StringVar(&retrainRequestedBy, "requested-by", "",
		"operator starting the retraining; defaults to the current OS user")
	retrainingStartCmd.Flags().StringVar(&retrainNote, "note", "", "why the model is being retrained")
	retrainingListCmd.Flags().IntVar(&retrainListLimit, "limit", defaultListSize, "how many cycles to list")

	retrainingRunCmd.Flags().StringVar(&retrainConfig, "config", "", "the trenova-finetune configuration to train with")
	retrainingRunCmd.Flags().StringVar(&retrainWorkDir, "work-dir", "",
		"directory for each cycle's dataset, run and model")
	retrainingRunCmd.Flags().StringVar(&retrainFinetuneDir, "finetune-dir", defaultFinetuneFolder,
		"the ml/extraction-finetune project uv runs the pipeline from")
	retrainingRunCmd.Flags().StringVar(&retrainTrainer, "trainer", "",
		"name recorded on the cycle while this machine trains it; defaults to the host name")
	retrainingRunCmd.Flags().StringVar(&retrainUV, "uv", defaultUVCommand, "the uv executable")
	retrainingRunCmd.Flags().BoolVar(&retrainKeepData, "keep-data", false,
		"keep the rendered dataset and the run's built training data")

	retrainingCmd.AddCommand(
		retrainingStartCmd,
		retrainingListCmd,
		retrainingStatusCmd,
		retrainingCancelCmd,
		retrainingRunCmd,
	)
	AICmd.AddCommand(retrainingCmd)
}
