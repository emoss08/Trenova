package agentsafetyservice

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxComparedAgents    = 10
	agentToolCursorScope = "agent_tool_safety"
)

type agentToolRow struct {
	safety services.AgentToolSafety
	entry  *policyEntry
}

func (s *Service) ListAgentTools(
	ctx context.Context,
	req *services.ListAgentToolSafetyRequest,
) (*services.AgentToolSafetyPage, error) {
	if req == nil {
		req = &services.ListAgentToolSafetyRequest{}
	}

	ids, err := validateComparedAgents(req.AgentIDs)
	if err != nil {
		return nil, err
	}

	rows, err := s.agentToolRows(ctx, req, ids)
	if err != nil {
		return nil, err
	}

	page, err := s.agentTools.List(rows, &req.Table)
	if err != nil {
		return nil, err
	}

	out := &services.AgentToolSafetyPage{
		Edges:       make([]services.AgentToolSafetyEdge, 0, len(page.Items)),
		HasNextPage: page.HasNextPage,
		TotalCount:  page.TotalCount,
	}
	for idx, row := range page.Items {
		out.Edges = append(out.Edges, services.AgentToolSafetyEdge{
			Node:   row.safety,
			Cursor: page.Cursors[idx],
		})
	}

	return out, nil
}

func (s *Service) agentToolRows(
	ctx context.Context,
	req *services.ListAgentToolSafetyRequest,
	ids []pulid.ID,
) ([]agentToolRow, error) {
	definitions, err := s.definitionsByID(ctx, &services.ListAgentSafetyRequest{
		TenantInfo: req.TenantInfo,
		AgentIDs:   ids,
	})
	if err != nil {
		return nil, err
	}
	if len(definitions) == 0 {
		return []agentToolRow{}, nil
	}

	control, err := s.controls.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read agent control for safety: %w", err)
	}

	trust, err := s.trustFor(ctx, req.TenantInfo, definitions)
	if err != nil {
		return nil, err
	}

	rows := make([]agentToolRow, 0, len(definitions)*len(s.entries)/2+1)
	for _, definition := range definitions {
		assessed := s.Assess(ctx, &services.AssessAgentSafetyRequest{
			Subject: &services.AgentSafetySubject{Agent: definition, Control: control},
			Trust:   trust[definition.ID],
		})
		for idx := range assessed {
			entryIdx, ok := s.byName[assessed[idx].PolicyName]
			if !ok {
				continue
			}
			rows = append(rows, agentToolRow{
				safety: assessed[idx],
				entry:  &s.entries[entryIdx],
			})
		}
	}

	return rows, nil
}

func validateComparedAgents(ids []pulid.ID) ([]pulid.ID, error) {
	if len(ids) == 0 {
		return nil, errortypes.NewValidationError(
			"agentIds", errortypes.ErrRequired, "Choose at least one agent",
		)
	}
	if len(ids) > MaxComparedAgents {
		return nil, errortypes.NewValidationError(
			"agentIds", errortypes.ErrInvalid, "At most 10 agents can be compared at once",
		)
	}

	return validateAgentIDs(ids)
}

func newAgentToolTable(resources []string) *memtable.Table[agentToolRow] {
	view := func(row *agentToolRow) *services.AgentToolPolicyView { return &row.entry.view }

	return memtable.New(memtable.Config[agentToolRow]{
		CursorScope: agentToolCursorScope,
		Search: func(row *agentToolRow) string {
			return row.safety.PolicyName + "\x00" + row.entry.view.Title + "\x00" + row.safety.AgentName
		},
		Order: func(a, b *agentToolRow) int {
			return cmp.Or(
				compareFold(a.safety.AgentName, b.safety.AgentName),
				cmp.Compare(a.safety.AgentID.String(), b.safety.AgentID.String()),
				cmp.Compare(answerExposure(b.safety.Clean.Answer), answerExposure(a.safety.Clean.Answer)),
				cmp.Compare(a.safety.PolicyName, b.safety.PolicyName),
			)
		},
		Fields: []memtable.Field[agentToolRow]{
			{
				Name:       "agentId",
				Kind:       memtable.KindEnum,
				Filterable: true,
				Text:       func(row *agentToolRow) string { return row.safety.AgentID.String() },
			},
			{
				Name:       "agentName",
				Kind:       memtable.KindText,
				Filterable: true,
				Sortable:   true,
				Text:       func(row *agentToolRow) string { return row.safety.AgentName },
			},
			{
				Name:       "title",
				Kind:       memtable.KindText,
				Filterable: true,
				Sortable:   true,
				Text:       func(row *agentToolRow) string { return view(row).Title },
			},
			{
				Name:       "policyName",
				Kind:       memtable.KindText,
				Filterable: true,
				Sortable:   true,
				Text:       func(row *agentToolRow) string { return row.safety.PolicyName },
			},
			policyEgressField(view),
			policyTierField(view),
			policyKindField(view),
			{
				Name:       "resource",
				Kind:       memtable.KindEnum,
				Values:     resources,
				Filterable: true,
				Sortable:   true,
				Text:       func(row *agentToolRow) string { return row.entry.resource },
			},
			answerField("clean", func(row *agentToolRow) *services.ToolAutonomy {
				return &row.safety.Clean
			}),
			answerField("tainted", func(row *agentToolRow) *services.ToolAutonomy {
				return &row.safety.Tainted
			}),
			{
				Name: "heldBy",
				Kind: memtable.KindSet,
				Values: []string{
					agenttoolpolicy.HeldByAgentCeiling,
					agenttoolpolicy.HeldByToolMax,
					agenttoolpolicy.HeldByEgressClass,
					agenttoolpolicy.HeldByCondition,
					agenttoolpolicy.HeldByTainted,
					agenttoolpolicy.HeldByToolTier,
					agenttoolpolicy.HeldByPersonalExemption,
					agenttoolpolicy.HeldByShadowMode,
					agenttoolpolicy.HeldBySimulationMode,
				},
				Filterable: true,
				Set: func(row *agentToolRow) []string {
					return heldByUnion(&row.safety)
				},
			},
		},
	})
}

func answerField(
	name string,
	autonomy func(*agentToolRow) *services.ToolAutonomy,
) memtable.Field[agentToolRow] {
	return memtable.Field[agentToolRow]{
		Name: name,
		Kind: memtable.KindEnum,
		Values: []string{
			agent.AutonomyRunsOnItsOwn.String(),
			agent.AutonomyConditional.String(),
			agent.AutonomyNeedsApproval.String(),
			agent.AutonomyProposeOnly.String(),
			agent.AutonomySimulated.String(),
		},
		Filterable: true,
		Sortable:   true,
		Text:       func(row *agentToolRow) string { return autonomy(row).Answer.String() },
		Compare: func(a, b *agentToolRow) int {
			return cmp.Compare(answerExposure(autonomy(a).Answer), answerExposure(autonomy(b).Answer))
		},
	}
}

func answerExposure(answer agent.AutonomyAnswer) int {
	switch answer {
	case agent.AutonomyRunsOnItsOwn:
		return 4
	case agent.AutonomyConditional:
		return 3
	case agent.AutonomyNeedsApproval:
		return 2
	case agent.AutonomyProposeOnly:
		return 1
	default:
		return 0
	}
}

func heldByUnion(tool *services.AgentToolSafety) []string {
	out := make([]string, 0, len(tool.Clean.HeldBy)+len(tool.Tainted.HeldBy))
	out = append(out, tool.Clean.HeldBy...)
	for _, key := range tool.Tainted.HeldBy {
		if !slices.Contains(out, key) {
			out = append(out, key)
		}
	}

	return out
}
