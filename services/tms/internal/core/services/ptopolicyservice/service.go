package ptopolicyservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const realtimeResource = "pto_policy"

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.PTOPolicyRepository
	Ledger       *ptoledgerservice.Service
	WorkerRepo   repositories.WorkerRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.PTOPolicyRepository
	ledger       *ptoledgerservice.Service
	workerRepo   repositories.WorkerRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.pto-policy"),
		repo:         p.Repo,
		ledger:       p.Ledger,
		workerRepo:   p.WorkerRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListPTOPoliciesRequest,
) (*pagination.CursorListResult[*worker.PTOPolicy], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.PTOPolicySelectOptionsRequest,
) (*pagination.ListResult[*worker.PTOPolicy], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetPTOPolicyByIDRequest,
) (*worker.PTOPolicy, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) validate(ctx context.Context, entity *worker.PTOPolicy) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if entity.Code != "" {
		exists, err := s.repo.CodeExists(ctx, &repositories.PTOPolicyCodeExistsRequest{
			TenantInfo: tenantOf(entity),
			Code:       entity.Code,
			ExcludeID:  entity.ID,
		})
		if err != nil {
			return err
		}
		if exists {
			multiErr.Add(
				"code",
				errortypes.ErrDuplicate,
				"A PTO policy with this code already exists",
			)
		}
	}

	if entity.IsDefault && entity.Status != worker.PTOPolicyStatusActive {
		multiErr.Add("isDefault", errortypes.ErrInvalid, "Only an active policy can be the default")
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *worker.PTOPolicy,
	userID pulid.ID,
) (*worker.PTOPolicy, error) {
	log := s.l.With(zap.String("operation", "Create"), zap.String("userID", userID.String()))

	if err := s.validate(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create PTO policy", zap.Error(err))
		return nil, err
	}

	s.audit(created, nil, permission.OpCreate, userID, "PTO policy created", log)
	s.publish(ctx, created, permission.OpCreate, userID)

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *worker.PTOPolicy,
	userID pulid.ID,
) (*worker.PTOPolicy, error) {
	log := s.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	original, err := s.repo.GetByID(ctx, &repositories.GetPTOPolicyByIDRequest{
		ID:           entity.ID,
		TenantInfo:   tenantOf(entity),
		IncludeRules: true,
	})
	if err != nil {
		return nil, err
	}

	if err = s.validate(ctx, entity); err != nil {
		return nil, err
	}

	if original.Status == worker.PTOPolicyStatusActive &&
		entity.Status != worker.PTOPolicyStatusActive {
		if err = s.requireNoOpenAssignments(ctx, entity); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update PTO policy", zap.Error(err))
		return nil, err
	}

	s.audit(updated, original, permission.OpUpdate, userID, "PTO policy updated", log)
	s.publish(ctx, updated, permission.OpUpdate, userID)

	return updated, nil
}

type StatusChangeRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	UserID     pulid.ID
}

func (s *Service) Archive(
	ctx context.Context,
	req *StatusChangeRequest,
) (*worker.PTOPolicy, error) {
	return s.changeStatus(ctx, req, worker.PTOPolicyStatusInactive, permission.OpArchive)
}

func (s *Service) Restore(
	ctx context.Context,
	req *StatusChangeRequest,
) (*worker.PTOPolicy, error) {
	return s.changeStatus(ctx, req, worker.PTOPolicyStatusActive, permission.OpRestore)
}

func (s *Service) changeStatus(
	ctx context.Context,
	req *StatusChangeRequest,
	target worker.PTOPolicyStatus,
	operation permission.Operation,
) (*worker.PTOPolicy, error) {
	log := s.l.With(zap.String("operation", string(operation)), zap.String("id", req.ID.String()))

	original, err := s.repo.GetByID(ctx, &repositories.GetPTOPolicyByIDRequest{
		ID:           req.ID,
		TenantInfo:   req.TenantInfo,
		IncludeRules: true,
	})
	if err != nil {
		return nil, err
	}
	if original.Status == target {
		return original, nil
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"PTO policy was changed by someone else. Reload and try again",
		)
	}

	updated := *original
	updated.Rules = original.Rules
	updated.Status = target
	if target == worker.PTOPolicyStatusInactive {
		if err = s.requireNoOpenAssignments(ctx, &updated); err != nil {
			return nil, err
		}
		updated.IsDefault = false
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to change PTO policy status", zap.Error(err))
		return nil, err
	}

	s.audit(saved, original, operation, req.UserID, fmt.Sprintf("PTO policy %s", target), log)
	s.publish(ctx, saved, operation, req.UserID)

	return saved, nil
}

func (s *Service) CountOpenAssignments(ctx context.Context, entity *worker.PTOPolicy) (int, error) {
	return s.repo.CountOpenAssignments(ctx, &repositories.CountOpenPTOAssignmentsRequest{
		TenantInfo: tenantOf(entity),
		PolicyID:   entity.ID,
	})
}

func (s *Service) CountOpenAssignmentsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	policyIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return s.repo.CountOpenAssignmentsByIDs(ctx, &repositories.CountOpenPTOAssignmentsByIDsRequest{
		TenantInfo: tenantInfo,
		PolicyIDs:  policyIDs,
	})
}

func (s *Service) requireNoOpenAssignments(ctx context.Context, entity *worker.PTOPolicy) error {
	count, err := s.CountOpenAssignments(ctx, entity)
	if err != nil {
		return err
	}
	if count > 0 {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"{0} worker{1} still assigned to this policy. Reassign them before deactivating it",
			count,
			pluralSuffix(count),
		)
	}
	return nil
}

func pluralSuffix(count int) string {
	if count == 1 {
		return " is"
	}
	return "s are"
}

func (s *Service) audit(
	current, previous *worker.PTOPolicy,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
	log *zap.Logger,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourcePTOPolicy,
		ResourceID:     current.GetResourceID(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	opts := []services.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) publish(
	ctx context.Context,
	entity *worker.PTOPolicy,
	operation permission.Operation,
	userID pulid.ID,
) {
	if s.realtime == nil || entity == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    userID,
		ActorType:      services.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       realtimeResource,
		Action:         string(operation),
		RecordID:       entity.ID,
	}); err != nil {
		s.l.Warn("failed to publish PTO policy invalidation", zap.Error(err))
	}
}

func tenantOf(entity *worker.PTOPolicy) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
