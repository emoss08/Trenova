package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TrainingConsent struct {
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
	Granted        bool     `bun:"granted"`
	GrantedAt      int64    `bun:"granted_at"`
}

func (c TrainingConsent) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: c.OrganizationID, BuID: c.BusinessUnitID}
}

type ListConsentingOrganizationsRequest struct {
	AfterOrganizationID pulid.ID
	AfterBusinessUnitID pulid.ID
	Limit               int
}

type ListAITrainingExportsRequest struct {
	Limit int
}

type ReplaceAITrainingRecordsRequest struct {
	ExportID   pulid.ID
	TenantInfo pagination.TenantInfo
	Records    []*aitraining.TrainingExportRecord
}

type ListAITrainingHistoryRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type ListWithdrawnTrainingExamplesRequest struct {
	ExportID pulid.ID
	AfterID  pulid.ID
	Limit    int
}

type WithdrawnTrainingExample struct {
	ID        pulid.ID `bun:"id"`
	ExampleID string   `bun:"example_id"`
	Split     string   `bun:"split"`
}

type AITrainingExportRepository interface {
	Create(ctx context.Context, entity *aitraining.TrainingExport) (*aitraining.TrainingExport, error)
	GetByID(ctx context.Context, id pulid.ID) (*aitraining.TrainingExport, error)
	List(
		ctx context.Context,
		req ListAITrainingExportsRequest,
	) ([]*aitraining.TrainingExport, error)
	Update(ctx context.Context, entity *aitraining.TrainingExport) (*aitraining.TrainingExport, error)
	ListConsentingOrganizations(
		ctx context.Context,
		req ListConsentingOrganizationsRequest,
	) ([]TrainingConsent, error)
	GetConsent(ctx context.Context, tenantInfo pagination.TenantInfo) (TrainingConsent, error)
	ListOrganizationPeople(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		limit int,
	) ([]string, error)
}

type AITrainingRecordRepository interface {
	ReplaceForOrganization(ctx context.Context, req *ReplaceAITrainingRecordsRequest) error
	DeleteForOrganization(
		ctx context.Context,
		exportID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) error
	ListHistory(
		ctx context.Context,
		req ListAITrainingHistoryRequest,
	) ([]*aitraining.ExportHistoryEntry, error)
	ListWithdrawn(
		ctx context.Context,
		req ListWithdrawnTrainingExamplesRequest,
	) ([]WithdrawnTrainingExample, error)
}
