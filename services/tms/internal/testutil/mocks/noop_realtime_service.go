package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

type NoopRealtimeService struct{}

func (s *NoopRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	_ *agentports.PublishResourceInvalidationRequest,
) error {
	return nil
}
