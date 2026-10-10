package agenttrustservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// maxReadyTools bounds how many tools one change of the switch reads.
const maxReadyTools = 500

type readyTool struct {
	promotion  agentports.ToolPromotion
	row        *agent.ToolTrust
	definition *agentdefinition.Definition
}

func (s *Service) PromotionCandidates(
	ctx context.Context,
	req *agentports.PromoteReadyRequest,
) ([]agentports.ToolPromotion, error) {
	ready, err := s.ready(ctx, req)
	if err != nil {
		return nil, err
	}

	out := make([]agentports.ToolPromotion, 0, len(ready))
	for idx := range ready {
		out = append(out, ready[idx].promotion)
	}
	return out, nil
}

func (s *Service) PromoteReady(
	ctx context.Context,
	req *agentports.PromoteReadyRequest,
) ([]agentports.ToolPromotion, error) {
	ready, err := s.ready(ctx, req)
	if err != nil || len(ready) == 0 {
		return []agentports.ToolPromotion{}, err
	}

	now := timeutils.NowUnix()
	promoted := make([]agentports.ToolPromotion, 0, len(ready))
	for idx := range ready {
		tool := &ready[idx]
		change := tierChange{
			tenant:     req.TenantInfo,
			definition: tool.definition,
			row:        tool.row,
			current:    tool.promotion.From,
			decidedBy:  req.DecidedBy,
			at:         now,
		}
		moved, moveErr := s.moveTier(ctx, change, tool.promotion.To, true)
		if moveErr != nil {
			s.l.Warn("could not promote a tool that earned it",
				zap.String("agent", tool.definition.ID.String()),
				zap.String("tool", tool.row.ToolName),
				zap.Error(moveErr),
			)
			continue
		}
		if !moved {
			continue
		}
		s.logTierChange(change, tool.promotion.To,
			"Tool promoted when earned autonomy was turned on: its streak had already met the threshold")
		promoted = append(promoted, tool.promotion)
	}

	return promoted, nil
}

// ready reads the tools whose streak meets the threshold and that have a
// tier above them their agent's ceiling and the tool's policy allow.
func (s *Service) ready(ctx context.Context, req *agentports.PromoteReadyRequest) ([]readyTool, error) {
	if req.Threshold <= 0 {
		return []readyTool{}, nil
	}

	rows, err := s.trust.ListReady(ctx, agentports.ListReadyToolTrustRequest{
		TenantInfo: req.TenantInfo,
		MinStreak:  req.Threshold,
		Limit:      maxReadyTools,
	})
	if err != nil || len(rows) == 0 {
		return []readyTool{}, err
	}

	ids := make([]pulid.ID, 0, len(rows))
	seen := make(map[pulid.ID]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.AgentDefinitionID]; ok {
			continue
		}
		seen[row.AgentDefinitionID] = struct{}{}
		ids = append(ids, row.AgentDefinitionID)
	}

	definitions, err := s.definitions.ListByIDs(ctx, agentdefinitionports.ListAgentDefinitionsByIDsRequest{
		IDs:        ids,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*agentdefinition.Definition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}

	out := make([]readyTool, 0, len(rows))
	for _, row := range rows {
		definition, ok := byID[row.AgentDefinitionID]
		if !ok || !row.ReadyForPromotion(req.Threshold) {
			continue
		}
		current := definition.EffectiveTier(row.ToolName, s.defaultTier(row.ToolName))
		next, ok := current.Next()
		if !ok || !definition.WithinCeiling(next) || next.Above(s.promotable(row.ToolName)) {
			continue
		}
		out = append(out, readyTool{
			promotion: agentports.ToolPromotion{
				AgentDefinitionID: definition.ID,
				AgentName:         definition.Name,
				ToolName:          row.ToolName,
				Streak:            row.Streak,
				From:              current,
				To:                next,
				Approvals:         row.Approvals,
				Rejections:        row.Rejections,
				TrackedSince:      row.CreatedAt,
			},
			row:        row,
			definition: definition,
		})
	}
	return out, nil
}

func (s *Service) PromoteTool(
	ctx context.Context,
	req *agentports.PromoteToolRequest,
) (*agentports.ToolPromotion, error) {
	ready, err := s.ready(ctx, &req.PromoteReadyRequest)
	if err != nil {
		return nil, err
	}

	for idx := range ready {
		tool := &ready[idx]
		if tool.definition.ID != req.AgentDefinitionID || tool.row.ToolName != req.ToolName {
			continue
		}
		change := tierChange{
			tenant:     req.TenantInfo,
			definition: tool.definition,
			row:        tool.row,
			current:    tool.promotion.From,
			decidedBy:  req.DecidedBy,
			at:         timeutils.NowUnix(),
		}
		moved, moveErr := s.moveTier(ctx, change, tool.promotion.To, true)
		if moveErr != nil {
			return nil, moveErr
		}
		if !moved {
			break
		}
		s.logTierChange(change, tool.promotion.To,
			"Tool promoted from a tune-up: its streak had met the threshold")
		return &tool.promotion, nil
	}

	return nil, errortypes.NewBusinessError(
		"{0} is no longer ready to move up a tier on this agent", req.ToolName,
	)
}
