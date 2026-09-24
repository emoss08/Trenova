package agentqualityservice

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	MaxQualityAgents      = 2000
	MaxWorstRatedAnswers  = 1000
	agentListChunk        = 100
	agentDataChunk        = 500
	fieldSatisfaction     = "satisfaction"
	fieldSatisfactionDiff = "satisfactionDelta"
	fieldRatings          = "ratings"
	fieldQualityScore     = "qualityScore"
	fieldOpenRegression   = "openRegression"
	fieldLastRunStatus    = "lastRunStatus"
	fieldLastRunAt        = "lastRunAt"
	fieldQuestion         = "question"
	fieldAgentName        = "agentName"
	fieldAgentDefinition  = "agentDefinitionId"
)

type agentParts struct {
	history bool
	latest  bool
	ratings bool
}

func (p agentParts) requested() bool { return p.history || p.latest || p.ratings }

func (p agentParts) without(done agentParts) agentParts {
	return agentParts{
		history: p.history && !done.history,
		latest:  p.latest && !done.latest,
		ratings: p.ratings && !done.ratings,
	}
}

type agentWindow struct {
	tenant        pagination.TenantInfo
	since         int64
	previousSince int64
}

var qualityAgentTable = memtable.New(memtable.Config[services.AgentQualityAgent]{
	CursorScope: agentsCursorScope,
	Search:      func(row *services.AgentQualityAgent) string { return row.Name },
	Order: func(a, b *services.AgentQualityAgent) int {
		return cmp.Or(
			cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)),
			cmp.Compare(a.AgentDefinitionID.String(), b.AgentDefinitionID.String()),
		)
	},
	Fields: []memtable.Field[services.AgentQualityAgent]{
		{
			Name:       "name",
			Kind:       memtable.KindText,
			Filterable: true,
			Sortable:   true,
			Text:       func(row *services.AgentQualityAgent) string { return row.Name },
		},
		{
			Name:       "enabled",
			Kind:       memtable.KindBoolean,
			Filterable: true,
			Sortable:   true,
			Bool:       func(row *services.AgentQualityAgent) bool { return row.Enabled },
		},
		{
			Name:     fieldSatisfaction,
			Kind:     memtable.KindNumber,
			Sortable: true,
			Number: func(row *services.AgentQualityAgent) (float64, bool) {
				return typeutils.Deref(row.Satisfaction)
			},
		},
		{
			Name:     fieldSatisfactionDiff,
			Kind:     memtable.KindNumber,
			Sortable: true,
			Number: func(row *services.AgentQualityAgent) (float64, bool) {
				return typeutils.Deref(row.SatisfactionDelta)
			},
		},
		{
			Name:     fieldRatings,
			Kind:     memtable.KindNumber,
			Sortable: true,
			Number: func(row *services.AgentQualityAgent) (float64, bool) {
				return float64(row.Ratings), true
			},
		},
		{
			Name:     fieldQualityScore,
			Kind:     memtable.KindNumber,
			Sortable: true,
			Number: func(row *services.AgentQualityAgent) (float64, bool) {
				return typeutils.Deref(row.QualityScore)
			},
		},
		{
			Name:       fieldOpenRegression,
			Kind:       memtable.KindBoolean,
			Filterable: true,
			Sortable:   true,
			Bool:       func(row *services.AgentQualityAgent) bool { return row.OpenRegression },
		},
		{
			Name:       fieldLastRunStatus,
			Kind:       memtable.KindEnum,
			Values:     suiteRunStatusValues(),
			Filterable: true,
			Sortable:   true,
			Text: func(row *services.AgentQualityAgent) string {
				if row.LastSuiteRun == nil {
					return ""
				}

				return string(row.LastSuiteRun.Status)
			},
		},
		{
			Name:     fieldLastRunAt,
			Kind:     memtable.KindNumber,
			Sortable: true,
			Number: func(row *services.AgentQualityAgent) (float64, bool) {
				if row.LastSuiteRun == nil {
					return 0, false
				}

				return float64(row.LastSuiteRun.StartedAt), true
			},
		},
	},
})

func (s *Service) ListAgentTable(
	ctx context.Context,
	req *services.ListAgentQualityAgentTableRequest,
) (*services.AgentQualityAgentPage, error) {
	window, err := windowDays(req.WindowDays)
	if err != nil {
		return nil, err
	}

	needed := agentParts{
		history: qualityAgentTable.References(&req.Table, fieldQualityScore) ||
			qualityAgentTable.References(&req.Table, fieldOpenRegression),
		latest: qualityAgentTable.References(&req.Table, fieldLastRunStatus) ||
			qualityAgentTable.References(&req.Table, fieldLastRunAt),
		ratings: qualityAgentTable.References(&req.Table, fieldSatisfaction) ||
			qualityAgentTable.References(&req.Table, fieldSatisfactionDiff) ||
			qualityAgentTable.References(&req.Table, fieldRatings),
	}
	if needed.ratings && !req.IncludeRatings {
		return nil, errortypes.NewValidationError(
			"sort",
			errortypes.ErrInvalid,
			"Sorting by ratings needs access to agent feedback",
		)
	}

	rows, err := s.allQualityAgents(ctx, req)
	if err != nil {
		return nil, err
	}

	now := s.now()
	since := now - int64(window)*secondsPerDay
	scope := agentWindow{
		tenant:        req.TenantInfo,
		since:         since,
		previousSince: since - int64(window)*secondsPerDay,
	}
	if needed.requested() {
		if err = s.fillAgents(ctx, scope, sliceutils.Pointers(rows), needed); err != nil {
			return nil, err
		}
	}

	page, err := qualityAgentTable.List(rows, &req.Table)
	if err != nil {
		return nil, err
	}

	all := agentParts{history: true, latest: true, ratings: req.IncludeRatings}
	if rest := all.without(needed); rest.requested() && len(page.Items) > 0 {
		if err = s.fillAgents(ctx, scope, page.Items, rest); err != nil {
			return nil, err
		}
	}

	out := &services.AgentQualityAgentPage{
		Edges:       make([]*services.AgentQualityAgentEdge, 0, len(page.Items)),
		HasNextPage: page.HasNextPage,
		TotalCount:  page.TotalCount,
	}
	for idx, node := range page.Items {
		out.Edges = append(out.Edges, &services.AgentQualityAgentEdge{
			Node:   node,
			Cursor: page.Cursors[idx],
		})
	}

	return out, nil
}

func (s *Service) allQualityAgents(
	ctx context.Context,
	req *services.ListAgentQualityAgentTableRequest,
) ([]services.AgentQualityAgent, error) {
	out := make([]services.AgentQualityAgent, 0, agentListChunk)
	for offset := 0; offset < MaxQualityAgents; offset += agentListChunk {
		rows, err := s.suiteRuns.ListAgents(ctx, repositories.ListAgentQualityAgentsRequest{
			TenantInfo: req.TenantInfo,
			Limit:      agentListChunk,
			Offset:     offset,
		})
		if err != nil {
			return nil, err
		}

		for _, row := range rows.Items {
			out = append(out, services.AgentQualityAgent{
				AgentDefinitionID: row.ID,
				Name:              row.Name,
				Enabled:           row.Enabled,
				RatingsVisible:    req.IncludeRatings,
				QualityPoints:     []*services.AgentQualityPoint{},
			})
		}
		if !rows.HasNextPage {
			break
		}
	}

	return out, nil
}

func (s *Service) fillAgents(
	ctx context.Context,
	scope agentWindow,
	nodes []*services.AgentQualityAgent,
	parts agentParts,
) error {
	for chunk := range slices.Chunk(nodes, agentDataChunk) {
		ids := make([]pulid.ID, 0, len(chunk))
		for _, node := range chunk {
			ids = append(ids, node.AgentDefinitionID)
		}

		data, err := s.agentPageData(ctx, agentPageRequest{
			tenant:         scope.tenant,
			ids:            ids,
			since:          scope.since,
			previousSince:  scope.previousSince,
			includeRatings: parts.ratings,
			skipHistory:    !parts.history,
			skipLatest:     !parts.latest,
		})
		if err != nil {
			return err
		}

		for _, node := range chunk {
			applyAgentData(node, data, parts)
		}
	}

	return nil
}

func applyAgentData(node *services.AgentQualityAgent, data *agentPageData, parts agentParts) {
	id := node.AgentDefinitionID
	if parts.history {
		node.QualityPoints = qualityPoints(data.history[id])
		node.QualityScore = nil
		node.OpenRegression = false
		if count := len(node.QualityPoints); count > 0 {
			score := node.QualityPoints[count-1].QualityScore
			node.QualityScore = &score
			node.OpenRegression = node.QualityPoints[count-1].Regression
		}
	}
	if parts.latest {
		node.LastSuiteRun = data.latest[id]
	}
	if parts.ratings {
		current := data.current[id]
		previous := data.previous[id]
		node.Ratings = current.Positive + current.Negative
		node.Satisfaction = satisfaction(current.Positive, current.Negative)
		node.SatisfactionDelta = nil
		if before := satisfaction(previous.Positive, previous.Negative); before != nil &&
			node.Satisfaction != nil {
			delta := *node.Satisfaction - *before
			node.SatisfactionDelta = &delta
		}
	}
}

type worstRatedRow struct {
	answer *services.AgentWorstRatedAnswer
	rank   int
}

var worstRatedTable = memtable.New(memtable.Config[worstRatedRow]{
	CursorScope: worstRatedCursorScope,
	Search: func(row *worstRatedRow) string {
		return questionOf(row.answer) + "\x00" + commentOf(row.answer) + "\x00" + row.answer.AgentName
	},
	Order: func(a, b *worstRatedRow) int { return cmp.Compare(a.rank, b.rank) },
	Fields: []memtable.Field[worstRatedRow]{
		{
			Name:       fieldQuestion,
			Kind:       memtable.KindText,
			Filterable: true,
			Sortable:   true,
			Text:       func(row *worstRatedRow) string { return questionOf(row.answer) },
		},
		{
			Name:       fieldAgentName,
			Kind:       memtable.KindText,
			Filterable: true,
			Sortable:   true,
			Text:       func(row *worstRatedRow) string { return row.answer.AgentName },
		},
		{
			Name:       fieldAgentDefinition,
			Kind:       memtable.KindEnum,
			Filterable: true,
			Text: func(row *worstRatedRow) string {
				if row.answer.AgentDefinitionID == nil {
					return ""
				}

				return row.answer.AgentDefinitionID.String()
			},
		},
		{
			Name: "targetType",
			Kind: memtable.KindEnum,
			Values: []string{
				aifeedback.TargetAssistantMessage.String(),
				aifeedback.TargetDelegatedAnswer.String(),
				aifeedback.TargetBriefing.String(),
				aifeedback.TargetBriefingSection.String(),
				aifeedback.TargetInsight.String(),
				aifeedback.TargetWatchtowerItem.String(),
			},
			Filterable: true,
			Sortable:   true,
			Text:       func(row *worstRatedRow) string { return row.answer.TargetType.String() },
		},
		{
			Name:       "negative",
			Kind:       memtable.KindNumber,
			Filterable: true,
			Sortable:   true,
			Number: func(row *worstRatedRow) (float64, bool) {
				return float64(row.answer.Negative), true
			},
		},
		{
			Name:       "positive",
			Kind:       memtable.KindNumber,
			Filterable: true,
			Sortable:   true,
			Number: func(row *worstRatedRow) (float64, bool) {
				return float64(row.answer.Positive), true
			},
		},
		{
			Name:     "lastRatedAt",
			Kind:     memtable.KindNumber,
			Sortable: true,
			Number: func(row *worstRatedRow) (float64, bool) {
				return float64(row.answer.LastRatedAt), true
			},
		},
	},
})

func (s *Service) ListWorstRatedTable(
	ctx context.Context,
	req *services.ListAgentWorstRatedTableRequest,
) (*services.AgentWorstRatedPage, error) {
	window, err := windowDays(req.WindowDays)
	if err != nil {
		return nil, err
	}
	control, err := s.control(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	scores, err := s.feedback.WorstRated(ctx, repositories.AIFeedbackWindowRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Since:             s.now() - int64(window)*secondsPerDay,
		Timezone:          s.timezoneOf(ctx, control, req.TenantInfo),
		Limit:             MaxWorstRatedAnswers,
	})
	if err != nil {
		return nil, err
	}

	rows := worstRatedRows(scores)
	samples := sampleScope{tenant: req.TenantInfo, viewer: req.ViewerID}

	samplesFirst := strings.TrimSpace(req.Table.Query) != "" ||
		worstRatedTable.References(&req.Table, fieldQuestion) ||
		worstRatedTable.References(&req.Table, fieldAgentName) ||
		worstRatedTable.References(&req.Table, fieldAgentDefinition)
	if samplesFirst {
		if err = s.attachSamples(ctx, samples, scores, rows); err != nil {
			return nil, err
		}
	}

	page, err := worstRatedTable.List(rows, &req.Table)
	if err != nil {
		return nil, err
	}

	if !samplesFirst && len(page.Items) > 0 {
		pageScores := make([]*repositories.AIFeedbackTargetScore, 0, len(page.Items))
		pageRows := make([]worstRatedRow, 0, len(page.Items))
		for _, row := range page.Items {
			pageScores = append(pageScores, scores[row.rank])
			pageRows = append(pageRows, *row)
		}
		if err = s.attachSamples(ctx, samples, pageScores, pageRows); err != nil {
			return nil, err
		}
	}

	out := &services.AgentWorstRatedPage{
		Edges:       make([]*services.AgentWorstRatedEdge, 0, len(page.Items)),
		HasNextPage: page.HasNextPage,
		TotalCount:  page.TotalCount,
	}
	for idx, row := range page.Items {
		out.Edges = append(out.Edges, &services.AgentWorstRatedEdge{
			Node:   row.answer,
			Cursor: page.Cursors[idx],
		})
	}

	return out, nil
}

type sampleScope struct {
	tenant pagination.TenantInfo
	viewer pulid.ID
}

func worstRatedRows(scores []*repositories.AIFeedbackTargetScore) []worstRatedRow {
	rows := make([]worstRatedRow, 0, len(scores))
	for idx, score := range scores {
		rows = append(rows, worstRatedRow{
			answer: &services.AgentWorstRatedAnswer{
				TargetType:  score.TargetType,
				TargetID:    score.TargetID,
				TargetPart:  score.TargetPart,
				Positive:    score.Positive,
				Negative:    score.Negative,
				LastRatedAt: score.LastRatedAt,
			},
			rank: idx,
		})
	}

	return rows
}

func (s *Service) attachSamples(
	ctx context.Context,
	scope sampleScope,
	scores []*repositories.AIFeedbackTargetScore,
	rows []worstRatedRow,
) error {
	sampleIDs := make([]pulid.ID, 0, len(scores))
	for _, score := range scores {
		if score.SampleID.IsNotNil() {
			sampleIDs = append(sampleIDs, score.SampleID)
		}
	}
	if len(sampleIDs) == 0 {
		return nil
	}

	samples := make(map[pulid.ID]*aifeedback.Feedback, len(sampleIDs))
	for chunk := range slices.Chunk(sampleIDs, agentDataChunk) {
		found, err := s.feedback.ListByIDs(ctx, repositories.ListAIFeedbackByIDsRequest{
			TenantInfo: scope.tenant,
			IDs:        chunk,
		})
		if err != nil {
			return err
		}
		for _, sample := range found {
			samples[sample.ID] = sample
		}
	}

	names, err := s.agentNames(ctx, scope.tenant, samples)
	if err != nil {
		return err
	}

	for idx := range rows {
		sample := samples[scores[idx].SampleID]
		if sample == nil {
			continue
		}

		answer := rows[idx].answer
		answer.Sample = sample
		answer.ThreadID = sample.ThreadID
		answer.CanOpenThread = sample.ThreadID != nil && scope.viewer.IsNotNil() &&
			sample.UserID == scope.viewer
		if sample.AgentDefinitionID != nil {
			answer.AgentDefinitionID = sample.AgentDefinitionID
			answer.AgentName = names[*sample.AgentDefinitionID]
		}
	}

	return nil
}

func questionOf(answer *services.AgentWorstRatedAnswer) string {
	if answer.Sample == nil || answer.Sample.TurnSnapshot == nil {
		return ""
	}

	return answer.Sample.TurnSnapshot.Question
}

func commentOf(answer *services.AgentWorstRatedAnswer) string {
	if answer.Sample == nil {
		return ""
	}

	return answer.Sample.Comment
}

func suiteRunStatusValues() []string {
	statuses := agentquality.AllSuiteRunStatuses()
	out := make([]string, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, string(status))
	}

	return out
}

func (s *Service) ListSuiteRunConnection(
	ctx context.Context,
	req *repositories.ListAgentSuiteRunConnectionRequest,
) (*pagination.CursorListResult[*agentquality.SuiteRun], error) {
	return s.suiteRuns.ListConnection(ctx, req)
}
