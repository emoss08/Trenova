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
	for _, grant := range grants {
		tenantInfo := pagination.TenantInfo{OrgID: grant.OrganizationID, BuID: grant.BusinessUnitID}
		scoped := dbscope.WithTenant(ctx, tenantInfo.DBTenant())

		marked, markErr := s.grants.MarkExpiryRecorded(scoped, grant, now)
		if markErr != nil {
			return result, fmt.Errorf(
				"record the expiry of support grant %s: %w",
				grant.ID,
				markErr,
			)
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

	if result.Recorded > 0 {
		s.l.Info(
			"recorded support grants that ended on their own",
			zap.Int("grants", result.Recorded),
		)
	}
	return result, nil
}
