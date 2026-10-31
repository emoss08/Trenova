package sessionports

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

type SessionLiveness interface {
	Ended(ctx context.Context, sessionIDs []pulid.ID) ([]pulid.ID, error)
}
