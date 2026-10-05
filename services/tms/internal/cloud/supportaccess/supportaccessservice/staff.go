package supportaccessservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessrepository"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type staffStore interface {
	UpsertStaffMember(
		ctx context.Context,
		member *supportaccess.StaffMember,
	) (*supportaccess.StaffMember, error)
	DeactivateStaffMember(
		ctx context.Context,
		req supportaccessrepository.DeactivateStaffRequest,
	) (bool, error)
	ListStaffMembers(ctx context.Context, includeInactive bool) ([]*supportaccess.StaffMember, error)
	EndSessionsForStaff(
		ctx context.Context,
		req supportaccessrepository.EndStaffSessionsRequest,
	) ([]*supportaccess.Session, error)
}

type StaffManagerParams struct {
	fx.In

	Repository *supportaccessrepository.Repository
	Users      repositories.UserRepository
	Auditor    services.SecurityAuditor
}

type StaffManager struct {
	repo    staffStore
	users   repositories.UserRepository
	auditor services.SecurityAuditor
	now     func() time.Time
}

func NewStaffManager(p StaffManagerParams) *StaffManager {
	return &StaffManager{
		repo:    p.Repository,
		users:   p.Users,
		auditor: p.Auditor,
		now:     time.Now,
	}
}

func (s *StaffManager) AddStaff(
	ctx context.Context,
	req *AddStaffRequest,
) (*supportaccess.StaffMember, error) {
	usr, err := s.users.FindByEmail(ctx, strings.TrimSpace(req.EmailAddress))
	if err != nil {
		return nil, errortypes.NewNotFoundError("No user has the address {0}", req.EmailAddress)
	}
	if err = usr.ValidateStatus(); err != nil {
		return nil, err
	}

	member := &supportaccess.StaffMember{
		UserID:  usr.ID,
		Role:    req.Role,
		Active:  true,
		AddedBy: strings.TrimSpace(req.AddedBy),
	}

	multiErr := errortypes.NewMultiError()
	member.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.repo.UpsertStaffMember(ctx, member)
	if err != nil {
		return nil, err
	}

	s.recordStaffChange(ctx, &staffChange{
		user:      usr,
		operation: permission.OpCreate,
		after:     map[string]any{"role": req.Role.String(), "active": true},
		comment: "Granted Trenova platform staff membership (" + req.Role.String() + ") by " +
			member.AddedBy,
	})

	saved.UserName = usr.Name
	saved.UserEmail = usr.EmailAddress

	return saved, nil
}

func (s *StaffManager) RemoveStaff(
	ctx context.Context,
	req *RemoveStaffRequest,
) (*RemoveStaffResult, error) {
	usr, err := s.users.FindByEmail(ctx, strings.TrimSpace(req.EmailAddress))
	if err != nil {
		return nil, errortypes.NewNotFoundError("No user has the address {0}", req.EmailAddress)
	}

	by := strings.TrimSpace(req.RemovedBy)
	if by == "" {
		return nil, errortypes.NewValidationError(
			"removedBy",
			errortypes.ErrRequired,
			"Who removed the staff member is required",
		)
	}

	now := s.now().Unix()
	removed, err := s.repo.DeactivateStaffMember(ctx, supportaccessrepository.DeactivateStaffRequest{
		UserID:        usr.ID,
		DeactivatedBy: by,
		Now:           now,
	})
	if err != nil {
		return nil, err
	}

	ended, err := s.repo.EndSessionsForStaff(ctx, supportaccessrepository.EndStaffSessionsRequest{
		StaffUserID: usr.ID,
		Reason:      supportaccess.EndReasonStaffRemoved,
		Now:         now,
	})
	if err != nil {
		return nil, err
	}

	for _, session := range ended {
		recordSessionAudit(ctx, s.auditor, &sessionEvent{
			session:   session,
			operation: sessionEnded,
			comment: "Trenova support session of " + session.StaffName + " ended: " +
				endReasonText(supportaccess.EndReasonStaffRemoved),
			metadata: map[string]any{"endReason": supportaccess.EndReasonStaffRemoved.String()},
		})
	}

	if removed {
		s.recordStaffChange(ctx, &staffChange{
			user:      usr,
			operation: permission.OpDelete,
			before:    map[string]any{"active": true},
			comment:   "Removed Trenova platform staff membership by " + by,
		})
	}

	return &RemoveStaffResult{Removed: removed, EndedSessions: len(ended)}, nil
}

func (s *StaffManager) ListStaff(
	ctx context.Context,
	includeInactive bool,
) ([]*supportaccess.StaffMember, error) {
	return s.repo.ListStaffMembers(ctx, includeInactive)
}

type staffChange struct {
	user      *tenant.User
	operation permission.Operation
	before    any
	after     any
	comment   string
}

func (s *StaffManager) recordStaffChange(ctx context.Context, change *staffChange) {
	if s.auditor == nil {
		return
	}

	s.auditor.RecordChange(
		tenantScope(ctx, change.user.CurrentOrganizationID, change.user.BusinessUnitID, pulid.Nil),
		&services.SecurityChange{
			Resource:       permission.ResourceUser,
			ResourceID:     change.user.ID.String(),
			Operation:      change.operation,
			Actor:          services.SystemAuditActor(),
			OrganizationID: change.user.CurrentOrganizationID,
			BusinessUnitID: change.user.BusinessUnitID,
			Before:         change.before,
			After:          change.after,
			Comment:        change.comment,
			Metadata: map[string]any{
				metadataSupportAccess: "platform_staff_membership",
				"userId":              change.user.ID.String(),
			},
		},
	)
}
