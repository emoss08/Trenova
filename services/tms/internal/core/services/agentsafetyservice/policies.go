package agentsafetyservice

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/memtable"
)

const (
	GeneralResource        = "general"
	FieldRunsWithoutPerson = "runsWithoutPerson"
	toolRuleCursorScope    = "agent_tool_rule"
)

type policyEntry struct {
	view     services.AgentToolPolicyView
	resource string
}

type toolRuleRow struct {
	entry *policyEntry
	alone bool
}

func (s *Service) indexPolicies(views []services.AgentToolPolicyView) {
	slices.SortStableFunc(views, func(a, b services.AgentToolPolicyView) int {
		return cmp.Compare(a.Policy.Name, b.Policy.Name)
	})

	s.entries = make([]policyEntry, 0, len(views))
	s.byName = make(map[string]int, len(views))
	resources := make(map[string]struct{}, len(views))
	for idx := range views {
		view := views[idx]
		if _, duplicate := s.byName[view.Policy.Name]; duplicate {
			continue
		}

		entry := policyEntry{
			view:     view,
			resource: GeneralResource,
		}
		if view.Needs != nil {
			entry.resource = view.Needs.Resource.String()
		}

		s.byName[view.Policy.Name] = len(s.entries)
		s.entries = append(s.entries, entry)
		resources[entry.resource] = struct{}{}
	}

	s.resources = make([]string, 0, len(resources))
	for resource := range resources {
		s.resources = append(s.resources, resource)
	}
	slices.Sort(s.resources)

	s.toolRules = newToolRuleTable(s.resources)
	s.agentTools = newAgentToolTable(s.resources)
}

func (s *Service) ListToolPolicies(
	ctx context.Context,
	req *services.ListAgentToolPoliciesRequest,
) (*services.AgentToolPolicyPage, error) {
	if req == nil {
		req = &services.ListAgentToolPoliciesRequest{}
	}

	rows := make([]toolRuleRow, len(s.entries))
	for idx := range s.entries {
		rows[idx].entry = &s.entries[idx]
	}

	attended := req.WithAttendance || s.toolRules.References(&req.Table, FieldRunsWithoutPerson)
	if attended {
		survey, err := s.survey(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		for idx := range rows {
			_, rows[idx].alone = survey.unattended[rows[idx].entry.view.Policy.Name]
		}
	}

	page, err := s.toolRules.List(rows, &req.Table)
	if err != nil {
		return nil, err
	}

	out := &services.AgentToolPolicyPage{
		Edges:       make([]services.AgentToolPolicyEdge, 0, len(page.Items)),
		HasNextPage: page.HasNextPage,
		TotalCount:  page.TotalCount,
	}
	for idx, row := range page.Items {
		edge := services.AgentToolPolicyEdge{
			View:   row.entry.view,
			Cursor: page.Cursors[idx],
		}
		if attended {
			alone := row.alone
			edge.RunsWithoutPerson = &alone
		}
		out.Edges = append(out.Edges, edge)
	}

	return out, nil
}

func newToolRuleTable(resources []string) *memtable.Table[toolRuleRow] {
	view := func(row *toolRuleRow) *services.AgentToolPolicyView { return &row.entry.view }

	return memtable.New(memtable.Config[toolRuleRow]{
		CursorScope: toolRuleCursorScope,
		Search: func(row *toolRuleRow) string {
			return row.entry.view.Policy.Name + "\x00" + row.entry.view.Title
		},
		Order: func(a, b *toolRuleRow) int {
			return cmp.Compare(a.entry.view.Policy.Name, b.entry.view.Policy.Name)
		},
		Fields: []memtable.Field[toolRuleRow]{
			{
				Name:       "title",
				Kind:       memtable.KindText,
				Filterable: true,
				Sortable:   true,
				Text:       func(row *toolRuleRow) string { return view(row).Title },
			},
			{
				Name:       "name",
				Kind:       memtable.KindText,
				Filterable: true,
				Sortable:   true,
				Text:       func(row *toolRuleRow) string { return view(row).Policy.Name },
			},
			policyEgressField(view),
			policyTierField(view),
			{
				Name:       "resource",
				Kind:       memtable.KindEnum,
				Values:     resources,
				Filterable: true,
				Sortable:   true,
				Text:       func(row *toolRuleRow) string { return row.entry.resource },
			},
			policyKindField(view),
			policyExternalReadField(view),
			{
				Name:       "leavesOrganization",
				Kind:       memtable.KindBoolean,
				Filterable: true,
				Sortable:   true,
				Bool:       func(row *toolRuleRow) bool { return view(row).Leaves },
			},
			{
				Name:       FieldRunsWithoutPerson,
				Kind:       memtable.KindBoolean,
				Filterable: true,
				Sortable:   true,
				Bool:       func(row *toolRuleRow) bool { return row.alone },
			},
		},
	})
}

func policyEgressField[T any](
	view func(*T) *services.AgentToolPolicyView,
) memtable.Field[T] {
	return memtable.Field[T]{
		Name:       "egress",
		Kind:       memtable.KindSet,
		Values:     egressValues(),
		Filterable: true,
		Sortable:   true,
		Set: func(row *T) []string {
			return egressStrings(view(row).Policy.Egress)
		},
		Compare: func(a, b *T) int {
			return cmp.Compare(egressRank(view(a).Policy.Egress), egressRank(view(b).Policy.Egress))
		},
	}
}

func policyTierField[T any](
	view func(*T) *services.AgentToolPolicyView,
) memtable.Field[T] {
	return memtable.Field[T]{
		Name: "maxTier",
		Kind: memtable.KindEnum,
		Values: []string{
			string(agent.TierPropose),
			string(agent.TierActWithApproval),
			string(agent.TierAutoExecute),
		},
		Filterable: true,
		Sortable:   true,
		Text:       func(row *T) string { return string(view(row).Promotable) },
		Compare: func(a, b *T) int {
			return cmp.Compare(view(a).Promotable.Rank(), view(b).Promotable.Rank())
		},
	}
}

func policyKindField[T any](
	view func(*T) *services.AgentToolPolicyView,
) memtable.Field[T] {
	return memtable.Field[T]{
		Name: "kind",
		Kind: memtable.KindEnum,
		Values: []string{
			agent.ToolKindQuery.String(),
			agent.ToolKindAction.String(),
			agent.ToolKindRuntime.String(),
		},
		Filterable: true,
		Sortable:   true,
		Text:       func(row *T) string { return view(row).Policy.Kind.String() },
	}
}

func policyExternalReadField[T any](
	view func(*T) *services.AgentToolPolicyView,
) memtable.Field[T] {
	return memtable.Field[T]{
		Name: "readsExternal",
		Kind: memtable.KindEnum,
		Values: []string{
			agent.ExternalReadNever.String(),
			agent.ExternalReadAlways.String(),
			agent.ExternalReadMarked.String(),
		},
		Filterable: true,
		Sortable:   true,
		Text: func(row *T) string {
			read := view(row).Policy.ReadsExternal
			if !read.IsValid() {
				return agent.ExternalReadNever.String()
			}

			return read.String()
		},
		Compare: func(a, b *T) int {
			return cmp.Compare(
				externalReadRank(view(a).Policy.ReadsExternal),
				externalReadRank(view(b).Policy.ReadsExternal),
			)
		},
	}
}

func externalReadRank(read agent.ExternalRead) int {
	switch read {
	case agent.ExternalReadMarked:
		return 1
	case agent.ExternalReadAlways:
		return 2
	default:
		return 0
	}
}

func egressValues() []string {
	classes := agent.EgressClasses()
	out := make([]string, 0, len(classes))
	for _, class := range classes {
		out = append(out, class.String())
	}

	return out
}

func egressStrings(classes []agent.EgressClass) []string {
	out := make([]string, 0, len(classes))
	for _, class := range classes {
		out = append(out, class.String())
	}

	return out
}

func egressRank(classes []agent.EgressClass) int {
	order := agent.EgressClasses()
	rank := -1
	for _, class := range classes {
		if idx := slices.Index(order, class); idx > rank {
			rank = idx
		}
	}

	return rank
}

func compareFold(a, b string) int {
	return cmp.Compare(strings.ToLower(a), strings.ToLower(b))
}
