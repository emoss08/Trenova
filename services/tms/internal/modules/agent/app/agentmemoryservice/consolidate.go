package agentmemoryservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentmemoryrank"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/setutils"
	"go.uber.org/zap"
)

const (
	DuplicateOverlap     = 0.8
	maxFunctionWordRunes = 3
	maxJudgedNeighbors   = 8
	neighborCandidates   = 16
)

type consolidation struct {
	req     *agentmemoryports.RememberRequest
	plan    *agentmemoryports.RememberPlan
	actor   *permissionports.RequestActor
	manages bool
	judge   bool
}

func (s *Service) consolidate(ctx context.Context, c *consolidation) {
	plan := c.plan
	if plan.Existing != nil || plan.Replaced != nil {
		return
	}
	entity := plan.Memory

	same := sameMemoryRequest(c.req.TenantInfo, entity)
	same.IncludeSuggested = entity.Status == agentmemory.MemoryStatusSuggested
	var near []pulid.ID
	if c.judge {
		near = s.similarIDs(ctx, c.req, entity, same)
	}
	neighbors, err := s.repo.ListNeighbors(ctx, &agentmemoryports.ListAgentMemoryNeighborsRequest{
		Same:  same,
		Terms: agentmemoryrank.RecallTerms(entity.Content),
		IDs:   near,
		Limit: neighborCandidates,
	})
	if err != nil {
		s.l.Warn("agent memory: the memories beside a new one could not be read; saving it as is",
			zap.String("organization", c.req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)

		return
	}

	if duplicate := DuplicateOf(entity, neighbors); duplicate != nil {
		plan.Existing = duplicate

		return
	}

	if !c.judge {
		return
	}
	judged := judgeable(neighbors, maxJudgedNeighbors)
	wider := judgeable(s.widerNeighbors(ctx, c, same), maxJudgedNeighbors-len(judged))
	if len(judged)+len(wider) == 0 {
		return
	}

	judgement, err := s.judge.Judge(ctx, &agentmemoryports.JudgeMemoryRequest{
		TenantInfo: c.req.TenantInfo,
		Memory:     entity,
		Candidates: append(
			append(make([]*agentmemory.Memory, 0, len(judged)+len(wider)), judged...),
			wider...),
		Attribution: consolidationAttribution(c.req, entity),
	})
	if err != nil {
		s.l.Warn(
			"agent memory: a new memory could not be checked against kept ones; saving it as is",
			zap.String("organization", c.req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)

		return
	}

	s.applyJudgement(c, judged, judgement)
	if plan.Existing == nil && plan.Replaced == nil {
		applyWiderJudgement(plan, wider, judgement)
	}
}

// widerNeighbors reads the memories kept for a wider audience that the new
// memory's readers also read, about the same record and tool: the ones a
// personal or team memory can agree with or differ from without replacing.
func (s *Service) widerNeighbors(
	ctx context.Context,
	c *consolidation,
	same *agentmemoryports.FindActiveAgentMemoryRequest,
) []*agentmemory.Memory {
	entity := c.plan.Memory
	if entity.Scope != agentmemory.MemoryScopeUser && entity.Scope != agentmemory.MemoryScopeRole {
		return nil
	}

	var roleIDs []pulid.ID
	if entity.Scope == agentmemory.MemoryScopeUser && entity.OwnerUserID != nil {
		roleIDs = s.promptReader(ctx, c.req.TenantInfo, *entity.OwnerUserID).RoleIDs
	}
	wider, err := s.repo.ListNeighbors(ctx, &agentmemoryports.ListAgentMemoryNeighborsRequest{
		Same:    same,
		Terms:   agentmemoryrank.RecallTerms(entity.Content),
		Limit:   neighborCandidates,
		Wider:   true,
		RoleIDs: roleIDs,
	})
	if err != nil {
		s.l.Warn("agent memory: the shared memories beside a new one could not be read",
			zap.String("organization", c.req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)

		return nil
	}

	return wider
}

// applyWiderJudgement settles a personal or team memory against what a wider
// audience keeps. One that already says it is kept in place of a copy; one
// that says otherwise stays, and the new memory is kept beside it and marked
// as differing so the person hears that the two disagree.
func applyWiderJudgement(
	plan *agentmemoryports.RememberPlan,
	wider []*agentmemory.Memory,
	judgement *agentmemoryports.MemoryJudgement,
) {
	for _, neighbor := range wider {
		if judgement.Verdict(neighbor.ID) == agentmemoryports.MemoryVerdictSame &&
			neighbor.Kind == plan.Memory.Kind {
			plan.Existing = neighbor

			return
		}
	}
	for _, neighbor := range wider {
		if judgement.Verdict(neighbor.ID) == agentmemoryports.MemoryVerdictContradicts {
			plan.Differs = neighbor

			return
		}
	}
}

func (s *Service) applyJudgement(
	c *consolidation,
	judged []*agentmemory.Memory,
	judgement *agentmemoryports.MemoryJudgement,
) {
	entity := c.plan.Memory
	for _, neighbor := range judged {
		if judgement.Verdict(neighbor.ID) != agentmemoryports.MemoryVerdictSame {
			continue
		}
		if entity.Kind.Rank() < neighbor.Kind.Rank() && neighbor.Status.Replaceable() {
			replaceContradicted(c, neighbor)

			return
		}
		c.plan.Existing = neighbor

		return
	}

	for _, neighbor := range judged {
		if judgement.Verdict(neighbor.ID) == agentmemoryports.MemoryVerdictContradicts &&
			neighbor.Status.Replaceable() {
			replaceContradicted(c, neighbor)

			return
		}
	}
}

func replaceContradicted(c *consolidation, replaced *agentmemory.Memory) {
	entity := c.plan.Memory
	id := replaced.ID
	entity.SupersedesID = &id
	c.plan.Replaced = replaced
	c.plan.Contradicted = true

	if !contradictionTakesEffect(c, replaced) {
		c.plan.Held = true
		entity.Status = agentmemory.MemoryStatusSuggested
	}
}

func contradictionTakesEffect(c *consolidation, replaced *agentmemory.Memory) bool {
	entity := c.plan.Memory
	if entity.Tainted || entity.Status != agentmemory.MemoryStatusActive {
		return false
	}

	return mayReplaceNow(replaced, entity, c.req, c.actor, c.manages)
}

func DuplicateOf(entity *agentmemory.Memory, neighbors []*agentmemory.Memory) *agentmemory.Memory {
	words := agentsearch.TokenSet(entity.Content)
	if len(words) == 0 {
		return nil
	}
	for _, neighbor := range neighbors {
		if neighbor == nil || neighbor.Kind != entity.Kind {
			continue
		}
		kept := agentsearch.TokenSet(neighbor.Content)
		if setutils.Jaccard(words, kept) >= DuplicateOverlap &&
			onlyFunctionWordsDiffer(words, kept) {
			return neighbor
		}
	}

	return nil
}

func onlyFunctionWordsDiffer(a, b map[string]struct{}) bool {
	for word := range a {
		if _, shared := b[word]; !shared && utf8.RuneCountInString(word) > maxFunctionWordRunes {
			return false
		}
	}
	for word := range b {
		if _, shared := a[word]; !shared && utf8.RuneCountInString(word) > maxFunctionWordRunes {
			return false
		}
	}

	return true
}

func judgeable(neighbors []*agentmemory.Memory, limit int) []*agentmemory.Memory {
	judged := make([]*agentmemory.Memory, 0, max(min(len(neighbors), limit), 0))
	for _, neighbor := range neighbors {
		if len(judged) >= limit {
			break
		}
		if neighbor == nil || strings.TrimSpace(neighbor.Content) == "" {
			continue
		}
		judged = append(judged, neighbor)
	}

	return judged
}

func (s *Service) similarIDs(
	ctx context.Context,
	req *agentmemoryports.RememberRequest,
	entity *agentmemory.Memory,
	same *agentmemoryports.FindActiveAgentMemoryRequest,
) []pulid.ID {
	if same.Subject != nil || strings.TrimSpace(same.ToolName) != "" {
		return nil
	}

	similar, err := s.vectors.SimilarMemories(ctx, &retrievalports.SimilarMemoriesRequest{
		TenantInfo:  req.TenantInfo,
		Text:        entity.Content,
		Limit:       neighborCandidates,
		Attribution: consolidationAttribution(req, entity),
	})
	if err != nil {
		s.l.Warn("agent memory: memories near a new one by meaning could not be read",
			zap.String("organization", req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)

		return nil
	}
	if !similar.Semantics.Used {
		return nil
	}

	ids := make([]pulid.ID, 0, len(similar.Memories))
	for _, hit := range similar.Memories {
		if hit.Similarity >= similar.Floor {
			ids = append(ids, hit.MemoryID)
		}
	}

	return ids
}

func consolidationAttribution(
	req *agentmemoryports.RememberRequest,
	entity *agentmemory.Memory,
) aiusageports.AIUsageAttribution {
	return aiusageports.AIUsageAttribution{
		UserID:            req.PersonUserID,
		AgentDefinitionID: pulid.ConvertFromPtr(entity.AgentDefinitionID),
		ThreadID:          req.ThreadID,
		RunID:             req.RunID,
		Feature:           aiusage.FeatureAgentMemoryConsolidation,
	}
}
