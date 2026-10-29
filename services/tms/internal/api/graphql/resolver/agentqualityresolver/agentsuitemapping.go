package agentqualityresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func qualityWindow(window *int) int {
	if window == nil {
		return agentqualityports.DefaultAgentQualityWindowDays
	}

	return *window
}

func agentQualityAgentsToModel(
	page *agentqualityports.AgentQualityAgentPage,
) *gqlmodel.AgentQualityAgentConnection {
	out := &gqlmodel.AgentQualityAgentConnection{
		Edges:           make([]*gqlmodel.AgentQualityAgentEdge, 0, len(page.Edges)),
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}
	last := ""
	for _, edge := range page.Edges {
		out.Edges = append(out.Edges, &gqlmodel.AgentQualityAgentEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
		last = edge.Cursor
	}
	out.PageInfo = gqlconnection.PageInfoFor(page.HasNextPage, last)

	return out
}

func agentWorstRatedToModel(
	page *agentqualityports.AgentWorstRatedPage,
) *gqlmodel.AgentWorstRatedAnswerConnection {
	out := &gqlmodel.AgentWorstRatedAnswerConnection{
		Edges:           make([]*gqlmodel.AgentWorstRatedAnswerEdge, 0, len(page.Edges)),
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}
	last := ""
	for _, edge := range page.Edges {
		out.Edges = append(out.Edges, &gqlmodel.AgentWorstRatedAnswerEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
		last = edge.Cursor
	}
	out.PageInfo = gqlconnection.PageInfoFor(page.HasNextPage, last)

	return out
}

func agentSuiteRunConnectionToModel(
	result *pagination.CursorListResult[*agentqualitysuite.SuiteRun],
) (*gqlmodel.AgentSuiteRunConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *agentqualitysuite.SuiteRun, cursor string) *gqlmodel.AgentSuiteRunEdge {
			return &gqlmodel.AgentSuiteRunEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentSuiteRunEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentSuiteRunConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func agentSuiteRunCaseConnectionToModel(
	result *pagination.CursorListResult[*agentevaluation.Evaluation],
) (*gqlmodel.AgentSuiteRunCaseConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *agentevaluation.Evaluation, cursor string) *gqlmodel.AgentSuiteRunCaseEdge {
			return &gqlmodel.AgentSuiteRunCaseEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentSuiteRunCaseEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentSuiteRunCaseConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func suiteCaseTableInput(
	input *gqlmodel.DataTableConnectionInput,
) *gqlmodel.DataTableConnectionInput {
	if len(input.Sort) > 0 {
		return input
	}

	sorted := *input
	sorted.Sort = []*gqlmodel.SortFieldInput{
		{Field: "suiteOrdinal", Direction: string(dbtype.SortDirectionAsc)},
	}

	return &sorted
}

func fingerprintChangesToModel(run *agentqualitysuite.SuiteRun) []*gqlmodel.AgentFingerprintChange {
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

func changeSummary(run *agentqualitysuite.SuiteRun) string {
	if run.BaselineRunID == nil || run.BaselineRunID.IsNil() {
		return "This is the agent's first scored run."
	}

	return agentqualitysuite.DescribeChanges(run.FingerprintChanges)
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
) (*agentqualitycontrol.Control, error) {
	nightly, err := parseBudget("nightlyBudgetUsd", input.NightlyBudgetUsd)
	if err != nil {
		return nil, err
	}
	monthly, err := parseBudget("monthlyBudgetUsd", input.MonthlyBudgetUsd)
	if err != nil {
		return nil, err
	}

	return &agentqualitycontrol.Control{
		OrganizationID:      tenant.OrgID,
		BusinessUnitID:      tenant.BuID,
		Enabled:             input.Enabled,
		RunHourLocal:        input.RunHourLocal,
		Timezone:            gqlscalar.DerefString(input.Timezone),
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
	if !loaders.Available(ctx) || agentID.IsNil() {
		return "", nil
	}

	definition, err := loaders.AgentDefinitionByID(ctx, agentID.String())
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return "", nil
		}

		return "", err
	}

	return definition.Name, nil
}
