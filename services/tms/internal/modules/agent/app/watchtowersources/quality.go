package watchtowersources

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweritem"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func QualityRegressionPath(run *agentqualitysuite.SuiteRun) string {
	return pathAgentControl + "?tab=quality&agent=" + run.AgentDefinitionID.String() +
		"&suiteRun=" + run.ID.String()
}

func QualityRegressionSeverity(run *agentqualitysuite.SuiteRun) watchtower.Severity {
	if run.HardFailures > 0 {
		return watchtower.SeverityCritical
	}

	return watchtower.SeverityWarning
}

func QualityRegressionSummary(run *agentqualitysuite.SuiteRun) string {
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
		"Scored "+score+baseline+". "+agentqualitysuite.DescribeChanges(run.FingerprintChanges)+
			" "+run.Comments,
		maxQualitySummary,
	)
}

const maxQualitySummary = 1000

func DescribeQualityRegression(
	run *agentqualitysuite.SuiteRun,
	agentName string,
) watchtowerports.WatchtowerItemInput {
	who := agentName
	if who == "" {
		who = "An agent"
	}
	occurredAt := run.StartedAt
	if run.FinishedAt != nil {
		occurredAt = *run.FinishedAt
	}

	return watchtowerports.WatchtowerItemInput{
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
	runs        agentqualityports.AgentSuiteRunRepository
	definitions agentdefinitionports.AgentDefinitionRepository
}

func NewQualityRegressionSource(
	runs agentqualityports.AgentSuiteRunRepository,
	definitions agentdefinitionports.AgentDefinitionRepository,
) watchtowerports.WatchtowerSource {
	return &QualityRegressionSource{runs: runs, definitions: definitions}
}

func (s *QualityRegressionSource) Kind() watchtower.SourceKind {
	return watchtower.SourceAgentQualityRegression
}

func (s *QualityRegressionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	latest, err := s.runs.Latest(ctx, agentqualityports.LatestAgentSuiteRunsRequest{
		TenantInfo: tenant,
		Statuses: []agentqualitysuite.SuiteRunStatus{
			agentqualitysuite.SuiteRunStatusCompleted,
			agentqualitysuite.SuiteRunStatusBudgetStopped,
		},
		Limit: watchtoweritem.SnapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	regressed := make([]*agentqualitysuite.SuiteRun, 0, len(latest))
	definitionIDs := make([]pulid.ID, 0, len(latest))
	for _, run := range latest {
		if run != nil && run.Regression {
			regressed = append(regressed, run)
			definitionIDs = append(definitionIDs, run.AgentDefinitionID)
		}
	}
	if len(regressed) == 0 {
		return []watchtowerports.WatchtowerItemInput{}, nil
	}

	names := map[pulid.ID]string{}
	if s.definitions != nil {
		definitions, dErr := s.definitions.ListByIDs(
			ctx,
			agentdefinitionports.ListAgentDefinitionsByIDsRequest{
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

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(regressed))
	for _, run := range regressed {
		items = append(items, DescribeQualityRegression(run, names[run.AgentDefinitionID]))
	}

	return items, nil
}
