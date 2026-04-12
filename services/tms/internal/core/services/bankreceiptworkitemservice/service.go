package bankreceiptworkitemservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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
	Logger       *zap.Logger
	Repo         repositories.BankReceiptWorkItemRepository
	AuditService serviceports.AuditService
}
type Service struct {
	l            *zap.Logger
	repo         repositories.BankReceiptWorkItemRepository
	auditService serviceports.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.bank-receipt-work-item"),
		repo:         p.Repo,
		auditService: p.AuditService,
	}
}

func (s *Service) ListActive(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*bankreceiptworkitem.WorkItem, error) {
	return s.repo.ListActive(ctx, tenantInfo)
}

func (s *Service) Get(
	ctx context.Context,
	req *serviceports.GetBankReceiptWorkItemRequest,
) (*bankreceiptworkitem.WorkItem, error) {
	return s.repo.GetByID(
		ctx,
		repositories.GetBankReceiptWorkItemByIDRequest{
			ID:         req.WorkItemID,
			TenantInfo: req.TenantInfo,
		},
	)
}

func (s *Service) Assign(
	ctx context.Context,
	req *serviceports.AssignBankReceiptWorkItemRequest,
	actor *serviceports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	userID, err := requireWorkItemUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		repositories.GetBankReceiptWorkItemByIDRequest{
			ID:         req.WorkItemID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if !entity.Status.IsActive() {
		return nil, errortypes.NewBusinessError(
			"Only active bank receipt work items can be assigned",
		)
	}
	original := *entity
	now := timeutils.NowUnix()
	entity.Status = bankreceiptworkitem.StatusAssigned
	entity.AssignedToUserID = req.AssignedToUserID
	entity.AssignedAt = &now
	entity.UpdatedByID = userID
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, &original, userID, "Bank receipt work item assigned")
	return updated, nil
}

func (s *Service) StartReview(
	ctx context.Context,
	req *serviceports.GetBankReceiptWorkItemRequest,
	actor *serviceports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	userID, err := requireWorkItemUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		repositories.GetBankReceiptWorkItemByIDRequest{
			ID:         req.WorkItemID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if !entity.Status.IsActive() {
		return nil, errortypes.NewBusinessError(
			"Only active bank receipt work items can enter review",
		)
	}
	original := *entity
	entity.Status = bankreceiptworkitem.StatusInReview
	entity.UpdatedByID = userID
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(
		updated,
		&original,
		userID,
		"Bank receipt work item moved to review",
	)
	return updated, nil
}

func (s *Service) Resolve(
	ctx context.Context,
	req *serviceports.ResolveBankReceiptWorkItemRequest,
	actor *serviceports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	userID, err := requireWorkItemUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		repositories.GetBankReceiptWorkItemByIDRequest{
			ID:         req.WorkItemID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if !entity.Status.IsActive() {
		return nil, errortypes.NewBusinessError(
			"Only active bank receipt work items can be resolved",
		)
	}
	if strings.TrimSpace(req.ResolutionNote) == "" {
		return nil, errortypes.NewValidationError(
			"resolutionNote",
			errortypes.ErrRequired,
			"Resolution note is required",
		)
	}
	original := *entity
	now := timeutils.NowUnix()
	entity.Status = bankreceiptworkitem.StatusResolved
	entity.ResolutionType = req.ResolutionType
	entity.ResolutionNote = strings.TrimSpace(req.ResolutionNote)
	entity.ResolvedByUserID = userID
	entity.ResolvedAt = &now
	entity.UpdatedByID = userID
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, &original, userID, "Bank receipt work item resolved")
	return updated, nil
}

func (s *Service) Dismiss(
	ctx context.Context,
	req *serviceports.DismissBankReceiptWorkItemRequest,
	actor *serviceports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	userID, err := requireWorkItemUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		repositories.GetBankReceiptWorkItemByIDRequest{
			ID:         req.WorkItemID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if !entity.Status.IsActive() {
		return nil, errortypes.NewBusinessError(
			"Only active bank receipt work items can be dismissed",
		)
	}
	original := *entity
	now := timeutils.NowUnix()
	entity.Status = bankreceiptworkitem.StatusDismissed
	entity.ResolutionType = bankreceiptworkitem.ResolutionMarkedFalsePositive
	entity.ResolutionNote = strings.TrimSpace(req.ResolutionNote)
	entity.ResolvedByUserID = userID
	entity.ResolvedAt = &now
	entity.UpdatedByID = userID
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, &original, userID, "Bank receipt work item dismissed")
	return updated, nil
}

func requireWorkItemUser(actor *serviceports.RequestActor) (pulid.ID, error) {
	if actor == nil || actor.UserID.IsNil() {
		return pulid.Nil, errortypes.NewAuthorizationError(
			"Bank receipt work item action requires an authenticated user",
		)
	}
	return actor.UserID, nil
}

func (s *Service) logAudit(
	current, previous *bankreceiptworkitem.WorkItem,
	userID pulid.ID,
	comment string,
) {
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceBankReceiptWorkItem,
		ResourceID:     current.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error(
			"failed to log bank receipt work item audit action",
			zap.Error(err),
			zap.String("workItemId", current.ID.String()),
		)
	}
}
