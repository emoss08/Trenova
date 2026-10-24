package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                kernelports.Transactor
	ProfileRepo       driverpayports.PayProfileRepository
	AssignmentRepo    driverpayports.WorkerPayAssignmentRepository
	DeductionRepo     driverpayports.RecurringDeductionRepository
	PayCodeRepo       driverpayports.PayCodeRepository
	EarningRepo       driverpayports.RecurringEarningRepository
	AdvanceRepo       driverpayports.PayAdvanceRepository
	EscrowRepo        driverpayports.EscrowAccountRepository
	SettlementControl tenantports.SettlementControlRepository
	AuditService      auditports.AuditService
}

type Service struct {
	l                 *zap.Logger
	db                kernelports.Transactor
	profileRepo       driverpayports.PayProfileRepository
	assignmentRepo    driverpayports.WorkerPayAssignmentRepository
	deductionRepo     driverpayports.RecurringDeductionRepository
	payCodeRepo       driverpayports.PayCodeRepository
	earningRepo       driverpayports.RecurringEarningRepository
	advanceRepo       driverpayports.PayAdvanceRepository
	escrowRepo        driverpayports.EscrowAccountRepository
	settlementControl tenantports.SettlementControlRepository
	auditService      auditports.AuditService
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:                 p.Logger.Named("service.driver-pay"),
		db:                p.DB,
		profileRepo:       p.ProfileRepo,
		assignmentRepo:    p.AssignmentRepo,
		deductionRepo:     p.DeductionRepo,
		payCodeRepo:       p.PayCodeRepo,
		earningRepo:       p.EarningRepo,
		advanceRepo:       p.AdvanceRepo,
		escrowRepo:        p.EscrowRepo,
		settlementControl: p.SettlementControl,
		auditService:      p.AuditService,
	}
}

func requireActor(actor *permissionports.RequestActor, operation string) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError("{0} requires an authenticated user", operation)
	}
	return nil
}

func (s *Service) ListProfiles(
	ctx context.Context,
	req *driverpayports.ListPayProfilesRequest,
) (*pagination.ListResult[*driverpay.PayProfile], error) {
	return s.profileRepo.List(ctx, req)
}

func (s *Service) ListProfilesConnection(
	ctx context.Context,
	req *driverpayports.ListPayProfileConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayProfile], error) {
	return s.profileRepo.ListConnection(ctx, req)
}

func (s *Service) GetProfile(
	ctx context.Context,
	req driverpayports.GetPayProfileByIDRequest,
) (*driverpay.PayProfile, error) {
	return s.profileRepo.GetByID(ctx, req)
}

func (s *Service) ProfileSelectOptions(
	ctx context.Context,
	req *driverpayports.PayProfileSelectOptionsRequest,
) (*pagination.ListResult[*driverpay.PayProfile], error) {
	return s.profileRepo.SelectOptions(ctx, req)
}

func (s *Service) CreateProfile(
	ctx context.Context,
	entity *driverpay.PayProfile,
	actor *permissionports.RequestActor,
) (*driverpay.PayProfile, error) {
	if err := requireActor(actor, "Pay profile creation"); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.profileRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logProfileAudit(created, nil, actor.UserID, permission.OpCreate, "Pay profile created")
	return created, nil
}

func (s *Service) UpdateProfile(
	ctx context.Context,
	entity *driverpay.PayProfile,
	actor *permissionports.RequestActor,
) (*driverpay.PayProfile, error) {
	if err := requireActor(actor, "Pay profile update"); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	previous, err := s.profileRepo.GetByID(ctx, driverpayports.GetPayProfileByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeComponents: true,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.profileRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logProfileAudit(updated, previous, actor.UserID, permission.OpUpdate, "Pay profile updated")
	return updated, nil
}

func (s *Service) ListWorkerAssignments(
	ctx context.Context,
	req driverpayports.ListWorkerPayAssignmentsRequest,
) ([]*driverpay.WorkerPayAssignment, error) {
	return s.assignmentRepo.ListForWorker(ctx, req)
}

func (s *Service) ListProfileAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profileID pulid.ID,
) ([]*driverpay.WorkerPayAssignment, error) {
	return s.assignmentRepo.ListForProfile(ctx, tenantInfo, profileID)
}

func (s *Service) GetEffectiveAssignment(
	ctx context.Context,
	req driverpayports.GetWorkerPayAssignmentRequest,
) (*driverpay.WorkerPayAssignment, error) {
	if req.AsOf == 0 {
		req.AsOf = timeutils.NowUnix()
	}
	return s.assignmentRepo.GetEffectiveForWorker(ctx, req)
}

func (s *Service) AssignProfileToWorker(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
	actor *permissionports.RequestActor,
) (*driverpay.WorkerPayAssignment, error) {
	if err := requireActor(actor, "Pay profile assignment"); err != nil {
		return nil, err
	}
	plan, err := s.PlanAssignment(ctx, entity)
	if err != nil {
		return nil, err
	}

	var created *driverpay.WorkerPayAssignment
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if txErr := s.endOverlappingAssignments(txCtx, entity); txErr != nil {
			return txErr
		}
		entity.CreatedByID = actor.UserID
		var txErr error
		created, txErr = s.assignmentRepo.Create(txCtx, entity)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logAssignmentAudit(created, actor.UserID, permission.OpAssign,
		"Pay profile "+plan.Profile.Name+" assigned to worker")
	return created, nil
}

func validateRateOverrides(
	entity *driverpay.WorkerPayAssignment,
	profile *driverpay.PayProfile,
) error {
	if len(entity.RateOverrides) == 0 {
		return nil
	}
	validComponents := make(map[pulid.ID]struct{}, len(profile.Components))
	for _, component := range profile.Components {
		if component != nil {
			validComponents[component.ID] = struct{}{}
		}
	}
	overrideErr := errortypes.NewMultiError()
	for idx, override := range entity.RateOverrides {
		if _, ok := validComponents[override.ComponentID]; !ok {
			overrideErr.WithIndex("rateOverrides", idx).Add(
				"componentId",
				errortypes.ErrInvalid,
				"Override component does not belong to the selected pay profile",
			)
		}
	}
	if overrideErr.HasErrors() {
		return overrideErr
	}
	return nil
}

func (s *Service) endOverlappingAssignments(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) error {
	overlapping, err := s.assignmentRepo.ListOverlapping(ctx, entity)
	if err != nil {
		return err
	}
	ended, err := PlanEndOverlapping(entity, overlapping)
	if err != nil {
		return err
	}
	for _, closed := range ended {
		if _, err = s.assignmentRepo.Update(ctx, closed); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) EndAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
	endDate int64,
	actor *permissionports.RequestActor,
) (*driverpay.WorkerPayAssignment, error) {
	if err := requireActor(actor, "Pay assignment termination"); err != nil {
		return nil, err
	}
	entity, err := s.assignmentRepo.GetByID(ctx, tenantInfo, assignmentID)
	if err != nil {
		return nil, err
	}
	if err = PlanEndAssignment(entity, endDate); err != nil {
		return nil, err
	}
	updated, err := s.assignmentRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAssignmentAudit(updated, actor.UserID, permission.OpUnassign, "Pay assignment ended")
	return updated, nil
}

func (s *Service) logProfileAudit(
	current, previous *driverpay.PayProfile,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceDriverPayProfile,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []auditports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log pay profile audit action", zap.Error(err))
	}
}

func (s *Service) logAssignmentAudit(
	current *driverpay.WorkerPayAssignment,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceDriverPayProfile,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if err := s.auditService.LogAction(
		params,
		auditservice.WithComment(comment),
	); err != nil {
		s.l.Error("failed to log pay assignment audit action", zap.Error(err))
	}
}
