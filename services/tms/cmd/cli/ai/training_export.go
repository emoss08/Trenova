package ai

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const (
	dateLayout      = "2006-01-02"
	withdrawnPage   = 1000
	startupTimeout  = 30 * time.Second
	shutdownTimeout = 15 * time.Second
	defaultListSize = 20
)

var (
	exportFrom              string
	exportTo                string
	exportMaxPerOrg         int
	exportValidationPercent int
	exportRequestedBy       string
	exportNote              string
	exportListLimit         int
	withdrawnOutput         string
)

var trainingExportCmd = &cobra.Command{
	Use:   "training-export",
	Short: "Anonymized model-training exports from consenting organizations",
	Long: `Anonymized model-training exports.

An export reads the AI corrections of every organization whose AI training
consent is on at the moment it is read, anonymizes each one inside Trenova, and
writes JSONL files and a manifest to object storage under ai-training-exports/.
Consent is read again before an organization's files are kept; an organization
that withdraws while it is being exported is left out.`,
}

var trainingExportStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a training export",
	Long: `Start a training export over the corrections captured in a window.

The window is --from (inclusive) to --to (exclusive), as UTC calendar dates.
--to defaults to now. One export runs at a time.

Examples:
  trenova ai training-export start --from 2026-01-01 --requested-by "Jordan Lee"
  trenova ai training-export start --from 2026-01-01 --to 2026-07-01 --max-per-org 500 --validation-percent 15`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		req, err := startRequest()
		if err != nil {
			return err
		}

		return withOperator(cmd.Context(), func(ctx context.Context, operator services.AITrainingExportOperator) error {
			created, startErr := operator.Start(ctx, req)
			if startErr != nil {
				return startErr
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Started training export %s (workflow %s)\n", created.ID, created.WorkflowID)
			fmt.Fprintf(cmd.OutOrStdout(), "Follow it with: trenova ai training-export status %s\n", created.ID)
			return nil
		})
	},
}

var trainingExportListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent training exports",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return withOperator(cmd.Context(), func(ctx context.Context, operator services.AITrainingExportOperator) error {
			exports, err := operator.List(ctx, exportListLimit)
			if err != nil {
				return err
			}
			if len(exports) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No training exports yet.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tWINDOW\tORGS\tEXAMPLES\tREQUESTED BY\tCREATED")
			for _, e := range exports {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d/%d\t%d\t%s\t%s\n",
					e.ID, e.Status, window(e), e.OrganizationsIncluded, e.OrganizationsConsidered,
					e.ExamplesTotal, e.RequestedBy, formatUnix(e.CreatedAt))
			}
			return w.Flush()
		})
	},
}

var trainingExportStatusCmd = &cobra.Command{
	Use:   "status <export-id>",
	Short: "Show one training export",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := exportID(args[0])
		if err != nil {
			return err
		}

		return withOperator(cmd.Context(), func(ctx context.Context, operator services.AITrainingExportOperator) error {
			entity, getErr := operator.Get(ctx, id)
			if getErr != nil {
				return getErr
			}
			printExport(cmd.OutOrStdout(), entity)
			return nil
		})
	},
}

var trainingExportCancelCmd = &cobra.Command{
	Use:   "cancel <export-id>",
	Short: "Cancel a queued or running training export",
	Long: `Cancel a queued or running training export.

The organization being exported stops at its next page of corrections and its
partial files are deleted; organizations already finished keep their files, and
the manifest still lists them.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := exportID(args[0])
		if err != nil {
			return err
		}

		return withOperator(cmd.Context(), func(ctx context.Context, operator services.AITrainingExportOperator) error {
			canceled, cancelErr := operator.Cancel(ctx, id)
			if cancelErr != nil {
				return cancelErr
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Canceled training export %s\n", canceled.ID)
			return nil
		})
	},
}

var trainingExportWithdrawnCmd = &cobra.Command{
	Use:   "withdrawn <export-id>",
	Short: "List the examples of an export whose organizations have since withdrawn consent",
	Long: `List the example ids in an export whose organization has withdrawn AI
training consent since the export read it, changed it since, or no longer exists.

Remove these examples from any dataset built from the export before training on
it again. Each line is "<example-id>\t<split>".`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := exportID(args[0])
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		var file *os.File
		if withdrawnOutput != "" {
			created, createErr := os.Create(withdrawnOutput)
			if createErr != nil {
				return fmt.Errorf("create %s: %w", withdrawnOutput, createErr)
			}
			file = created
			out = file
		}

		runErr := withOperator(cmd.Context(), func(ctx context.Context, operator services.AITrainingExportOperator) error {
			count, writeErr := writeWithdrawn(ctx, operator, id, out)
			if writeErr != nil {
				return writeErr
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "%d withdrawn example(s)\n", count)
			return nil
		})
		if file != nil {
			if closeErr := file.Close(); closeErr != nil {
				runErr = errors.Join(runErr, fmt.Errorf("close %s: %w", withdrawnOutput, closeErr))
			}
		}

		return runErr
	},
}

func writeWithdrawn(
	ctx context.Context,
	operator services.AITrainingExportOperator,
	id pulid.ID,
	out io.Writer,
) (int, error) {
	buffered := bufio.NewWriter(out)
	count := 0
	var after pulid.ID
	for {
		page, err := operator.WithdrawnExamples(ctx, repositories.ListWithdrawnTrainingExamplesRequest{
			ExportID: id,
			AfterID:  after,
			Limit:    withdrawnPage,
		})
		if err != nil {
			return count, err
		}
		for _, example := range page {
			if _, err = fmt.Fprintf(buffered, "%s\t%s\n", example.ExampleID, example.Split); err != nil {
				return count, fmt.Errorf("write withdrawn examples: %w", err)
			}
			count++
			after = example.ID
		}
		if len(page) < withdrawnPage {
			break
		}
	}

	return count, buffered.Flush()
}

func withOperator(
	parent context.Context,
	run func(ctx context.Context, operator services.AITrainingExportOperator) error,
) error {
	var operator services.AITrainingExportOperator
	return withCommandApp(parent, []any{&operator}, func(ctx context.Context) error {
		return run(ctx, operator)
	})
}

func withCommandApp(parent context.Context, targets []any, run func(ctx context.Context) error) error {
	if parent == nil {
		parent = context.Background()
	}

	app := fx.New(
		bootstrap.TrainingExportCommandOptions(),
		fx.Populate(targets...),
		fx.StartTimeout(startupTimeout),
		fx.StopTimeout(shutdownTimeout),
	)
	if err := app.Err(); err != nil {
		return fmt.Errorf("prepare training export command: %w", err)
	}

	startCtx, cancelStart := context.WithTimeout(parent, startupTimeout)
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		return fmt.Errorf("start training export command: %w", err)
	}

	runErr := run(parent)

	stopCtx, cancelStop := context.WithTimeout(context.WithoutCancel(parent), shutdownTimeout)
	defer cancelStop()

	return errors.Join(runErr, app.Stop(stopCtx))
}

func startRequest() (*services.StartAITrainingExportRequest, error) {
	from, err := parseDate("from", exportFrom)
	if err != nil {
		return nil, err
	}
	var to int64
	if exportTo != "" {
		if to, err = parseDate("to", exportTo); err != nil {
			return nil, err
		}
	}

	requestedBy := strings.TrimSpace(exportRequestedBy)
	if requestedBy == "" {
		if current, userErr := user.Current(); userErr == nil {
			requestedBy = current.Username
		}
	}

	return &services.StartAITrainingExportRequest{
		CapturedFrom:       from,
		CapturedTo:         to,
		MaxPerOrganization: exportMaxPerOrg,
		ValidationPercent:  exportValidationPercent,
		RequestedBy:        requestedBy,
		Note:               exportNote,
	}, nil
}

func parseDate(flag, value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("--%s is required, as YYYY-MM-DD", flag)
	}
	parsed, err := time.ParseInLocation(dateLayout, strings.TrimSpace(value), time.UTC)
	if err != nil {
		return 0, fmt.Errorf("--%s must be a date as YYYY-MM-DD: %w", flag, err)
	}

	return parsed.Unix(), nil
}

func exportID(raw string) (pulid.ID, error) {
	id, err := pulid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("%q is not an export id: %w", raw, err)
	}

	return id, nil
}

func printExport(out io.Writer, e *aitraining.TrainingExport) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Export\t%s\n", e.ID)
	fmt.Fprintf(w, "Status\t%s\n", e.Status)
	fmt.Fprintf(w, "Format\t%s\n", e.Format)
	fmt.Fprintf(w, "Window\t%s\n", window(e))
	fmt.Fprintf(w, "Per organization\tup to %d, %d%% validation\n", e.MaxPerOrganization, e.ValidationPercent)
	fmt.Fprintf(w, "Requested by\t%s\n", e.RequestedBy)
	if e.Note != "" {
		fmt.Fprintf(w, "Note\t%s\n", e.Note)
	}
	fmt.Fprintf(w, "Organizations\t%d included of %d considered\n", e.OrganizationsIncluded, e.OrganizationsConsidered)
	fmt.Fprintf(w, "Examples\t%d (%d train, %d validation)\n", e.ExamplesTotal, e.TrainExamples, e.ValidationExamples)
	for _, reason := range e.DroppedReasons() {
		fmt.Fprintf(w, "Dropped: %s\t%d\n", reason, e.Dropped[reason])
	}
	if e.StartedAt != nil {
		fmt.Fprintf(w, "Started\t%s\n", formatUnix(*e.StartedAt))
	}
	if e.FinishedAt != nil {
		fmt.Fprintf(w, "Finished\t%s\n", formatUnix(*e.FinishedAt))
	}
	if e.ManifestKey != "" {
		fmt.Fprintf(w, "Manifest\t%s (sha256 %s)\n", e.ManifestKey, e.ManifestSHA256)
	}
	if e.FailureMessage != "" {
		fmt.Fprintf(w, "Failure\t%s\n", e.FailureMessage)
	}
	_ = w.Flush()

	if len(e.Parts) == 0 {
		return
	}
	fmt.Fprintln(out)
	parts := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(parts, "PART\tSPLIT\tEXAMPLES\tBYTES\tSHA256")
	for _, part := range e.Parts {
		fmt.Fprintf(parts, "%s\t%s\t%d\t%d\t%s\n", part.Key, part.Split, part.Examples, part.Bytes, part.SHA256)
	}
	_ = parts.Flush()
}

func window(e *aitraining.TrainingExport) string {
	return formatUnix(e.CapturedFrom) + " to " + formatUnix(e.CapturedTo)
}

func formatUnix(value int64) string {
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func init() {
	trainingExportStartCmd.Flags().StringVar(&exportFrom, "from", "", "first capture date to include, YYYY-MM-DD (UTC)")
	trainingExportStartCmd.Flags().StringVar(&exportTo, "to", "", "capture date to stop before, YYYY-MM-DD (UTC); defaults to now")
	trainingExportStartCmd.Flags().IntVar(&exportMaxPerOrg, "max-per-org", aitraining.DefaultMaxPerOrganization,
		"most examples taken from one organization")
	trainingExportStartCmd.Flags().IntVar(&exportValidationPercent, "validation-percent", aitraining.DefaultValidationPercent,
		"share of examples held out for validation, 0 to 50")
	trainingExportStartCmd.Flags().StringVar(&exportRequestedBy, "requested-by", "",
		"operator starting the export; defaults to the current OS user")
	trainingExportStartCmd.Flags().StringVar(&exportNote, "note", "", "why the export is being taken")
	trainingExportListCmd.Flags().IntVar(&exportListLimit, "limit", defaultListSize, "how many exports to list")
	trainingExportWithdrawnCmd.Flags().StringVar(&withdrawnOutput, "output", "", "write to this file instead of stdout")

	trainingExportCmd.AddCommand(
		trainingExportStartCmd,
		trainingExportListCmd,
		trainingExportStatusCmd,
		trainingExportCancelCmd,
		trainingExportWithdrawnCmd,
	)
}
