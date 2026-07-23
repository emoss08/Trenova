package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                ports.DBConnection
	ProfileRepo       repositories.PayProfileRepository
	AssignmentRepo    repositories.WorkerPayAssignmentRepository
	DeductionRepo     repositories.RecurringDeductionRepository
	PayCodeRepo       repositories.PayCodeRepository
	EarningRepo       repositories.RecurringEarningRepository
	AdvanceRepo       repositories.PayAdvanceRepository
	EscrowRepo        repositories.EscrowAccountRepository
	SettlementControl repositories.SettlementControlRepository
	AuditService      serviceports.AuditService
}

type Service struct {
	l                 *zap.Logger
	db                ports.DBConnection
	profileRepo       repositories.PayProfileRepository
	assignmentRepo    repositories.WorkerPayAssignmentRepository
	deductionRepo     repositories.RecurringDeductionRepository
	payCodeRepo       repositories.PayCodeRepository
	earningRepo       repositories.RecurringEarningRepository
	advanceRepo       repositories.PayAdvanceRepository
	escrowRepo        repositories.EscrowAccountRepository
	settlementControl repositories.SettlementControlRepository
	auditService      serviceports.AuditService
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

func requireActor(actor *serviceports.RequestActor, operation string) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(operation + " requires an authenticated user")
	}
	return nil
}

func (s *Service) ListProfiles(
	ctx context.Context,
	req *repositories.ListPayProfilesRequest,
) (*pagination.ListResult[*driverpay.PayProfile], error) {
	return s.profileRepo.List(ctx, req)
}

func (s *Service) ListProfilesConnection(
	ctx context.Context,
	req *repositories.ListPayProfileConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayProfile], error) {
	return s.profileRepo.ListConnection(ctx, req)
}

func (s *Service) GetProfile(
	ctx context.Context,
	req repositories.GetPayProfileByIDRequest,
) (*driverpay.PayProfile, error) {
	return s.profileRepo.GetByID(ctx, req)
}

func (s *Service) CreateProfile(
	ctx context.Context,
	entity *driverpay.PayProfile,
	actor *serviceports.RequestActor,
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
	actor *serviceports.RequestActor,
) (*driverpay.PayProfile, error) {
	if err := requireActor(actor, "Pay profile update"); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	previous, err := s.profileRepo.GetByID(ctx, repositories.GetPayProfileByIDRequest{
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
	req repositories.ListWorkerPayAssignmentsRequest,
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
	req repositories.GetWorkerPayAssignmentRequest,
) (*driverpay.WorkerPayAssignment, error) {
	if req.AsOf == 0 {
		req.AsOf = timeutils.NowUnix()
	}
	return s.assignmentRepo.GetEffectiveForWorker(ctx, req)
}

func (s *Service) AssignProfileToWorker(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
	actor *serviceports.RequestActor,
) (*driverpay.WorkerPayAssignment, error) {
	if err := requireActor(actor, "Pay profile assignment"); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	profile, err := s.profileRepo.GetByID(ctx, repositories.GetPayProfileByIDRequest{
		ID: entity.PayProfileID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeComponents: true,
	})
	if err != nil {
		return nil, err
	}

	if err = validateRateOverrides(entity, profile); err != nil {
		return nil, err
	}

	var created *driverpay.WorkerPayAssignment
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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
		"Pay profile "+profile.Name+" assigned to worker")
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
	for _, existing := range overlapping {
		if existing.EffectiveTo != nil && *existing.EffectiveTo <= entity.EffectiveFrom {
			continue
		}
		if existing.EffectiveFrom >= entity.EffectiveFrom {
			return errortypes.NewValidationError(
				"effectiveFrom",
				errortypes.ErrInvalid,
				"Worker already has a pay assignment starting on or after this date",
			)
		}
		endDate := entity.EffectiveFrom
		existing.EffectiveTo = &endDate
		if _, err = s.assignmentRepo.Update(ctx, existing); err != nil {
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
	actor *serviceports.RequestActor,
) (*driverpay.WorkerPayAssignment, error) {
	if err := requireActor(actor, "Pay assignment termination"); err != nil {
		return nil, err
	}
	entity, err := s.assignmentRepo.GetByID(ctx, tenantInfo, assignmentID)
	if err != nil {
		return nil, err
	}
	if endDate <= entity.EffectiveFrom {
		return nil, errortypes.NewValidationError(
			"endDate",
			errortypes.ErrInvalid,
			"End date must be after the assignment's effective from date",
		)
	}
	entity.EffectiveTo = &endDate
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
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceDriverPayProfile,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
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
	params := &serviceports.LogActionParams{
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
