// Package workerdrugalcoholservice owns the controlled-substances and alcohol
// testing programme: the test records themselves, the random selection pools
// and the rounds drawn from them, the FMCSA Clearinghouse queries, and the
// violations whose return-to-duty process decides whether a driver may be
// dispatched at all.
//
// One service covers all four because the answer that matters — may this driver
// work today — is derived from every one of them at once. Splitting them would
// mean four services reading each other to answer a single question.
package workerdrugalcoholservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeTest       = "worker_dot_test"
	realtimeViolation  = "worker_dot_violation"
	realtimeQuery      = "worker_clearinghouse_query"
	realtimePool       = "dot_random_pool"
	realtimeDraw       = "dot_random_draw"
	workerResourceType = "worker"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.WorkerDrugAlcoholRepository
	WorkerRepo   repositories.WorkerRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.WorkerDrugAlcoholRepository
	workerRepo   repositories.WorkerRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.worker-drug-alcohol"),
		repo:         p.Repo,
		workerRepo:   p.WorkerRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger       *zap.Logger
	Repo         repositories.WorkerDrugAlcoholRepository
	WorkerRepo   repositories.WorkerRepository
	AuditService services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:            logger.Named("service.worker-drug-alcohol"),
		repo:         d.Repo,
		workerRepo:   d.WorkerRepo,
		auditService: d.AuditService,
	}
}

type auditParams struct {
	resource   permission.Resource
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
		Resource:       p.resource,
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
		ActorType:      services.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       resource,
		Action:         string(operation),
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish drug and alcohol invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

// Standing reads a worker's whole testing file and reduces it to the answer the
// roster and the dispatch check need. It is the one place the derivation
// happens; everything else caches what it returns.
func (s *Service) Standing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (worker.DrugAlcoholStanding, error) {
	tests, err := s.repo.ListTests(ctx, &repositories.ListWorkerDOTTestsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return worker.DrugAlcoholStanding{}, err
	}

	violation, err := s.repo.GetOpenViolation(ctx, tenantInfo, workerID)
	if err != nil {
		return worker.DrugAlcoholStanding{}, err
	}

	queries, err := s.repo.ListQueries(ctx, &repositories.ListClearinghouseQueriesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return worker.DrugAlcoholStanding{}, err
	}

	return worker.EvaluateDrugAlcoholStanding(worker.DrugAlcoholInput{
		Tests:         tests,
		OpenViolation: violation,
		Queries:       queries,
	}), nil
}

// RefreshRollup recomputes the standing and caches it on the worker's profile
// so the roster and the dispatch check can read it in one column rather than
// replaying every test for every row.
func (s *Service) RefreshRollup(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (worker.DrugAlcoholStanding, error) {
	standing, err := s.Standing(ctx, tenantInfo, workerID)
	if err != nil {
		return worker.DrugAlcoholStanding{}, err
	}

	if err = s.repo.UpdateProfileRollup(ctx, &repositories.UpdateDrugAlcoholRollupRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
		Rollup: repositories.DrugAlcoholRollup{
			Status:                    standing.Status,
			ReturnToDuty:              standing.ReturnToDuty,
			LastClearinghouseQueryAt:  standing.LastClearinghouseQueryAt,
			NextClearinghouseQueryDue: standing.NextClearinghouseQueryDue,
		},
	}); err != nil {
		return worker.DrugAlcoholStanding{}, err
	}

	return standing, nil
}

// refreshRollupQuietly caches the standing after a change. Best-effort on
// purpose: the cache is re-checked by the nightly sweep, and a cache write must
// never fail the record that triggered it.
func (s *Service) refreshRollupQuietly(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) {
	if _, err := s.RefreshRollup(ctx, tenantInfo, workerID); err != nil {
		s.l.Warn("failed to refresh drug and alcohol rollup",
			zap.String("workerId", workerID.String()),
			zap.Error(err))
	}
}

func testTenant(entity *worker.WorkerDOTTest) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func violationTenant(entity *worker.WorkerDOTViolation) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func queryTenant(entity *worker.WorkerClearinghouseQuery) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
