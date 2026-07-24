package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListVehiclePositionsRequest struct {
	TenantInfo     pagination.TenantInfo
	TractorIDs     []pulid.ID
	MaxAgeSeconds  int64
	IncludeTractor bool
	IncludeWorker  bool
}

type ListWorkerHOSStatesRequest struct {
	TenantInfo    pagination.TenantInfo
	WorkerIDs     []pulid.ID
	IncludeWorker bool
	Limit         int
}

type GetWorkerHOSStateRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
}

type ListWorkerHOSViolationsRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	Since      int64
	Limit      int
}

type WorkerTelematicsMapping struct {
	WorkerID   pulid.ID
	ExternalID string
	FirstName  string
	LastName   string
}

type TractorTelematicsMapping struct {
	TractorID  pulid.ID
	ExternalID string
	Vin        string
	Code       string
}

type WorkerRulesetAssignment struct {
	WorkerID     pulid.ID
	Cycle        string
	Shift        string
	Restart      string
	Break        string
	Jurisdiction string
}

type UpdateWorkerRulesetsRequest struct {
	TenantInfo  pagination.TenantInfo
	Assignments []WorkerRulesetAssignment
}

type TractorExternalIDAssignment struct {
	TractorID  pulid.ID
	ExternalID string
}

type AssignTractorExternalIDsRequest struct {
	TenantInfo  pagination.TenantInfo
	Assignments []TractorExternalIDAssignment
}

type ListFormSubmissionsRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	WorkerID   pulid.ID
	Since      int64
	Limit      int
}

type ListFormMappingsRequest struct {
	TenantInfo pagination.TenantInfo
	Provider   string
	Enabled    *bool
}

type ListVehicleInspectionsRequest struct {
	TenantInfo pagination.TenantInfo
	TractorID  pulid.ID
	WorkerID   pulid.ID
	Since      int64
	Limit      int
}

type TrailerTelematicsMapping struct {
	TrailerID  pulid.ID
	ExternalID string
	Vin        string
	Code       string
}

type TrailerExternalIDAssignment struct {
	TrailerID  pulid.ID
	ExternalID string
}

type AssignTrailerExternalIDsRequest struct {
	TenantInfo  pagination.TenantInfo
	Assignments []TrailerExternalIDAssignment
}

type TelematicsWebhookConfig struct {
	TenantInfo    pagination.TenantInfo
	WebhookSecret string
}

type TelematicsRepository interface {
	UpsertVehiclePositions(
		ctx context.Context,
		positions []*telematics.VehiclePosition,
	) error
	ListVehiclePositions(
		ctx context.Context,
		req *ListVehiclePositionsRequest,
	) ([]*telematics.VehiclePosition, error)
	UpsertWorkerHOSStates(
		ctx context.Context,
		states []*telematics.WorkerHOSState,
	) error
	ListWorkerHOSStates(
		ctx context.Context,
		req *ListWorkerHOSStatesRequest,
	) ([]*telematics.WorkerHOSState, error)
	GetWorkerHOSState(
		ctx context.Context,
		req GetWorkerHOSStateRequest,
	) (*telematics.WorkerHOSState, error)
	UpsertWorkerHOSViolations(
		ctx context.Context,
		violations []*telematics.WorkerHOSViolation,
	) error
	ListWorkerHOSViolations(
		ctx context.Context,
		req *ListWorkerHOSViolationsRequest,
	) ([]*telematics.WorkerHOSViolation, error)
	UpsertVehicleInspections(
		ctx context.Context,
		inspections []*telematics.VehicleInspection,
	) error
	UpsertFormSubmission(
		ctx context.Context,
		submission *telematics.FormSubmission,
	) (bool, error)
	ListFormSubmissions(
		ctx context.Context,
		req *ListFormSubmissionsRequest,
	) ([]*telematics.FormSubmission, error)
	ListFormMappings(
		ctx context.Context,
		req *ListFormMappingsRequest,
	) ([]*telematics.FormMapping, error)
	GetFormMapping(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*telematics.FormMapping, error)
	SaveFormMapping(
		ctx context.Context,
		mapping *telematics.FormMapping,
		items []*telematics.FormMappingItem,
	) (*telematics.FormMapping, error)
	DeleteFormMapping(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) error
	ListVehicleInspections(
		ctx context.Context,
		req *ListVehicleInspectionsRequest,
	) ([]*telematics.VehicleInspection, error)
	InsertEvent(
		ctx context.Context,
		event *telematics.TelematicsEvent,
	) (bool, error)
	GetFeedState(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		provider string,
		feedType telematics.FeedType,
	) (*telematics.FeedState, error)
	UpsertFeedState(
		ctx context.Context,
		state *telematics.FeedState,
	) error
	ListWorkerMappings(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]WorkerTelematicsMapping, error)
	ListTractorMappings(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]TractorTelematicsMapping, error)
	AssignTractorExternalIDs(
		ctx context.Context,
		req AssignTractorExternalIDsRequest,
	) (int, error)
	ListTrailerMappings(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]TrailerTelematicsMapping, error)
	AssignTrailerExternalIDs(
		ctx context.Context,
		req AssignTrailerExternalIDsRequest,
	) (int, error)
	UpdateWorkerRulesets(
		ctx context.Context,
		req UpdateWorkerRulesetsRequest,
	) (int, error)
	CleanupExpired(
		ctx context.Context,
		eventsOlderThan int64,
		violationsOlderThan int64,
	) (int64, error)
	GetWebhookConfigByToken(
		ctx context.Context,
		typ integration.Type,
		token string,
	) (*TelematicsWebhookConfig, error)
}
