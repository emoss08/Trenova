package agenttoolports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

const MetricReadTool = "metric_read"

type MetricGate interface {
	Readable(
		ctx context.Context,
		actor *permissionports.RequestActor,
		access permission.FieldSensitivity,
		metric string,
	) bool
}
