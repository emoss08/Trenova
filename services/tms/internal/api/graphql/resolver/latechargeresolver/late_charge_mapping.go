package latechargeresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func lateChargeRequestFromInput(
	input *gqlmodel.LateChargeAssessmentInput,
	tenantInfo pagination.TenantInfo,
	preview bool,
) (*latechargeports.LateChargeAssessmentRequest, error) {
	req := &latechargeports.LateChargeAssessmentRequest{
		TenantInfo: tenantInfo,
		Preview:    preview,
	}
	if input == nil {
		return req, nil
	}
	customerIDs, err := base.PulidsFromStrings(input.CustomerIds, "customerIds")
	if err != nil {
		return nil, err
	}
	req.CustomerIDs = customerIDs
	if input.AsOfDate != nil {
		req.AsOfDate = int64(*input.AsOfDate)
	}

	return req, nil
}
