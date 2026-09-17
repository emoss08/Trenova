package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type CarrierIntelTenantCursor struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type ListCarrierIntelTenantsRequest struct {
	After CarrierIntelTenantCursor
	Limit int
}

type CarrierIntelControlRepository interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*carrierintel.CarrierIntelControl, error)
	Update(
		ctx context.Context,
		entity *carrierintel.CarrierIntelControl,
	) (*carrierintel.CarrierIntelControl, error)
	ListConfigured(
		ctx context.Context,
		req *ListCarrierIntelTenantsRequest,
	) ([]*carrierintel.CarrierIntelControl, error)
}

type CarrierIntelSubjectRef struct {
	SubjectType carrierintel.SubjectType
	SubjectID   string
}

type ListCarrierIntelSnapshotHistoryRequest struct {
	TenantInfo  pagination.TenantInfo
	SubjectType carrierintel.SubjectType
	SubjectID   string
	Limit       int
}

type ListSnapshotsForRecomputeRequest struct {
	TenantInfo        pagination.TenantInfo
	BelowVersion      int64
	AfterID           pulid.ID
	Limit             int
	IncludeUpToDateAt bool
}

type MarkSnapshotReviewedRequest struct {
	TenantInfo  pagination.TenantInfo
	SubjectType carrierintel.SubjectType
	SubjectID   string
	UserID      pulid.ID
	Note        string
	ReviewedAt  int64
}

type UpdateCarrierIntelSummaryRequest struct {
	TenantInfo     pagination.TenantInfo
	CarrierID      pulid.ID
	RiskLevel      string
	ReviewRequired bool
	BlockingCount  int
}

type ListCarrierIntelReviewQueueRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type CarrierIntelSnapshotRepository interface {
	GetCurrent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		subject CarrierIntelSubjectRef,
	) (*carrierintel.CarrierIntelSnapshot, error)
	GetCurrentByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) ([]*carrierintel.CarrierIntelSnapshot, error)
	GetCurrentBySubjectIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		subjectType carrierintel.SubjectType,
		subjectIDs []string,
	) ([]*carrierintel.CarrierIntelSnapshot, error)
	GetLatestFullByDOT(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		dotNumber string,
		since int64,
	) (*carrierintel.CarrierIntelSnapshot, error)
	ListHistory(
		ctx context.Context,
		req *ListCarrierIntelSnapshotHistoryRequest,
	) ([]*carrierintel.CarrierIntelSnapshot, error)
	InsertCurrent(
		ctx context.Context,
		entity *carrierintel.CarrierIntelSnapshot,
	) (*carrierintel.CarrierIntelSnapshot, error)
	UpdateEvaluation(ctx context.Context, entity *carrierintel.CarrierIntelSnapshot) error
	TouchConfirmed(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		snapshotIDs []pulid.ID,
		confirmedAt int64,
	) error
	MarkReviewed(ctx context.Context, req *MarkSnapshotReviewedRequest) error
	ListForRecompute(
		ctx context.Context,
		req *ListSnapshotsForRecomputeRequest,
	) ([]*carrierintel.CarrierIntelSnapshot, error)
	ListReviewQueue(
		ctx context.Context,
		req *ListCarrierIntelReviewQueueRequest,
	) ([]*carrierintel.CarrierIntelSnapshot, error)
	PruneHistory(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		keep int,
	) (int, error)
	UpdateCarrierSummary(ctx context.Context, req *UpdateCarrierIntelSummaryRequest) error
}

type CarrierIntelRawPayloadRepository interface {
	Insert(
		ctx context.Context,
		entity *carrierintel.CarrierIntelRawPayload,
	) (*carrierintel.CarrierIntelRawPayload, error)
	GetByID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*carrierintel.CarrierIntelRawPayload, error)
	PurgeExpired(ctx context.Context, before int64, limit int) (int, error)
}

type ListCarrierIntelEventsRequest struct {
	Filter      *pagination.QueryOptions
	Cursor      pagination.CursorInfo
	Statuses    []carrierintel.EventStatus
	Severities  []carrierintel.Severity
	Categories  []carrierintel.Section
	SubjectType carrierintel.SubjectType
	SubjectID   string
	CarrierID   pulid.ID
	OpenOnly    bool
}

type BulkAcknowledgeCarrierIntelEventsRequest struct {
	TenantInfo     pagination.TenantInfo
	EventIDs       []pulid.ID
	UserID         pulid.ID
	AcknowledgedAt int64
}

type ListCarrierIntelDigestEventsRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	Limit      int
}

type CarrierIntelEventCounts struct {
	Open         int
	Acknowledged int
	BySeverity   map[carrierintel.Severity]int
}

type CarrierIntelEventRepository interface {
	InsertIgnoreDuplicates(
		ctx context.Context,
		entities []*carrierintel.CarrierIntelEvent,
	) ([]*carrierintel.CarrierIntelEvent, error)
	ListConnection(
		ctx context.Context,
		req *ListCarrierIntelEventsRequest,
	) (*pagination.CursorListResult[*carrierintel.CarrierIntelEvent], error)
	GetByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*carrierintel.CarrierIntelEvent, error)
	Update(
		ctx context.Context,
		entity *carrierintel.CarrierIntelEvent,
	) (*carrierintel.CarrierIntelEvent, error)
	BulkAcknowledge(
		ctx context.Context,
		req *BulkAcknowledgeCarrierIntelEventsRequest,
	) (int, error)
	CountOpenByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) (map[pulid.ID]int, error)
	CountOpen(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*CarrierIntelEventCounts, error)
	ListForDigest(
		ctx context.Context,
		req *ListCarrierIntelDigestEventsRequest,
	) ([]*carrierintel.CarrierIntelEvent, error)
	MarkNotified(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
		notifiedAt int64,
	) error
}

type ListEnrollmentsBySubjectRequest struct {
	TenantInfo  pagination.TenantInfo
	Provider    integration.Type
	SubjectType carrierintel.SubjectType
	SubjectIDs  []string
}

type ListEnrollmentsNeedingSyncRequest struct {
	TenantInfo pagination.TenantInfo
	Provider   integration.Type
	Limit      int
}

type ListActiveEnrollmentsRequest struct {
	TenantInfo      pagination.TenantInfo
	Provider        integration.Type
	Mode            carrierintel.EnrollmentMode
	ConfirmedBefore *int64
	AfterID         pulid.ID
	Limit           int
}

type ListEnrollmentConnectionRequest struct {
	Filter       *pagination.QueryOptions
	Cursor       pagination.CursorInfo
	Provider     integration.Type
	DesiredState carrierintel.DesiredState
	VendorStates []carrierintel.VendorState
}

type TouchEnrollmentsConfirmedRequest struct {
	TenantInfo  pagination.TenantInfo
	Provider    integration.Type
	Mode        carrierintel.EnrollmentMode
	SubjectIDs  []string
	ConfirmedAt int64
}

type CarrierIntelEnrollmentCounts struct {
	Desired int
	Active  int
	Pending int
	Failed  int
}

type CarrierMonitoringEnrollmentRepository interface {
	ListBySubjects(
		ctx context.Context,
		req *ListEnrollmentsBySubjectRequest,
	) ([]*carrierintel.CarrierMonitoringEnrollment, error)
	GetByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) ([]*carrierintel.CarrierMonitoringEnrollment, error)
	ListDesired(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		provider integration.Type,
	) ([]*carrierintel.CarrierMonitoringEnrollment, error)
	Upsert(
		ctx context.Context,
		entities []*carrierintel.CarrierMonitoringEnrollment,
	) error
	SaveSyncState(
		ctx context.Context,
		entities []*carrierintel.CarrierMonitoringEnrollment,
	) error
	ListNeedingSync(
		ctx context.Context,
		req *ListEnrollmentsNeedingSyncRequest,
	) ([]*carrierintel.CarrierMonitoringEnrollment, error)
	ListActive(
		ctx context.Context,
		req *ListActiveEnrollmentsRequest,
	) ([]*carrierintel.CarrierMonitoringEnrollment, error)
	ListByProviderRefs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		provider integration.Type,
		refs []string,
	) ([]*carrierintel.CarrierMonitoringEnrollment, error)
	ListConnection(
		ctx context.Context,
		req *ListEnrollmentConnectionRequest,
	) (*pagination.CursorListResult[*carrierintel.CarrierMonitoringEnrollment], error)
	TouchConfirmed(ctx context.Context, req *TouchEnrollmentsConfirmedRequest) error
	Counts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		provider integration.Type,
	) (*CarrierIntelEnrollmentCounts, error)
}

type CarrierIntelFeedStateKey struct {
	TenantInfo pagination.TenantInfo
	Provider   integration.Type
	FeedType   carrierintel.FeedType
}

type CarrierIntelFeedStateRepository interface {
	Get(
		ctx context.Context,
		key CarrierIntelFeedStateKey,
	) (*carrierintel.CarrierIntelFeedState, error)
	Upsert(ctx context.Context, entity *carrierintel.CarrierIntelFeedState) error
}

type CarrierIntelUsageSummaryRow struct {
	Provider      integration.Type
	Endpoint      carrierintel.Endpoint
	Calls         int
	BillableUnits int
	EstimatedCost decimal.Decimal
}

type ListCarrierIntelUsageRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	Until      int64
}

type ListCarrierIntelUsageDailyRequest struct {
	TenantInfo pagination.TenantInfo
	FromDay    int
	ToDay      int
}

type CountBillableUsageRequest struct {
	TenantInfo pagination.TenantInfo
	Provider   integration.Type
	Endpoint   carrierintel.Endpoint
	Since      int64
}

type CarrierIntelUsageRepository interface {
	Insert(ctx context.Context, entity *carrierintel.CarrierIntelUsageRecord) (bool, error)
	ExistsDedupeKey(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		provider integration.Type,
		dedupeKey string,
	) (bool, error)
	CostSince(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		since int64,
	) (decimal.Decimal, error)
	CountBillableSince(ctx context.Context, req *CountBillableUsageRequest) (int, error)
	Summary(
		ctx context.Context,
		req *ListCarrierIntelUsageRequest,
	) ([]CarrierIntelUsageSummaryRow, error)
	ListDaily(
		ctx context.Context,
		req *ListCarrierIntelUsageDailyRequest,
	) ([]*carrierintel.CarrierIntelUsageDaily, error)
	RollupDay(ctx context.Context, dayStart int64) (int, error)
	DeleteOlderThan(ctx context.Context, before int64, limit int) (int, error)
}

type CarrierIntelOverrideRepository interface {
	ListActiveByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
		now int64,
	) ([]*carrierintel.CarrierIntelOverride, error)
	ListByCarrier(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierID pulid.ID,
	) ([]*carrierintel.CarrierIntelOverride, error)
	GetByID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*carrierintel.CarrierIntelOverride, error)
	Create(
		ctx context.Context,
		entity *carrierintel.CarrierIntelOverride,
	) (*carrierintel.CarrierIntelOverride, error)
	Update(
		ctx context.Context,
		entity *carrierintel.CarrierIntelOverride,
	) (*carrierintel.CarrierIntelOverride, error)
}

type CarrierEquipmentVerificationRepository interface {
	Create(
		ctx context.Context,
		entity *carrierintel.CarrierEquipmentVerification,
	) (*carrierintel.CarrierEquipmentVerification, error)
	GetByID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*carrierintel.CarrierEquipmentVerification, error)
	ListByAssignmentIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentIDs []pulid.ID,
	) ([]*carrierintel.CarrierEquipmentVerification, error)
	Update(
		ctx context.Context,
		entity *carrierintel.CarrierEquipmentVerification,
	) (*carrierintel.CarrierEquipmentVerification, error)
}

type CarrierIntelSubject struct {
	SubjectType  carrierintel.SubjectType
	SubjectID    string
	CarrierID    pulid.ID
	Name         string
	DOTNumber    string
	DocketNumber string
	LastUsedAt   *int64
	Broker       bool
	Exempt       bool
}

type ListCarrierIntelSubjectsRequest struct {
	TenantInfo         pagination.TenantInfo
	ActiveOnly         bool
	UsedSince          *int64
	IncludeOpenTenders bool
	CarrierIDs         []pulid.ID
	AfterID            pulid.ID
	Limit              int
}

type CarrierIntelSubjectRepository interface {
	ListCarrierSubjects(
		ctx context.Context,
		req *ListCarrierIntelSubjectsRequest,
	) ([]CarrierIntelSubject, error)
	ListBrokerCustomerSubjects(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]CarrierIntelSubject, error)
	GetOrganizationSubject(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*CarrierIntelSubject, error)
	GetCustomerSubject(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
	) (*CarrierIntelSubject, error)
	ListExistingCarrierDOTs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		dotNumbers []string,
	) (map[string]pulid.ID, error)
}
