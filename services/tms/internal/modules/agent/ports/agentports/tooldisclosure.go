package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
)

type ToolDisclosure struct {
	Attribution aiusageports.AIUsageAttribution
	Disclosed   bool
	Held        int
	Need        string
	Ranked      []string
}

type ToolDisclosureObserver interface {
	Preselected(ctx context.Context, disclosure *ToolDisclosure)
	Found(ctx context.Context, disclosure *ToolDisclosure)
}
