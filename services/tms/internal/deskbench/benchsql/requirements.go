package benchsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
)

func (d *Database) UnmetRequirements(
	ctx context.Context,
	session *deskbench.Session,
	requires []benchscenario.Requirement,
) (string, error) {
	if len(requires) == 0 {
		return "", nil
	}

	unmet, err := dbtx.Read(
		session.Context(ctx),
		d.db,
		func(ctx context.Context) ([]string, error) {
			db := d.db.Keel(ctx)
			unmet := make([]string, 0, len(requires))
			for idx := range requires {
				statement, args := requires[idx].Probe()
				ready, err := keel.QueryValue[bool](ctx, db, statement, args...)
				if err != nil {
					return nil, fmt.Errorf(
						"check the scenario needs %s: %w",
						requires[idx].Label(),
						err,
					)
				}
				if !ready {
					unmet = append(unmet, requires[idx].Label())
				}
			}

			return unmet, nil
		},
	)
	if err != nil || len(unmet) == 0 {
		return "", err
	}

	return "needs " + strings.Join(unmet, "; and "), nil
}
