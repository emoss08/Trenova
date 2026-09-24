package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func qualityWindow(window *int) int {
	if window == nil {
		return services.DefaultAgentQualityWindowDays
	}

	return *window
}

func pageInfoFor(hasNext bool, lastCursor string) *gqlmodel.PageInfo {
	info := &gqlmodel.PageInfo{HasNextPage: hasNext}
	if lastCursor != "" {
		cursor := lastCursor
		info.EndCursor = &cursor
	}

	return info
}

func agentQualityAgentsToModel(
	page *services.AgentQualityAgentPage,
) *gqlmodel.AgentQualityAgentConnection {
	out := &gqlmodel.AgentQualityAgentConnection{
		Edges:      make([]*gqlmodel.AgentQualityAgentEdge, 0, len(page.Edges)),
		TotalCount: page.TotalCount,
	}
	last := ""
	for _, edge := range page.Edges {
		out.Edges = append(out.Edges, &gqlmodel.AgentQualityAgentEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
		last = edge.Cursor
	}
	out.PageInfo = pageInfoFor(page.HasNextPage, last)

	return out
}

func agentWorstRatedToModel(
	page *services.AgentWorstRatedPage,
) *gqlmodel.AgentWorstRatedAnswerConnection {
	out := &gqlmodel.AgentWorstRatedAnswerConnection{
		Edges: make([]*gqlmodel.AgentWorstRatedAnswerEdge, 0, len(page.Edges)),
	}
	last := ""
	for _, edge := range page.Edges {
		out.Edges = append(out.Edges, &gqlmodel.AgentWorstRatedAnswerEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
		last = edge.Cursor
	}
	out.PageInfo = pageInfoFor(page.HasNextPage, last)

	return out
}

func agentSuiteRunsToModel(
	page *services.AgentSuiteRunConnection,
) *gqlmodel.AgentSuiteRunConnection {
	out := &gqlmodel.AgentSuiteRunConnection{
		Edges:      make([]*gqlmodel.AgentSuiteRunEdge, 0, len(page.Edges)),
		TotalCount: page.TotalCount,
	}
	last := ""
	for _, edge := range page.Edges {
		out.Edges = append(out.Edges, &gqlmodel.AgentSuiteRunEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
		last = edge.Cursor
	}
	out.PageInfo = pageInfoFor(page.HasNextPage, last)

	return out
}

func agentSuiteRunCasesToModel(
	page *services.AgentSuiteRunCasePage,
) *gqlmodel.AgentSuiteRunCaseConnection {
	out := &gqlmodel.AgentSuiteRunCaseConnection{
		Edges:      make([]*gqlmodel.AgentSuiteRunCaseEdge, 0, len(page.Edges)),
		TotalCount: page.TotalCount,
	}
	last := ""
	for _, edge := range page.Edges {
		out.Edges = append(out.Edges, &gqlmodel.AgentSuiteRunCaseEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
		last = edge.Cursor
	}
	out.PageInfo = pageInfoFor(page.HasNextPage, last)

	return out
}

func fingerprintChangesToModel(run *agentquality.SuiteRun) []*gqlmodel.AgentFingerprintChange {
	out := make([]*gqlmodel.AgentFingerprintChange, 0, len(run.FingerprintChanges))
	for _, change := range run.FingerprintChanges {
		out = append(out, &gqlmodel.AgentFingerprintChange{
			Field: string(change.Field),
			From:  change.From,
			To:    change.To,
		})
	}

	return out
}

func changeSummary(run *agentquality.SuiteRun) string {
	if run.BaselineRunID == nil || run.BaselineRunID.IsNil() {
		return "This is the agent's first scored run."
	}

	return agentquality.DescribeChanges(run.FingerprintChanges)
}

func parseBudget(field, raw string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Budget must be an amount in dollars",
		)
	}

	return value, nil
}

func qualityControlFromInput(
	tenant pagination.TenantInfo,
	input *gqlmodel.UpdateAgentQualityControlInput,
) (*agentquality.Control, error) {
	nightly, err := parseBudget("nightlyBudgetUsd", input.NightlyBudgetUsd)
	if err != nil {
		return nil, err
	}
	monthly, err := parseBudget("monthlyBudgetUsd", input.MonthlyBudgetUsd)
	if err != nil {
		return nil, err
	}

	return &agentquality.Control{
		OrganizationID:      tenant.OrgID,
		BusinessUnitID:      tenant.BuID,
		Enabled:             input.Enabled,
		RunHourLocal:        input.RunHourLocal,
		Timezone:            derefString(input.Timezone),
		MaxCasesPerAgent:    input.MaxCasesPerAgent,
		NightlyBudgetUSD:    nightly,
		MonthlyBudgetUSD:    monthly,
		JudgeEnabled:        input.JudgeEnabled,
		JudgeSampleRate:     input.JudgeSampleRate,
		RegressionThreshold: input.RegressionThreshold,
		MinCases:            input.MinCases,
		ForceRerunDays:      input.ForceRerunDays,
		Version:             int64(input.Version),
	}, nil
}

func suiteRunAgentName(ctx context.Context, agentID pulid.ID) (string, error) {
	l, ok := loaders.FromContext(ctx)
	if !ok || agentID.IsNil() {
		return "", nil
	}

	definition, err := l.AgentDefinitionByID.Load(ctx, agentID.String())
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return "", nil
		}

		return "", err
	}

	return definition.Name, nil
}
