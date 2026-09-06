// Package workerdqfservice assembles a driver qualification file (49 CFR
// 391.51) and owns the one part of it that had no home: the safety performance
// history investigation of a driver's previous employers (391.23, 382.413).
//
// Everything else in the file already exists somewhere that owns it properly —
// the credential registry holds the licence, medical card, road test, annual
// review and violation certification; the document packet holds the
// application and the paper responses; the testing programme holds the
// pre-employment gates. This service reads those and reports them as one file
// rather than copying them into a second store that could drift.
package workerdqfservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	realtimeVerification = "worker_employment_verification"
	workerResourceType   = "Worker"
	secondsPerDay        = int64(86400)
)

// CredentialSummarizer is the slice of the credential service this needs.
type CredentialSummarizer interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*worker.WorkerCredentialSummary, error)
}

// PacketSummarizer is the slice of the document service this needs: which
// required documents a worker has on file, which the packet rules already
// answer for every resource type.
type PacketSummarizer interface {
	GetPacketSummary(
		ctx context.Context,
		resourceType, resourceID string,
		tenantInfo pagination.TenantInfo,
	) (*documentpacketrule.PacketSummary, error)
}

// DrugAlcoholStander is the slice of the testing programme this needs.
type DrugAlcoholStander interface {
	Standing(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (worker.DrugAlcoholStanding, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.WorkerDQFRepository
	WorkerRepo    repositories.WorkerRepository
	RetentionRepo repositories.DataRetentionRepository
	Credentials   CredentialSummarizer
	Documents     PacketSummarizer
	DrugAlcohol   DrugAlcoholStander
	AuditService  services.AuditService
	Realtime      services.RealtimeService `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	repo          repositories.WorkerDQFRepository
	workerRepo    repositories.WorkerRepository
	retentionRepo repositories.DataRetentionRepository
	credentials   CredentialSummarizer
	documents     PacketSummarizer
	drugAlcohol   DrugAlcoholStander
	auditService  services.AuditService
	realtime      services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.worker-dqf"),
		repo:          p.Repo,
		workerRepo:    p.WorkerRepo,
		retentionRepo: p.RetentionRepo,
		credentials:   p.Credentials,
		documents:     p.Documents,
		drugAlcohol:   p.DrugAlcohol,
		auditService:  p.AuditService,
		realtime:      p.Realtime,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger        *zap.Logger
	Repo          repositories.WorkerDQFRepository
	WorkerRepo    repositories.WorkerRepository
	RetentionRepo repositories.DataRetentionRepository
	Credentials   CredentialSummarizer
	Documents     PacketSummarizer
	DrugAlcohol   DrugAlcoholStander
	AuditService  services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:             logger.Named("service.worker-dqf"),
		repo:          d.Repo,
		workerRepo:    d.WorkerRepo,
		retentionRepo: d.RetentionRepo,
		credentials:   d.Credentials,
		documents:     d.Documents,
		drugAlcohol:   d.DrugAlcohol,
		auditService:  d.AuditService,
	}
}

// File assembles the driver qualification file. The four reads run
// concurrently because they touch unrelated areas and the file shows them
// together.
//
// A collaborator the caller is not allowed to read, or that is not configured,
// is left nil rather than failed: a section with no evidence contributes no
// items, which is a fair report of the truth rather than a pile of invented
// gaps.
func (s *Service) File(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.DQFFile, error) {
	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             workerID,
		TenantInfo:     tenantInfo,
		IncludeProfile: true,
	})
	if err != nil {
		return nil, err
	}

	var (
		credentials   *worker.WorkerCredentialSummary
		documents     *documentpacketrule.PacketSummary
		verifications []*worker.WorkerEmploymentVerification
		standing      worker.DrugAlcoholStanding
	)

	group, groupCtx := errgroup.WithContext(ctx)
	if s.credentials != nil {
		group.Go(func() error {
			var gErr error
			credentials, gErr = s.credentials.Summary(groupCtx, tenantInfo, workerID)
			return gErr
		})
	}
	if s.documents != nil {
		group.Go(func() error {
			var gErr error
			documents, gErr = s.documents.GetPacketSummary(
				groupCtx,
				workerResourceType,
				workerID.String(),
				tenantInfo,
			)
			return gErr
		})
	}
	group.Go(func() error {
		var gErr error
		verifications, gErr = s.repo.ListVerifications(
			groupCtx,
			&repositories.ListEmploymentVerificationsRequest{
				TenantInfo: tenantInfo,
				WorkerID:   workerID,
			},
		)
		return gErr
	})
	if s.drugAlcohol != nil {
		group.Go(func() error {
			var gErr error
			standing, gErr = s.drugAlcohol.Standing(groupCtx, tenantInfo, workerID)
			return gErr
		})
	}

	if err = group.Wait(); err != nil {
		return nil, err
	}

	in := worker.DQFInput{
		WorkerID:      workerID,
		Credentials:   credentials,
		Documents:     documents,
		Verifications: verifications,
		RetentionDays: s.retentionDays(ctx, tenantInfo),
		Now:           timeutils.NowUnix(),
	}
	if s.drugAlcohol != nil {
		in.DrugAlcohol = &standing
	}
	if wrk.Profile != nil {
		in.HireDate = wrk.Profile.HireDate
		in.TerminationDate = wrk.Profile.TerminationDate
	}

	return worker.BuildDQF(in), nil
}

// retentionDays reads the organisation's hold period. A tenant with no
// retention row configured falls back to the regulation's three years rather
// than to zero, which would report every terminated file as purgeable.
func (s *Service) retentionDays(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) int {
	if s.retentionRepo == nil {
		return worker.DefaultDQFRetentionDays
	}
	retention, err := s.retentionRepo.Get(ctx, repositories.GetDataRetentionRequest{
		OrgID: tenantInfo.OrgID,
		BuID:  tenantInfo.BuID,
	})
	if err != nil || retention == nil || retention.DriverQualificationRetentionPeriod <= 0 {
		if err != nil {
			s.l.Warn("failed to read data retention settings", zap.Error(err))
		}
		return worker.DefaultDQFRetentionDays
	}
	return retention.DriverQualificationRetentionPeriod
}

// RetentionCandidates lists the terminated drivers whose file has passed its
// hold period. It flags only: purging a qualification file is a deliberate act,
// not something a sweep should do on somebody's behalf.
func (s *Service) RetentionCandidates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]repositories.DQFRetentionCandidate, error) {
	days := s.retentionDays(ctx, tenantInfo)
	return s.repo.ListRetentionCandidates(ctx, &repositories.ListDQFRetentionCandidatesRequest{
		TenantInfo:       tenantInfo,
		RetentionSeconds: int64(days) * secondsPerDay,
		AsOf:             timeutils.NowUnix(),
		Limit:            limit,
	})
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
	params := &services.LogActionParams{
		Resource:       permission.ResourceQualification,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   jsonutils.MustToJSON(p.current),
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
	}
	opts := []services.LogOption{auditservice.WithComment(p.comment)}
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
		ActorType:      services.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       realtimeVerification,
		Action:         string(operation),
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish employment verification invalidation", zap.Error(err))
	}
}

func verificationTenant(
	entity *worker.WorkerEmploymentVerification,
) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
