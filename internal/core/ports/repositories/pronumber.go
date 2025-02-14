package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ProNumberRepository interface {
	GetNextProNumber(ctx context.Context, orgID pulid.ID) (string, error)
}
