package insightresolver

import (
	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/pkg/authctx"
)

const (
	maxActiveLimit  = 25
	maxBrowseLimit  = 100
	maxBrowseOffset = 10_000
)

func activeInsightsRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.ActiveInsightsInput,
) insightports.ListInsightsRequest {
	req := insightports.ListInsightsRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		UserID:     authCtx.UserID,
		Actor:      actorutil.FromAuthContext(authCtx),
	}
	if input == nil {
		return req
	}

	req.Categories = input.Categories
	if input.Surface != nil {
		req.Surface = *input.Surface
	}
	req.Limit = bounded(input.Limit, maxActiveLimit)

	return req
}

func browseInsightsRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.BrowseInsightsInput,
) insightports.BrowseInsightsRequest {
	return insightports.BrowseInsightsRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		UserID:     authCtx.UserID,
		Actor:      actorutil.FromAuthContext(authCtx),
		Categories: emptyToNil(input.Categories),
		Severities: emptyToNil(input.Severities),
		Statuses:   emptyToNil(input.Statuses),
		Limit:      bounded(input.Limit, maxBrowseLimit),
		Offset:     bounded(input.Offset, maxBrowseOffset),
	}
}

func bounded(value *int, ceiling int) int {
	parsed := gqlscalar.IntValue(value)
	if parsed <= 0 {
		return 0
	}

	return min(parsed, ceiling)
}

func emptyToNil[T insight.Category | insight.Severity | insight.Status](values []T) []T {
	if len(values) == 0 {
		return nil
	}

	return values
}
