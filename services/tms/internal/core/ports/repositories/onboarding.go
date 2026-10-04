package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetOnboardingRequest struct {
	TenantInfo pagination.TenantInfo
}

type OnboardingRepository interface {
	Get(ctx context.Context, req GetOnboardingRequest) (*onboarding.Onboarding, error)
	Create(ctx context.Context, entity *onboarding.Onboarding) (*onboarding.Onboarding, error)
	Complete(ctx context.Context, entity *onboarding.Onboarding) (*onboarding.Onboarding, error)
}
