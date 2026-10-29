package volume

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrEnvironment = errors.New(
	"volume data goes only into a database marked development or test: run up -env development first",
)

var Command = keelcli.Command{
	Name:  "volume",
	Usage: "[-shipments n] [-months n] [-seed n] [-org id]",
	Summary: "fill one organization with a large, consistent data set for load and UI testing; " +
		"running it again grows the data set",
	Run: runCommand,
}

func runCommand(ctx context.Context, env *keelcli.Env, args []string) error {
	fs := env.FlagSet("volume")
	shipments := fs.Int("shipments", volumegen.DefaultShipments, "number of shipments to generate")
	months := fs.Int(
		"months",
		volumegen.DefaultHistoryMonths,
		"months of history to spread the data across",
	)
	seedValue := fs.Uint64("seed", 0, "random seed for a reproducible data set (0 picks one)")
	org := fs.String("org", "", "organization to fill (default: the development organization)")
	if err := env.Parse(fs, args); err != nil {
		return err
	}

	orgID := pulid.Nil
	if *org != "" {
		parsed, err := pulid.Parse(*org)
		if err != nil {
			return keelcli.Usagef("volume: -org %q: %v", *org, err)
		}
		orgID = parsed
	}

	db, err := env.DB(ctx)
	if err != nil {
		return err
	}
	mark, err := seed.Environment(ctx, db)
	if err != nil {
		return fmt.Errorf("read the database environment: %w", err)
	}
	if mark != seed.Development && mark != seed.Test {
		return ErrEnvironment
	}

	report, err := Run(ctx, db, volumegen.Options{
		Shipments:      *shipments,
		HistoryMonths:  *months,
		Seed:           *seedValue,
		OrganizationID: orgID,
		Logf: func(format string, args ...any) {
			_, _ = fmt.Fprintf(env.Stdout, format+"\n", args...)
		},
	})
	if report != nil {
		for _, count := range report.Counts {
			_, _ = fmt.Fprintf(env.Stdout, "  %-32s %10d\n", count.Entity, count.Rows)
		}
	}
	if err != nil {
		return fmt.Errorf("volume: %w", err)
	}

	_, _ = fmt.Fprintf(
		env.Stdout,
		"Volume data seeded in %s (seed %d)\n",
		report.Duration.Round(time.Millisecond),
		report.Seed,
	)

	return nil
}
