package ai

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/aidocumentservice"
	"github.com/emoss08/trenova/internal/core/services/aitrainingservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/extractionevaljobs"
	"github.com/spf13/cobra"
)

const percent = 100

var (
	scoreEvaluationPath  string
	scorePredictionsPath string
	scoreJSON            bool
	scoreMaxRegression   float64
	scoreMinAccuracy     float64
)

var errScoreGate = errors.New("the fine-tuned model did not pass the score gate")

var fineTuneCmd = &cobra.Command{
	Use:   "fine-tune",
	Short: "Checks for extraction models fine-tuned on training exports",
}

var fineTuneScoreCmd = &cobra.Command{
	Use:   "score",
	Short: "Score a model's predictions on a rendered validation set",
	Long: `Score a model's predictions on the validation set of a rendered dataset.

Every reply is read the way production reads one and scored with the same
field comparison AI corrections use, beside the production model's original
prediction for the same document. A prediction that is missing, failed or
cannot be read counts as every field missed.

The command fails when the model's accuracy is below --min-accuracy, or more
than --max-regression below the baseline's.

Examples:
  trenova ai fine-tune score --eval ./datasets/aitx_01J/eval-validation.jsonl --predictions ./runs/qwen/predictions.jsonl
  trenova ai fine-tune score --eval ... --predictions ... --json > score.json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		evaluation, err := os.Open(scoreEvaluationPath)
		if err != nil {
			return fmt.Errorf("open %s: %w", scoreEvaluationPath, err)
		}
		defer evaluation.Close()
		predictions, err := os.Open(scorePredictionsPath)
		if err != nil {
			return fmt.Errorf("open %s: %w", scorePredictionsPath, err)
		}
		defer predictions.Close()

		scorer := aitrainingservice.NewScorer(
			extractionevaljobs.NewReplyReader(aidocumentservice.NewContract()),
		)
		report, err := scorer.Score(cmd.Context(), &services.ScoreTrainingPredictionsRequest{
			Evaluation:  evaluation,
			Predictions: predictions,
		})
		if err != nil {
			return err
		}

		if scoreJSON {
			encoded, encodeErr := sonic.MarshalIndent(report, "", "  ")
			if encodeErr != nil {
				return fmt.Errorf("encode score: %w", encodeErr)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
		} else {
			printScore(cmd.OutOrStdout(), report)
		}

		return scoreGate(report, scoreMinAccuracy, scoreMaxRegression)
	},
}

func scoreGate(report *aitraining.ScoreReport, minAccuracy, maxRegression float64) error {
	if report.Examples == 0 {
		return fmt.Errorf("%w: the validation set is empty", errScoreGate)
	}
	if report.Model.Accuracy < minAccuracy {
		return fmt.Errorf("%w: accuracy %.2f%% is below the minimum %.2f%%",
			errScoreGate, report.Model.Accuracy*percent, minAccuracy*percent)
	}
	if report.AccuracyDelta < -maxRegression {
		return fmt.Errorf("%w: accuracy is %.2f points below the baseline, more than the %.2f allowed",
			errScoreGate, -report.AccuracyDelta*percent, maxRegression*percent)
	}

	return nil
}

func printScore(out io.Writer, report *aitraining.ScoreReport) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Examples\t%d\n", report.Examples)
	fmt.Fprintf(w, "Predictions read\t%d\n", report.Predicted)
	fmt.Fprintf(w, "Missing / failed / unreadable\t%d / %d / %d\n",
		report.MissingPredictions, report.FailedPredictions, report.UnreadablePredictions)
	if report.UnknownPredictions > 0 {
		fmt.Fprintf(w, "Predictions for unknown examples\t%d\n", report.UnknownPredictions)
	}
	fmt.Fprintf(w, "Model accuracy\t%s (%d of %d)\n",
		formatPercent(report.Model.Accuracy), report.Model.Correct, report.Model.Scored)
	fmt.Fprintf(w, "Baseline accuracy\t%s (%d of %d)\n",
		formatPercent(report.Baseline.Accuracy), report.Baseline.Correct, report.Baseline.Scored)
	fmt.Fprintf(w, "Change\t%+.2f points\n", report.AccuracyDelta*percent)
	_ = w.Flush()

	baseline := make(map[string]aicorrection.FieldAccuracy, len(report.Baseline.Fields))
	for _, field := range report.Baseline.Fields {
		baseline[field.Key] = field
	}
	keys := make([]string, 0, len(report.Model.Fields))
	model := make(map[string]aicorrection.FieldAccuracy, len(report.Model.Fields))
	for _, field := range report.Model.Fields {
		model[field.Key] = field
		keys = append(keys, field.Key)
	}
	for key := range baseline {
		if _, ok := model[key]; !ok {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	fmt.Fprintln(out)
	fields := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(fields, "FIELD\tMODEL\tBASELINE\tCHANGE\tSCORED")
	for _, key := range keys {
		m, b := model[key], baseline[key]
		fmt.Fprintf(fields, "%s\t%s\t%s\t%+.2f\t%d\n",
			key, formatPercent(m.Accuracy), formatPercent(b.Accuracy), (m.Accuracy-b.Accuracy)*percent, m.Scored)
	}
	_ = fields.Flush()
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value*percent)
}

func init() {
	fineTuneScoreCmd.Flags().StringVar(&scoreEvaluationPath, "eval", "", "the rendered eval-validation.jsonl")
	fineTuneScoreCmd.Flags().StringVar(&scorePredictionsPath, "predictions", "", "the model's predictions.jsonl")
	fineTuneScoreCmd.Flags().BoolVar(&scoreJSON, "json", false, "print the full report as JSON")
	fineTuneScoreCmd.Flags().Float64Var(&scoreMaxRegression, "max-regression", 0,
		"how far below the baseline's accuracy the model may score, as a fraction (0.01 is one point)")
	fineTuneScoreCmd.Flags().Float64Var(&scoreMinAccuracy, "min-accuracy", 0,
		"the lowest accuracy the model may score, as a fraction")
	_ = fineTuneScoreCmd.MarkFlagRequired("eval")
	_ = fineTuneScoreCmd.MarkFlagRequired("predictions")
	fineTuneCmd.AddCommand(fineTuneScoreCmd)
	AICmd.AddCommand(fineTuneCmd)
}
