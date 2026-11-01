package formulalibraryresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/formulamapping"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
)

func readinessToModel(resp *formulalibraryservice.ReadinessResponse) *gqlmodel.FormulaReadiness {
	checks := make([]*gqlmodel.FormulaReadinessCheck, 0, len(resp.Checks))
	for i := range resp.Checks {
		check := &resp.Checks[i]
		checks = append(
			checks,
			formulamapping.ReadinessCheck(check.Key, check.Label, check.Status, check.Detail),
		)
	}
	return formulamapping.Readiness(&formulamapping.ReadinessParams{
		CanSubmit:   resp.CanSubmit,
		CanApprove:  resp.CanApprove,
		Checks:      checks,
		TestTotal:   resp.TestTotal,
		TestPassed:  resp.TestPassed,
		TestFailing: resp.TestFailing,
		Coverage:    resp.Coverage,
	})
}

func reviewDiffToModel(resp *formulalibraryservice.ReviewDiffResponse) *gqlmodel.FormulaReviewDiff {
	return formulamapping.ReviewDiff(&formulamapping.ReviewDiffParams{
		HasApprovedBase: resp.HasApprovedBase,
		BaseVersion:     resp.BaseVersion,
		CurrentVersion:  resp.CurrentVersion,
		BaseSource:      resp.BaseSource,
		CurrentSource:   resp.CurrentSource,
		BaseImports:     resp.BaseImports,
		CurrentImports:  resp.CurrentImports,
		Changes:         resp.Changes,
		ChangeCount:     resp.ChangeCount,
	})
}

func reviewersToModel(reviewers []formulalibraryservice.Reviewer) []*gqlmodel.FormulaReviewer {
	out := make([]*gqlmodel.FormulaReviewer, 0, len(reviewers))
	for i := range reviewers {
		reviewer := &reviewers[i]
		out = append(out, formulamapping.Reviewer(formulamapping.ReviewerParams{
			ID:            reviewer.ID,
			Name:          reviewer.Name,
			Username:      reviewer.Username,
			ProfilePicURL: reviewer.ProfilePicURL,
		}))
	}
	return out
}
