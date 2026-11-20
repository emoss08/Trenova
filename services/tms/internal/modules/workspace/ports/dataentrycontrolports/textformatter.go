package dataentrycontrolports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
)

type TextFormatter interface {
	Format(ctx context.Context, entity dataentrycontrol.TextFormattable) error
}
