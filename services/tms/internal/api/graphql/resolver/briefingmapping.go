package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

// briefingRequest reads the optional role and day off the input. Both
// absent is the common case: the shared briefing for the organization's
// current day, which is what a home screen asks for.
func briefingRequest(
	tenant pagination.TenantInfo,
	input gqlmodel.TodaysBriefingInput,
) services.GetBriefingRequest {
	req := services.GetBriefingRequest{TenantInfo: tenant}
	if input.RoleKey != nil {
		req.RoleKey = *input.RoleKey
	}
	if input.BriefingDate != nil {
		req.BriefingDate = *input.BriefingDate
	}

	return req
}
