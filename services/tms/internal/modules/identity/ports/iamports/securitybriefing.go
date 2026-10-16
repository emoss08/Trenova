package iamports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type MemberSummaryRequest struct {
	TenantInfo    pagination.TenantInfo
	Now           int64
	DormantBefore int64
}

type MFACoverageRequest struct {
	TenantInfo  pagination.TenantInfo
	Now         int64
	Limit       int
	IncludeList bool
}

type SignInSummaryRequest struct {
	TenantInfo  pagination.TenantInfo
	Since       int64
	IncludeRisk bool
}

type APIKeySummaryRequest struct {
	TenantInfo     pagination.TenantInfo
	Now            int64
	UnusedBefore   int64
	ExpiringBefore int64
}

type SecurityBriefingRepository interface {
	MemberSummary(ctx context.Context, req MemberSummaryRequest) (*iam.SecurityMemberSummary, error)
	MFACoverage(ctx context.Context, req MFACoverageRequest) (*iam.SecurityMFACoverage, error)
	SignInSummary(ctx context.Context, req SignInSummaryRequest) (*iam.SecuritySignInSummary, error)
	APIKeySummary(ctx context.Context, req APIKeySummaryRequest) (*iam.SecurityAPIKeySummary, error)
}

type SecurityBriefingService interface {
	Get(ctx context.Context, actor *permissionports.RequestActor) (*iam.SecurityBriefing, error)
}
