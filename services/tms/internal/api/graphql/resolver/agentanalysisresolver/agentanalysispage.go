package agentanalysisresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func listRunsRequest(
	before *int,
	beforeID *string,
	limit *int,
) (*analysisports.ListRunsRequest, error) {
	req := &analysisports.ListRunsRequest{}
	if before != nil {
		if *before < 0 {
			return nil, errortypes.NewValidationError("before", errortypes.ErrInvalid,
				"before must be a Unix time in seconds")
		}
		req.Before = int64(*before)
	}
	if beforeID != nil && *beforeID != "" {
		id, err := gqlid.ParseIDField("beforeId", *beforeID)
		if err != nil {
			return nil, errortypes.NewValidationError("beforeId", errortypes.ErrInvalid,
				"beforeId must be the id of the last analysis on the page before")
		}
		req.BeforeID = id
	}
	if limit != nil {
		if *limit < 1 || *limit > maxRunsPerPage {
			return nil, errortypes.NewValidationError("limit", errortypes.ErrInvalid,
				"limit must be a whole number from 1 to {0}", maxRunsPerPage)
		}
		req.Limit = *limit
	}

	return req, nil
}
