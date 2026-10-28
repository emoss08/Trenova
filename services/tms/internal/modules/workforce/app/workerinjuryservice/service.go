// Package workerinjuryservice owns injury and illness recordkeeping: the cases
// themselves, the OSHA 300 log they form, and the 300A annual summary.
//
// The summary's case totals are derived from the log every time it is read
// rather than stored. 29 CFR 1904.33 requires the log be corrected for five
// years after the year it covers, and a stored total would go stale the first
// time somebody did that.
package workerinjuryservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeInjury  = "worker_injury"
	realtimeSummary = "osha_annual_summary"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         workerports.WorkerInjuryRepository
	WorkerRepo   workerports.WorkerRepository
	AuditService auditports.AuditService
	Realtime     realtimeports.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         workerports.WorkerInjuryRepository
	workerRepo   workerports.WorkerRepository
	auditService auditports.AuditService
	realtime     realtimeports.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.worker-injury"),
		repo:         p.Repo,
		workerRepo:   p.WorkerRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger       *zap.Logger
	Repo         workerports.WorkerInjuryRepository
	WorkerRepo   workerports.WorkerRepository
	AuditService auditports.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:            logger.Named("service.worker-injury"),
		repo:         d.Repo,
		workerRepo:   d.WorkerRepo,
		auditService: d.AuditService,
	}
}

type auditParams struct {
	resourceID string
	operation  permission.Operation
	userID     pulid.ID
	tenant     pagination.TenantInfo
	current    any
	previous   any
	comment    string
}

func (s *Service) audit(p *auditParams) {
	if s.auditService == nil || p.userID.IsNil() {
		return
	}
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceWorkerInjury,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   jsonutils.MustToJSON(p.current),
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
	}
	opts := []auditports.LogOption{auditservice.WithComment(p.comment)}
	if p.previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.previous)
		opts = append(opts, auditservice.WithDiff(p.previous, p.current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource string,
	operation permission.Operation,
	recordID pulid.ID,
	userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    userID,
		ActorType:      permissionports.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       resource,
		Action:         string(operation),
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish injury invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func (s *Service) ListInjuries(
	ctx context.Context,
	req *workerports.ListWorkerInjuriesRequest,
) ([]*workersafety.WorkerInjury, error) {
	return s.repo.ListInjuries(ctx, req)
}

func (s *Service) GetInjury(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*workersafety.WorkerInjury, error) {
	return s.repo.GetInjuryByID(ctx, &workerports.GetWorkerInjuryByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeWorker:   true,
		IncludeDocument: true,
		IncludeEvent:    true,
	})
}

func (s *Service) prepare(entity *workersafety.WorkerInjury) error {
	entity.Description = strings.TrimSpace(entity.Description)
	entity.Location = strings.TrimSpace(entity.Location)
	entity.BodyPart = strings.TrimSpace(entity.BodyPart)
	entity.HarmfulAgent = strings.TrimSpace(entity.HarmfulAgent)
	entity.ClaimNumber = strings.TrimSpace(entity.ClaimNumber)
	entity.ClaimCarrier = strings.TrimSpace(entity.ClaimCarrier)
	entity.Notes = strings.TrimSpace(entity.Notes)

	if entity.Status == "" {
		entity.Status = workersafety.InjuryCaseOpen
	}
	if entity.IllnessType == "" {
		entity.IllnessType = workersafety.IllnessInjury
	}
	if entity.Treatment == "" {
		entity.Treatment = workersafety.TreatmentNone
	}
	if entity.ClaimStatus == "" {
		entity.ClaimStatus = workersafety.ClaimNotFiled
	}
	// Recordability is the employer's judgement, but it should not have to be
	// made from a blank field: the suggestion stands unless somebody chooses
	// otherwise.
	if entity.Classification == "" {
		entity.Classification = workersafety.SuggestClassification(
			entity.Treatment,
			entity.DaysAway,
			entity.DaysRestricted,
		)
	}
	// A claim that was never filed has no dates, however the form was filled
	// in before somebody changed their mind.
	if entity.ClaimStatus == workersafety.ClaimNotFiled {
		entity.ClaimFiledAt = nil
		entity.ClaimClosedAt = nil
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// RecordInjury files a case and gives it the next line on the year's log.
func (s *Service) RecordInjury(
	ctx context.Context,
	entity *workersafety.WorkerInjury,
	userID pulid.ID,
) (*workersafety.WorkerInjury, error) {
	tenantInfo := injuryTenant(entity)

	if err := s.prepareRecord(ctx, entity, userID); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateInjury(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Recorded case " + caseLabel(created),
	})
	s.publish(ctx, tenantInfo, realtimeInjury, permission.OpCreate, created.ID, userID)

	return created, nil
}

// UpdateInjuryRequest carries a correction. 29 CFR 1904.33 requires the log be
// kept accurate for five years after the year it covers, so a case stays
// editable long after it closes.
type UpdateInjuryRequest struct {
	TenantInfo       pagination.TenantInfo
	InjuryID         pulid.ID
	Classification   *workersafety.OSHACaseClassification
	IllnessType      *workersafety.OSHAIllnessType
	Treatment        *workersafety.InjuryTreatment
	Status           *workersafety.InjuryCaseStatus
	OccurredAt       *int64
	ReportedAt       *int64
	ReturnedToWorkAt *int64
	Location         *string
	Description      *string
	BodyPart         *string
	HarmfulAgent     *string
	DaysAway         *int32
	DaysRestricted   *int32
	PrivacyCase      *bool
	ClaimStatus      *workersafety.WorkersCompClaimStatus
	ClaimNumber      *string
	ClaimCarrier     *string
	ClaimFiledAt     *int64
	ClaimClosedAt    *int64
	SafetyEventID    pulid.ID
	DocumentID       pulid.ID
	Notes            *string
	UserID           pulid.ID
}

func (s *Service) UpdateInjury(
	ctx context.Context,
	req *UpdateInjuryRequest,
) (*workersafety.WorkerInjury, error) {
	change, err := s.PlanUpdateInjury(ctx, req)
	if err != nil {
		return nil, err
	}
	previous, entity := change.Before, change.After

	updated, err := s.repo.UpdateInjury(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   previous,
		comment:    "Updated case " + caseLabel(updated),
	})
	s.publish(ctx, req.TenantInfo, realtimeInjury, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

// DeleteInjury removes a case recorded in error. The case number is not reused:
// an auditor reading a log with two case 4s has no way to tell them apart.
func (s *Service) DeleteInjury(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) error {
	entity, err := s.PlanDeleteInjury(ctx, tenantInfo, id)
	if err != nil {
		return err
	}

	if err = s.repo.DeleteInjury(ctx, &workerports.GetWorkerInjuryByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		return err
	}

	s.audit(&auditParams{
		resourceID: id.String(),
		operation:  permission.OpDelete,
		userID:     userID,
		tenant:     tenantInfo,
		current:    entity,
		comment:    "Deleted case " + caseLabel(entity),
	})
	s.publish(ctx, tenantInfo, realtimeInjury, permission.OpDelete, id, userID)

	return nil
}

func ApplyInjuryUpdate(entity *workersafety.WorkerInjury, req *UpdateInjuryRequest) {
	if req.Classification != nil {
		entity.Classification = *req.Classification
	}
	if req.IllnessType != nil {
		entity.IllnessType = *req.IllnessType
	}
	if req.Treatment != nil {
		entity.Treatment = *req.Treatment
	}
	if req.Status != nil {
		entity.Status = *req.Status
	}
	if req.OccurredAt != nil {
		entity.OccurredAt = *req.OccurredAt
	}
	if req.ReportedAt != nil {
		entity.ReportedAt = req.ReportedAt
	}
	if req.ReturnedToWorkAt != nil {
		entity.ReturnedToWorkAt = req.ReturnedToWorkAt
	}
	if req.Location != nil {
		entity.Location = *req.Location
	}
	if req.Description != nil {
		entity.Description = *req.Description
	}
	if req.BodyPart != nil {
		entity.BodyPart = *req.BodyPart
	}
	if req.HarmfulAgent != nil {
		entity.HarmfulAgent = *req.HarmfulAgent
	}
	if req.DaysAway != nil {
		entity.DaysAway = *req.DaysAway
	}
	if req.DaysRestricted != nil {
		entity.DaysRestricted = *req.DaysRestricted
	}
	if req.PrivacyCase != nil {
		entity.PrivacyCase = *req.PrivacyCase
	}
	applyInjuryClaim(entity, req)
}

func applyInjuryClaim(entity *workersafety.WorkerInjury, req *UpdateInjuryRequest) {
	if req.ClaimStatus != nil {
		entity.ClaimStatus = *req.ClaimStatus
	}
	if req.ClaimNumber != nil {
		entity.ClaimNumber = *req.ClaimNumber
	}
	if req.ClaimCarrier != nil {
		entity.ClaimCarrier = *req.ClaimCarrier
	}
	if req.ClaimFiledAt != nil {
		entity.ClaimFiledAt = req.ClaimFiledAt
	}
	if req.ClaimClosedAt != nil {
		entity.ClaimClosedAt = req.ClaimClosedAt
	}
	if !req.SafetyEventID.IsNil() {
		entity.SafetyEventID = req.SafetyEventID
	}
	if !req.DocumentID.IsNil() {
		entity.DocumentID = req.DocumentID
	}
	if req.Notes != nil {
		entity.Notes = *req.Notes
	}
}

func injuryTenant(entity *workersafety.WorkerInjury) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func summaryTenant(entity *workersafety.OSHAAnnualSummary) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func caseLabel(entity *workersafety.WorkerInjury) string {
	return strconv.Itoa(int(entity.CaseYear)) + "-" + strconv.Itoa(int(entity.CaseNumber))
}

func yearOf(unix int64) int16 {
	if unix <= 0 {
		unix = timeutils.NowUnix()
	}
	return int16(timeutils.YearOfUnix(unix))
}
