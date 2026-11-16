package supportaccessservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const ActionLogsPulled = "support_access.logs_pulled"

type LogPullRequest struct {
	StaffEmail string
	Kind       supportlog.PullKind
}

func (s *Service) RecordLogPull(ctx context.Context, req *LogPullRequest) error {
	if !s.enabled {
		return &supportaccess.RefusalError{Reason: supportaccess.RefusalNotConfigured}
	}
	if !req.Kind.IsValid() {
		return &supportaccess.RefusalError{Reason: supportaccess.RefusalMalformed, Detail: "unknown pull kind"}
	}

	keys, err := s.keySet(ctx)
	if errors.Is(err, supportaccess.ErrNoKeySet) {
		return s.refusePull(&supportaccess.RefusalError{Reason: supportaccess.RefusalNotConfigured}, req)
	}
	if err != nil {
		return err
	}

	email, err := supportaccess.NormalizeStaffEmail(req.StaffEmail, keys.StaffDomain)
	if err != nil {
		return s.refusePull(
			&supportaccess.RefusalError{Reason: supportaccess.RefusalStaffEmail, Detail: err.Error()},
			req,
		)
	}

	now := s.clock()
	grants, err := s.grants.ListLiveForStaff(ctx, &supportaccessports.ListLiveForStaffRequest{
		StaffEmail: email,
		Now:        now,
		Limit:      supportaccessports.MaxLiveGrantsPerStaff,
	})
	if err != nil {
		return err
	}

	recorded := 0
	for _, grant := range grants {
		if grant.StaffEmail != email || !grant.LiveAt(now) {
			continue
		}
		tenantInfo := pagination.TenantInfo{OrgID: grant.OrganizationID, BuID: grant.BusinessUnitID}
		s.record(dbscope.WithTenant(ctx, tenantInfo.DBTenant()), &change{
			tenantInfo: tenantInfo,
			actor:      principalActor(grant),
			grant:      grant,
			operation:  permission.OpRead,
			action:     ActionLogsPulled,
			after: map[string]any{
				"staffEmail": grant.StaffEmail,
				"kind":       req.Kind.String(),
			},
			comment:  pullComment(req.Kind, grant.StaffEmail),
			metadata: map[string]any{"logPull": req.Kind.String(), "pulledAt": now},
		})
		recorded++
	}
	if recorded == 0 {
		return s.refusePull(&supportaccess.RefusalError{Reason: supportaccess.RefusalNoGrant}, req)
	}
	return nil
}

func pullComment(kind supportlog.PullKind, staffEmail string) string {
	if kind == supportlog.PullBundle {
		return "Trenova support (" + staffEmail + ") pulled a log bundle from this instance"
	}
	return "Trenova support (" + staffEmail + ") pulled recent log lines from this instance"
}

func (s *Service) refusePull(refusal *supportaccess.RefusalError, req *LogPullRequest) error {
	s.l.Warn(
		"refused a Trenova support log pull",
		zap.String("reason", string(refusal.Reason)),
		zap.String("kind", req.Kind.String()),
		zap.String("staffEmail", req.StaffEmail),
	)
	return refusal
}
