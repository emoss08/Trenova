package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AITuneUpAgent struct {
	ID     pulid.ID
	Name   string
	Icon   string
	Accent string
}

type AITuneUpProvider struct {
	ID   pulid.ID
	Name string
	Kind aiprovider.Kind
}

type AITuneUpView struct {
	TuneUp        *aituneup.TuneUp
	Agent         *AITuneUpAgent
	Provider      *AITuneUpProvider
	OtherProvider *AITuneUpProvider
}

type AITuneUpList struct {
	Items      []*AITuneUpView
	ComputedAt *int64
	WindowDays int
}

type AITuneUpDecisionRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	Days       int
}

type AITuneUpService interface {
	List(ctx context.Context, tenantInfo pagination.TenantInfo) (*AITuneUpList, error)
	Get(ctx context.Context, req repositories.GetAITuneUpRequest) (*aituneup.TuneUp, error)
	Compute(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error)
	Apply(ctx context.Context, req *AITuneUpDecisionRequest, actor *RequestActor) (*AITuneUpView, error)
	Dismiss(ctx context.Context, req *AITuneUpDecisionRequest, actor *RequestActor) (*AITuneUpView, error)
	Restore(ctx context.Context, req *AITuneUpDecisionRequest, actor *RequestActor) (*AITuneUpView, error)
}
