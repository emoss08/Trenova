package aianalyticsservice

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/shared/pulid"
)

type failureDetailInput struct {
	request     *aianalyticsports.ToolFailureDetailRequest
	rows        *aianalyticsports.ToolFailureRows
	agents      map[pulid.ID]string
	providers   providerLookup
	deployments []aianalyticsports.DeploymentRow
}

func failureDetail(in failureDetailInput) *aianalyticsports.ToolFailureDetail {
	out := &aianalyticsports.ToolFailureDetail{
		SeriesKey: in.request.SeriesKey,
		ToolName:  in.request.ToolName,
		Calls:     in.rows.Calls,
		Failed:    in.rows.Failed,
		Causes:    make([]aianalyticsports.CauseCount, 0, len(in.rows.Causes)),
		Recent:    make([]aianalyticsports.FailureExample, 0, len(in.rows.Examples)),
		Providers: []aianalyticsports.ProviderRef{},
	}
	for _, row := range in.rows.Causes {
		out.Causes = append(out.Causes, aianalyticsports.CauseCount{
			Cause: row.Cause,
			Blame: row.Cause.Blame(),
			Calls: row.Calls,
		})
	}
	slices.SortFunc(out.Causes, func(left, right aianalyticsports.CauseCount) int {
		return cmp.Or(cmp.Compare(right.Calls, left.Calls), cmp.Compare(left.Cause, right.Cause))
	})

	for _, example := range in.rows.Examples {
		item := aianalyticsports.FailureExample{
			UnitID:   example.UnitID,
			Source:   example.Source,
			ToolName: example.ToolName,
			Message:  example.Message,
			At:       example.At,
		}
		if name, ok := in.agents[example.AgentDefinitionID]; ok {
			item.Agent = &aianalyticsports.AgentRef{ID: example.AgentDefinitionID, Name: name}
		}
		out.Recent = append(out.Recent, item)
	}

	groupOf := groupKeyOf(in.request.CompareBy)
	rows := slices.Clone(in.deployments)
	slices.SortStableFunc(rows, func(left, right aianalyticsports.DeploymentRow) int {
		return cmp.Compare(right.Runs, left.Runs)
	})
	seen := make(map[string]struct{}, len(rows))
	for idx := range rows {
		row := &rows[idx]
		if groupOf(row.DeploymentKey) != in.request.SeriesKey {
			continue
		}
		if _, ok := seen[row.ProviderID]; ok {
			continue
		}
		seen[row.ProviderID] = struct{}{}
		out.Providers = append(out.Providers, in.providers.ref(row.ProviderID))
	}
	if in.request.CompareBy == aianalytics.CompareByProvider && len(out.Providers) == 0 {
		out.Providers = append(out.Providers, in.providers.ref(in.request.SeriesKey))
	}

	return out
}
