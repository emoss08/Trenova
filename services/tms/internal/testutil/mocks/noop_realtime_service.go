package mocks

import (
	"context"

	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
)

type NoopRealtimeService struct{}

func (s *NoopRealtimeService) CreateTokenRequest(
	_ *servicesport.CreateRealtimeTokenRequest,
) (*servicesport.RealtimeTokenRequest, error) {
	return &servicesport.RealtimeTokenRequest{}, nil
}

func (s *NoopRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	_ *servicesport.PublishResourceInvalidationRequest,
) error {
	return nil
}
