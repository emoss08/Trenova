package mocks

import (
	"context"

	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
)

type NoopRealtimeService struct{}

func (s *NoopRealtimeService) CreateToken(
	_ *servicesport.CreateRealtimeTokenRequest,
) (*servicesport.RealtimeToken, error) {
	return &servicesport.RealtimeToken{}, nil
}

func (s *NoopRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	_ *servicesport.PublishResourceInvalidationRequest,
) error {
	return nil
}
