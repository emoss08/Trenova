package base

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
)

func seedRecorder(sc *seedhelpers.SeedContext, seedName string) tenantbootstrap.Recorder {
	return func(ctx context.Context, table string, id pulid.ID) error {
		return sc.TrackCreated(ctx, table, id, seedName)
	}
}
