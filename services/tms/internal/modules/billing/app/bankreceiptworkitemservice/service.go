package bankreceiptworkitemservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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
	Logger       *zap.Logger
	Repo         bankreceiptworkitemports.BankReceiptWorkItemRepository
	AuditService auditports.AuditService
}
type Service struct {
	l            *zap.Logger
	repo         bankreceiptworkitemports.BankReceiptWorkItemRepository
	auditService auditports.AuditService
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
	req *bankreceiptworkitemports.GetBankReceiptWorkItemRequest,
) (*bankreceiptworkitem.WorkItem, error) {
	return s.repo.GetByID(
		ctx,
		bankreceiptworkitemports.GetBankReceiptWorkItemByIDRequest{
			ID:         req.WorkItemID,
			TenantInfo: req.TenantInfo,
		},
	)
}

func (s *Service) PlanAssign(
	ctx context.Context,
	req *bankreceiptworkitemports.AssignBankReceiptWorkItemRequest,
	actor *permissionports.RequestActor,
) (*bankreceiptworkitemports.BankReceiptWorkItemChange, error) {
	userID, err := requireWorkItemUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		bankreceiptworkitemports.GetBankReceiptWorkItemByIDRequest{
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
	after := *entity
	now := timeutils.NowUnix()
	after.Status = bankreceiptworkitem.StatusAssigned
	after.AssignedToUserID = req.AssignedToUserID
	after.AssignedAt = &now
	after.UpdatedByID = userID
	return &bankreceiptworkitemports.BankReceiptWorkItemChange{Before: entity, After: &after}, nil
}

func (s *Service) Assign(
	ctx context.Context,
	req *bankreceiptworkitemports.AssignBankReceiptWorkItemRequest,
	actor *permissionports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	change, err := s.PlanAssign(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return s.save(ctx, change, "Bank receipt work item assigned")
}

func (s *Service) save(
	ctx context.Context,
	change *bankreceiptworkitemports.BankReceiptWorkItemChange,
	comment string,
) (*bankreceiptworkitem.WorkItem, error) {
	updated, err := s.repo.Update(ctx, change.After)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, change.Before, change.After.UpdatedByID, comment)
	return updated, nil
}

func (s *Service) PlanStartReview(
	ctx context.Context,
	req *bankreceiptworkitemports.GetBankReceiptWorkItemRequest,
	actor *permissionports.RequestActor,
) (*bankreceiptworkitemports.BankReceiptWorkItemChange, error) {
	userID, err := requireWorkItemUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		bankreceiptworkitemports.GetBankReceiptWorkItemByIDRequest{
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
	after := *entity
	after.Status = bankreceiptworkitem.StatusInReview
	after.UpdatedByID = userID
	return &bankreceiptworkitemports.BankReceiptWorkItemChange{Before: entity, After: &after}, nil
}

func (s *Service) StartReview(
	ctx context.Context,
	req *bankreceiptworkitemports.GetBankReceiptWorkItemRequest,
	actor *permissionports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	change, err := s.PlanStartReview(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return s.save(ctx, change, "Bank receipt work item moved to review")
}

func (s *Service) Resolve(
	ctx context.Context,
	req *bankreceiptworkitemports.ResolveBankReceiptWorkItemRequest,
	actor *permissionports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	userID, err := requireWorkItemUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		bankreceiptworkitemports.GetBankReceiptWorkItemByIDRequest{
			ID:         req.WorkItemID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	original := *entity
	if err = ResolveWorkItem(entity, req, userID, timeutils.NowUnix()); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, &original, userID, "Bank receipt work item resolved")
	return updated, nil
}

func (s *Service) Dismiss(
	ctx context.Context,
	req *bankreceiptworkitemports.DismissBankReceiptWorkItemRequest,
	actor *permissionports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	userID, err := requireWorkItemUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		bankreceiptworkitemports.GetBankReceiptWorkItemByIDRequest{
			ID:         req.WorkItemID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	original := *entity
	if err = DismissWorkItem(entity, req, userID, timeutils.NowUnix()); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(updated, &original, userID, "Bank receipt work item dismissed")
	return updated, nil
}

func requireWorkItemUser(actor *permissionports.RequestActor) (pulid.ID, error) {
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
	params := &auditports.LogActionParams{
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
	options := []auditports.LogOption{auditservice.WithComment(comment)}
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
