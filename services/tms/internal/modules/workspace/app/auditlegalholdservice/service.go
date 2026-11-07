package auditlegalholdservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxActiveHolds = 500

type Params struct {
	fx.In

	Repo    auditports.LegalHoldRepository
	Auditor permissionports.SecurityAuditor
	Logger  *zap.Logger
}

type Service struct {
	repo    auditports.LegalHoldRepository
	auditor permissionports.SecurityAuditor
	l       *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Service {
	return &Service{
		repo:    p.Repo,
		auditor: p.Auditor,
		l:       p.Logger.Named("service.audit-legal-hold"),
	}
}

type PlaceRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      kernelports.AuditActor
	Name       string
	Reason     string
	Resource   permission.Resource
	ResourceID string
}

type ReleaseRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      kernelports.AuditActor
	ID         pulid.ID
	Reason     string
}

func (s *Service) List(
	ctx context.Context,
	req *auditports.ListLegalHoldsRequest,
) (*pagination.CursorListResult[*audit.LegalHold], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(ctx context.Context, req auditports.GetLegalHoldRequest) (*audit.LegalHold, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Place(ctx context.Context, req *PlaceRequest) (*audit.LegalHold, error) {
	hold := &audit.LegalHold{
		ID:             pulid.MustNew("alh_"),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Name:           req.Name,
		Reason:         req.Reason,
		Resource:       req.Resource,
		ResourceID:     req.ResourceID,
		PlacedByID:     req.TenantInfo.UserID,
		PlacedAt:       timeutils.NowUnix(),
	}

	multiErr := errortypes.NewMultiError()
	hold.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	active, err := s.repo.ListActive(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if len(active) >= maxActiveHolds {
		return nil, errortypes.NewBusinessError(
			"An organization can hold at most 500 legal holds at once; release one first",
		)
	}

	created, err := s.repo.Create(ctx, hold)
	if err != nil {
		return nil, err
	}

	s.record(ctx, req.Actor, created, permission.OpLock, audit.ActionLegalHoldPlaced, nil,
		"Legal hold placed: "+created.Name)
	return created, nil
}

func (s *Service) Release(ctx context.Context, req *ReleaseRequest) (*audit.LegalHold, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("reason", errortypes.ErrRequired, "Say why the hold is no longer needed")
		return nil, multiErr
	}

	hold, err := s.repo.GetByID(ctx, auditports.GetLegalHoldRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !hold.Active() {
		return nil, errortypes.NewBusinessError("This legal hold was already released")
	}

	before := *hold
	now := timeutils.NowUnix()
	hold.ReleasedAt = &now
	hold.ReleasedByID = req.TenantInfo.UserID
	hold.ReleaseReason = reason

	released, err := s.repo.Release(ctx, hold)
	if err != nil {
		return nil, err
	}

	s.record(ctx, req.Actor, released, permission.OpUnlock, audit.ActionLegalHoldReleased, &before,
		"Legal hold released: "+released.Name)
	return released, nil
}

func (s *Service) record(
	ctx context.Context,
	actor kernelports.AuditActor,
	hold *audit.LegalHold,
	operation permission.Operation,
	action string,
	before *audit.LegalHold,
	comment string,
) {
	if s.auditor == nil {
		return
	}
	change := &permissionports.SecurityChange{
		Resource:       permission.ResourceAuditLog,
		ResourceID:     hold.ID.String(),
		Operation:      operation,
		Action:         action,
		Actor:          actor,
		OrganizationID: hold.OrganizationID,
		BusinessUnitID: hold.BusinessUnitID,
		After:          hold,
		Comment:        comment,
	}
	if before != nil {
		change.Before = before
	}
	s.auditor.RecordChange(ctx, change)
}
