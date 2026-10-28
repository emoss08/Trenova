package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
)

type NoopRealtimeService struct{}

func (s *NoopRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	_ *realtimeports.PublishResourceInvalidationRequest,
) error {
	return nil
}
