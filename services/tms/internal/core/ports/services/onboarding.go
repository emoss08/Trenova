package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type OnboardingOrganization struct {
	Name         string   `json:"name"`
	Timezone     string   `json:"timezone"`
	AddressLine1 string   `json:"addressLine1"`
	City         string   `json:"city"`
	StateID      pulid.ID `json:"stateId"`
	PostalCode   string   `json:"postalCode"`
	ScacCode     string   `json:"scacCode"`
	DOTNumber    string   `json:"dotNumber"`
}

type OnboardingState struct {
	Required         bool                     `json:"required"`
	Status           onboarding.Status        `json:"status"`
	OperationType    onboarding.OperationType `json:"operationType,omitempty"`
	SampleDataLoaded bool                     `json:"sampleDataLoaded"`
	CompletedAt      *int64                   `json:"completedAt"`
	Organization     *OnboardingOrganization  `json:"organization,omitempty"`
}

type CompleteOnboardingRequest struct {
	TenantInfo     pagination.TenantInfo
	Actor          *RequestActor
	Organization   OnboardingOrganization
	OperationType  onboarding.OperationType
	LoadSampleData bool
}

type OnboardingService interface {
	Get(ctx context.Context, tenantInfo pagination.TenantInfo) (*OnboardingState, error)
	Complete(ctx context.Context, req *CompleteOnboardingRequest) (*OnboardingState, error)
}
