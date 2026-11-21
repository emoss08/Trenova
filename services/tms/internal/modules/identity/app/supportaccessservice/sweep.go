package supportaccessservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const expirySweepBatch = 200

type ExpirySweepResult struct {
	Recorded int `json:"recorded"`
	Failed   int `json:"failed"`
}

// RecordExpiries writes, once per grant, the security entry saying a grant ended on its own, and
// ends any session it still has. Sessions never outlive their grant (their expiry is the
// grant's, and every request reads the grant), so this records the end rather than enforcing it.
func (s *Service) RecordExpiries(ctx context.Context, now int64) (*ExpirySweepResult, error) {
	result := &ExpirySweepResult{}
	if !s.enabled {
		return result, nil
	}

	grants, err := s.grants.ListUnrecordedExpiries(ctx, now, expirySweepBatch)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, grant := range grants {
		tenantInfo := pagination.TenantInfo{OrgID: grant.OrganizationID, BuID: grant.BusinessUnitID}
		scoped := dbscope.WithTenant(ctx, tenantInfo.DBTenant())

		marked, markErr := s.grants.MarkExpiryRecorded(scoped, grant, now)
		if markErr != nil {
			result.Failed++
			lastErr = markErr
			s.l.Error(
				"cannot record the expiry of a support grant; the other grants are recorded and this one is tried again on the next run",
				zap.String("grantID", grant.ID.String()),
				zap.String("organizationID", grant.OrganizationID.String()),
				zap.Error(markErr),
			)
			continue
		}
		if !marked {
			continue
		}
		s.endSessions(scoped, grant.PrincipalUserID)
		s.record(scoped, &change{
			tenantInfo: tenantInfo,
			actor:      permissionports.SystemAuditActor(),
			grant:      grant,
			operation:  permission.OpDelete,
			action:     ActionExpired,
			after:      grantView(grant, now),
			comment:    "Trenova support access for " + grant.StaffEmail + " ended on its own",
		})
		s.notify(scoped, &supportaccessports.Notice{
			Kind:       supportaccessports.NoticeExpired,
			TenantInfo: tenantInfo,
			Grant:      grant,
		})
		result.Recorded++
	}

	if result.Failed > 0 && result.Failed == len(grants) {
		return result, fmt.Errorf(
			"record the expiry of %d support grants: %w",
			result.Failed,
			lastErr,
		)
	}
	if result.Recorded > 0 {
		s.l.Info(
			"recorded support grants that ended on their own",
			zap.Int("grants", result.Recorded),
		)
	}
	return result, nil
}
