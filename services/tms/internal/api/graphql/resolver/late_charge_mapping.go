package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

func lateChargeRequestFromInput(
	input *gqlmodel.LateChargeAssessmentInput,
	tenantInfo pagination.TenantInfo,
	preview bool,
) (*services.LateChargeAssessmentRequest, error) {
	req := &services.LateChargeAssessmentRequest{
		TenantInfo: tenantInfo,
		Preview:    preview,
	}
	if input == nil {
		return req, nil
	}
	customerIDs, err := pulidsFromStrings(input.CustomerIds, "customerIds")
	if err != nil {
		return nil, err
	}
	req.CustomerIDs = customerIDs
	if input.AsOfDate != nil {
		req.AsOfDate = int64(*input.AsOfDate)
	}

	return req, nil
}
