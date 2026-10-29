package attribution

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Attributor) UserNames(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) map[pulid.ID]string {
	names := make(map[pulid.ID]string, len(ids))
	if s.Users == nil || len(ids) == 0 {
		return names
	}

	users, err := s.Users.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenant,
		UserIDs:    ids,
	})
	if err != nil {
		s.Logger.Warn("could not read who decided a conversation's plans", zap.Error(err))

		return names
	}
	for _, user := range users {
		if user != nil {
			names[user.ID] = user.Name
		}
	}

	return names
}
