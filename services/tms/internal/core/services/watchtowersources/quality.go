package watchtowersources

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func QualityRegressionPath(run *agentquality.SuiteRun) string {
	return pathAgentControl + "?tab=quality&agent=" + run.AgentDefinitionID.String() +
		"&suiteRun=" + run.ID.String()
}

func QualityRegressionSeverity(run *agentquality.SuiteRun) watchtower.Severity {
	if run.HardFailures > 0 {
		return watchtower.SeverityCritical
	}

	return watchtower.SeverityWarning
}

func QualityRegressionSummary(run *agentquality.SuiteRun) string {
	score := "no score"
	if run.QualityScore != nil {
		score = strconv.Itoa(int(*run.QualityScore*100+0.5)) + "%"
	}
	baseline := ""
	if run.BaselineScore != nil {
		baseline = " against a recent median of " +
			strconv.Itoa(int(*run.BaselineScore*100+0.5)) + "%"
	}

	return stringutils.Ellipsize(
		"Scored "+score+baseline+". "+agentquality.DescribeChanges(run.FingerprintChanges)+
			" "+run.Comments,
		maxQualitySummary,
	)
}

const maxQualitySummary = 1000

func DescribeQualityRegression(
	run *agentquality.SuiteRun,
	agentName string,
) services.WatchtowerItemInput {
	who := agentName
	if who == "" {
		who = "An agent"
	}
	occurredAt := run.StartedAt
	if run.FinishedAt != nil {
		occurredAt = *run.FinishedAt
	}

	return services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: run.OrganizationID,
			BuID:  run.BusinessUnitID,
		},
		SourceKind: watchtower.SourceAgentQualityRegression,
		SourceID:   run.ID.String(),
		Severity:   QualityRegressionSeverity(run),
		Title:      who + " answers its evaluation cases worse than before",
		Summary:    QualityRegressionSummary(run),
		Path:       QualityRegressionPath(run),
		OccurredAt: occurredAt,
	}
}

type QualityRegressionSource struct {
	runs        repositories.AgentSuiteRunRepository
	definitions repositories.AgentDefinitionRepository
}

func NewQualityRegressionSource(
	runs repositories.AgentSuiteRunRepository,
	definitions repositories.AgentDefinitionRepository,
) services.WatchtowerSource {
	return &QualityRegressionSource{runs: runs, definitions: definitions}
}

func (s *QualityRegressionSource) Kind() watchtower.SourceKind {
	return watchtower.SourceAgentQualityRegression
}

func (s *QualityRegressionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	latest, err := s.runs.Latest(ctx, repositories.LatestAgentSuiteRunsRequest{
		TenantInfo: tenant,
		Statuses: []agentquality.SuiteRunStatus{
			agentquality.SuiteRunStatusCompleted,
			agentquality.SuiteRunStatusBudgetStopped,
		},
		Limit: snapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	regressed := make([]*agentquality.SuiteRun, 0, len(latest))
	definitionIDs := make([]pulid.ID, 0, len(latest))
	for _, run := range latest {
		if run != nil && run.Regression {
			regressed = append(regressed, run)
			definitionIDs = append(definitionIDs, run.AgentDefinitionID)
		}
	}
	if len(regressed) == 0 {
		return []services.WatchtowerItemInput{}, nil
	}

	names := map[pulid.ID]string{}
	if s.definitions != nil {
		definitions, dErr := s.definitions.ListByIDs(
			ctx,
			repositories.ListAgentDefinitionsByIDsRequest{
				IDs:        unique(definitionIDs),
				TenantInfo: tenant,
			},
		)
		if dErr == nil {
			for _, definition := range definitions {
				if definition != nil {
					names[definition.ID] = definition.Name
				}
			}
		}
	}

	items := make([]services.WatchtowerItemInput, 0, len(regressed))
	for _, run := range regressed {
		items = append(items, DescribeQualityRegression(run, names[run.AgentDefinitionID]))
	}

	return items, nil
}
