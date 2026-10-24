package encryption

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/minio"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/enveloperepository"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/enveloperewrapservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfg *config.Config

	rewrapDryRun      bool
	rewrapOrg         string
	rewrapBatchSize   int
	rewrapSkipObjects bool

	errRewrapIncomplete = errors.New(
		"some sealed values were not moved to the active key; keep the retired keys and run again",
	)
)

func SetConfig(c *config.Config) {
	cfg = c
}

var EncryptionCmd = &cobra.Command{
	Use:   "encryption",
	Short: "Manage the keys that seal stored secrets and documents",
}

var rewrapCmd = &cobra.Command{
	Use:   "rewrap",
	Short: "Move every sealed value and document to the active encryption key",
	Long: `Rewrap every envelope sealed under a retired key so it is sealed under the
active key (security.encryption.key, or the GCP KMS crypto key with
keyManager: gcp-autokey). Only the data key inside each envelope is rewrapped;
the sealed value itself is not decrypted except to move a legacy AI provider
key, agent extension secret or integration secret onto the binding its
provider or tenant now reads it with.

Run it after rotating a key:
  1. Move the old key to security.encryption.retiredKeys and set the new one.
  2. Restart the API and worker, so new values are sealed with the new key.
  3. Run trenova encryption rewrap --dry-run to see how much is left to move.
  4. Run trenova encryption rewrap until it reports nothing left.
  5. Remove the old key from retiredKeys.

Each row is written only if it still holds the value that was read, so the
command is safe to run while Trenova is serving; a row changed in the meantime
is reported as a conflict and is picked up by the next run. Documents and
capture pages are read from and written back to object storage; with bucket
versioning or object lock the previous versions keep the old wrap until they
expire.

Examples:
  trenova encryption rewrap --dry-run
  trenova encryption rewrap
  trenova encryption rewrap --org org_01H... --skip-objects`,
	RunE: runRewrap,
}

func init() {
	rewrapCmd.Flags().BoolVar(&rewrapDryRun, "dry-run", false, "Report what would move without writing")
	rewrapCmd.Flags().StringVar(&rewrapOrg, "org", "", "Rewrap only this organization")
	rewrapCmd.Flags().IntVar(&rewrapBatchSize, "batch-size", 200, "Rows read per query")
	rewrapCmd.Flags().BoolVar(
		&rewrapSkipObjects,
		"skip-objects",
		false,
		"Rewrap database values only and leave documents in object storage",
	)
	EncryptionCmd.AddCommand(rewrapCmd)
}

func runRewrap(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	orgID := pulid.Nil
	if rewrapOrg != "" {
		parsed, err := pulid.Parse(rewrapOrg)
		if err != nil {
			return fmt.Errorf("invalid --org %q: %w", rewrapOrg, err)
		}
		orgID = parsed
	}

	logger, sink, err := config.ProvideLogger(cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = logger.Sync()
		_ = sink.Close()
	}()

	encryption, err := encryptionservice.New(encryptionservice.Params{Config: cfg})
	if err != nil {
		return fmt.Errorf("load the encryption keys: %w", err)
	}

	maint, err := postgres.OpenMaintenanceDB(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("open the database: %w", err)
	}
	defer maint.Close()

	var storage kernelports.Client
	if !rewrapSkipObjects {
		storage, err = minio.New(minio.Params{Config: cfg, Logger: zap.NewNop()})
		if err != nil {
			return fmt.Errorf("connect to object storage: %w", err)
		}
	}

	service := enveloperewrapservice.New(enveloperewrapservice.Deps{
		Repository: enveloperepository.New(
			enveloperepository.Params{DB: postgres.WrapDB(maint)},
		),
		Rewrapper: encryptionservice.NewEnvelopeRewrapper(encryption),
		Storage:   storage,
		Logger:    logger.Named("encryption.rewrap"),
	})

	report, err := service.Run(ctx, envelopeSources(!rewrapSkipObjects), enveloperewrapservice.Options{
		DryRun:         rewrapDryRun,
		BatchSize:      rewrapBatchSize,
		OrganizationID: orgID,
	})
	if err != nil {
		return err
	}

	printReport(report)
	if notBlobs := report.NotBlobs(); notBlobs > 0 {
		logger.Warn("stored objects are not streaming blobs and cannot be read",
			zap.Int("count", notBlobs))
	}
	if report.Failed() > 0 {
		return errRewrapIncomplete
	}
	return nil
}

func printReport(report *enveloperewrapservice.Report) {
	moved := "Rewrapped"
	if report.DryRun {
		moved = "To rewrap"
		color.Yellow("Dry run: nothing was written.")
	}
	for _, source := range report.Sources {
		fmt.Printf(
			"%-40s scanned %6d  %s %6d  current %6d  skipped %6d  not blobs %6d  conflicts %6d  failed %6d\n",
			source.Name,
			source.Scanned,
			moved,
			source.Rewrapped,
			source.Current,
			source.Skipped,
			source.NotBlobs,
			source.Conflicts,
			source.Failed,
		)
		for _, message := range source.Errors {
			color.Red("  %s", message)
		}
	}
	if notBlobs := report.NotBlobs(); notBlobs > 0 {
		color.Red(
			"%d stored objects are not streaming blobs (the retired envelope format) and cannot be read; "+
				"their keys are in the log. Delete them or upload the files again.",
			notBlobs,
		)
	}
	if report.Remaining() == 0 {
		color.Green("Every sealed value is on the active key.")
		return
	}
	color.Yellow("%d values still need the retired keys; run again.", report.Remaining())
}
