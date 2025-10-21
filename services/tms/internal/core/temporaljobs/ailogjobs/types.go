package ailogjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
)

type InsertAILogPayload struct {
	temporaltype.BasePayload
	Log *ailog.AILog
}

type InsertAILogResult struct {
	ID pulid.ID `json:"id"`
}
