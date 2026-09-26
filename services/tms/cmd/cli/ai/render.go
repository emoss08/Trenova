package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/spf13/cobra"
)

var (
	renderOutput           string
	renderStructuredOutput string
	renderDropUnverified   bool
)

var trainingExportRenderCmd = &cobra.Command{
	Use:   "render <export-id>",
	Short: "Render a finished export into fine-tuning datasets",
	Long: `Render a finished export into the datasets the fine-tuning pipeline reads.

Every part and the manifest are checked against the checksums the export
recorded, examples whose organization has since withdrawn consent are left
out, and each example is rendered with the production extraction prompt, so
the model is trained on exactly what it will be sent. The output directory
must not exist; it is written in full or not at all.

--structured-output must match how the fine-tuned model will be registered as
an AI provider: JSONSchema (vLLM with guided decoding, the default), JSONMode,
or Prompted.

Examples:
  trenova ai training-export render aitx_01J... --out ./datasets/aitx_01J
  trenova ai training-export render aitx_01J... --out ./datasets/strict --drop-unverified`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := exportID(args[0])
		if err != nil {
			return err
		}
		mode := aiprovider.StructuredOutputMode(strings.TrimSpace(renderStructuredOutput))
		if !mode.IsValid() {
			return fmt.Errorf("--structured-output must be JSONSchema, JSONMode or Prompted, not %q", renderStructuredOutput)
		}

		sink, err := newDirectorySink(renderOutput)
		if err != nil {
			return err
		}

		var renderer services.AITrainingDatasetRenderer
		runErr := withCommandApp(cmd.Context(), []any{&renderer}, func(ctx context.Context) error {
			dataset, renderErr := renderer.Render(ctx, &services.RenderTrainingDatasetRequest{
				ExportID:             id,
				StructuredOutputMode: mode,
				KeepUnverified:       !renderDropUnverified,
				Sink:                 sink,
			})
			if renderErr != nil {
				return renderErr
			}
			if commitErr := sink.commit(); commitErr != nil {
				return commitErr
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Rendered %d example(s) to %s\n", dataset.Counts.Examples, sink.dir)
			fmt.Fprintf(out, "  train %d, validation %d, preference pairs %d\n",
				dataset.Counts.Train, dataset.Counts.Validation, dataset.Counts.Preference)
			if dataset.Counts.WithdrawnExcluded > 0 {
				fmt.Fprintf(out, "  %d example(s) left out: their organization has withdrawn consent\n",
					dataset.Counts.WithdrawnExcluded)
			}
			fmt.Fprintf(out, "  prompt %s (%s)\n", dataset.PromptSHA256, dataset.StructuredOutputMode)
			return nil
		})
		if runErr != nil {
			sink.abort()
		}

		return runErr
	},
}

func init() {
	trainingExportRenderCmd.Flags().StringVar(&renderOutput, "out", "", "directory to create for the datasets")
	trainingExportRenderCmd.Flags().StringVar(&renderStructuredOutput, "structured-output",
		string(aiprovider.StructuredOutputJSONSchema),
		"how the served model will be asked for JSON: JSONSchema, JSONMode or Prompted")
	trainingExportRenderCmd.Flags().BoolVar(&renderDropUnverified, "drop-unverified", false,
		"train only on fields a person confirmed, leaving out the other fields the model predicted")
	_ = trainingExportRenderCmd.MarkFlagRequired("out")
	trainingExportCmd.AddCommand(trainingExportRenderCmd)
}
