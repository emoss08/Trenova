package aiusageresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

// featureTable is what the usage-by-feature table may search, filter and
// sort on.
var featureTable = &base.MemoryTable[services.AIUsageFeatureSlice]{
	Fields: map[string]base.MemoryField[services.AIUsageFeatureSlice]{
		"feature":      func(s services.AIUsageFeatureSlice) any { return featureName(s) },
		"calls":        func(s services.AIUsageFeatureSlice) any { return s.Calls },
		"failed":       func(s services.AIUsageFeatureSlice) any { return s.Failed },
		"tokens":       func(s services.AIUsageFeatureSlice) any { return s.InputTokens + s.OutputTokens },
		"costUsd":      func(s services.AIUsageFeatureSlice) any { return s.CostUSD },
		"latencyP50Ms": func(s services.AIUsageFeatureSlice) any { return s.LatencyP50Ms },
	},
	Search: featureName,
}

func featureName(s services.AIUsageFeatureSlice) string {
	if s.Feature == nil {
		return ""
	}
	return string(*s.Feature)
}

func featureConnection(
	input *gqlmodel.DataTableConnectionInput,
	slices []services.AIUsageFeatureSlice,
) (*gqlmodel.AIUsageFeatureConnection, error) {
	page, err := featureTable.Page(input, slices)
	if err != nil {
		return nil, err
	}

	edges := make([]*gqlmodel.AIUsageFeatureEdge, len(page.Rows))
	for idx := range page.Rows {
		edges[idx] = &gqlmodel.AIUsageFeatureEdge{Node: &page.Rows[idx], Cursor: page.CursorAt(idx)}
	}
	total := page.Total
	return &gqlmodel.AIUsageFeatureConnection{
		Edges:      edges,
		PageInfo:   base.PageInfo(page.HasNext, page.EndCursor),
		TotalCount: &total,
	}, nil
}
