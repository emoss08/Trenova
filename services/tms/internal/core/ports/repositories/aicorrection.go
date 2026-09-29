package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PurgeAICorrectionsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

type GetAICorrectionRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type GetLatestAICorrectionByDocumentRequest struct {
	TenantInfo pagination.TenantInfo
	Task       aicorrection.Task
	DocumentID pulid.ID
}

type ListAICorrectionConnectionRequest struct {
	Filter  *pagination.QueryOptions
	Cursor  pagination.CursorInfo
	Columns []string
}

type ListAICorrectionsForAccuracyRequest struct {
	TenantInfo pagination.TenantInfo
	Task       aicorrection.Task
	Since      int64
	Limit      int
}

type ListAICorrectionsForTrainingRequest struct {
	TenantInfo      pagination.TenantInfo
	Task            aicorrection.Task
	CapturedFrom    int64
	CapturedTo      int64
	BeforeCapturedAt int64
	BeforeID         pulid.ID
	Limit           int
}

type TotalAICorrectionsByProviderRequest struct {
	TenantInfo pagination.TenantInfo
	Task       aicorrection.Task
	ProviderID pulid.ID
	Since      int64
}

type AICorrectionProviderTotal struct {
	Candidate bool `bun:"candidate"`
	Scored    int  `bun:"scored"`
	Correct   int  `bun:"correct"`
}

type WeeklyAICorrectionTotalsRequest struct {
	TenantInfo pagination.TenantInfo
	Task       aicorrection.Task
	Since      int64
}

type CountTrainableAICorrectionsRequest struct {
	Task               aicorrection.Task
	CapturedFrom       int64
	CapturedTo         int64
	PerOrganizationCap int
}

type WeeklyTrainableAICorrectionTotalsRequest struct {
	Task  aicorrection.Task
	Since int64
}

type AICorrectionRepository interface {
	Upsert(ctx context.Context, entity *aicorrection.Correction) (*aicorrection.Correction, error)
	GetByID(ctx context.Context, req GetAICorrectionRequest) (*aicorrection.Correction, error)
	GetLatestByDocument(
		ctx context.Context,
		req *GetLatestAICorrectionByDocumentRequest,
	) (*aicorrection.Correction, error)
	ListConnection(
		ctx context.Context,
		req *ListAICorrectionConnectionRequest,
	) (*pagination.CursorListResult[*aicorrection.Correction], error)
	ListForAccuracy(
		ctx context.Context,
		req ListAICorrectionsForAccuracyRequest,
	) ([]*aicorrection.Correction, error)
	ListForTraining(
		ctx context.Context,
		req *ListAICorrectionsForTrainingRequest,
	) ([]*aicorrection.Correction, error)
	TotalsByProvider(
		ctx context.Context,
		req *TotalAICorrectionsByProviderRequest,
	) ([]AICorrectionProviderTotal, error)
	WeeklyTotalsByProvider(
		ctx context.Context,
		req *WeeklyAICorrectionTotalsRequest,
	) ([]aicorrection.WeekTotal, error)
	CountTrainable(ctx context.Context, req *CountTrainableAICorrectionsRequest) (int, error)
	WeeklyTrainableTotalsByProvider(
		ctx context.Context,
		req *WeeklyTrainableAICorrectionTotalsRequest,
	) ([]aicorrection.WeekTotal, error)
	PurgeBefore(ctx context.Context, req PurgeAICorrectionsRequest) (int64, error)
}
