package agentsafetyservice

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	DefaultToolPolicyPageSize = 25
	GeneralResource           = "general"
	MaxToolPolicyQueryLength  = 200
	toolPolicyCursorScope     = "agent_tool_policy"
)

type policyEntry struct {
	view     services.AgentToolPolicyView
	search   string
	resource string
}

type policyFilter struct {
	query      string
	egress     agent.EgressClass
	resource   string
	kind       agent.ToolKind
	unattended map[string]struct{}
	wantAlone  bool
	byAlone    bool
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
			search:   strings.ToLower(view.Policy.Name + "\x00" + view.Title),
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
}

func (s *Service) ListToolPolicies(
	ctx context.Context,
	req *services.ListAgentToolPoliciesRequest,
) (*services.AgentToolPolicyPage, error) {
	if req == nil {
		req = &services.ListAgentToolPoliciesRequest{}
	}

	filter, err := s.policyFilter(ctx, req)
	if err != nil {
		return nil, err
	}

	start, err := s.startAfter(req.After)
	if err != nil {
		return nil, err
	}

	limit := toolPolicyPageSize(req.First)
	page := &services.AgentToolPolicyPage{
		Edges: make([]services.AgentToolPolicyEdge, 0, min(limit, len(s.entries))),
	}

	from := start
	if req.IncludeTotalCount {
		from = 0
	}

	total := 0
	for idx := from; idx < len(s.entries); idx++ {
		entry := &s.entries[idx]
		if !filter.matches(entry) {
			continue
		}
		total++
		if idx < start {
			continue
		}
		if len(page.Edges) < limit {
			page.Edges = append(page.Edges, services.AgentToolPolicyEdge{
				View:   entry.view,
				Cursor: pagination.EncodeKeyCursor(toolPolicyCursorScope, entry.view.Policy.Name),
			})
			continue
		}

		page.HasNextPage = true
		if !req.IncludeTotalCount {
			break
		}
	}

	if req.IncludeTotalCount {
		page.TotalCount = &total
	}

	return page, nil
}

func (s *Service) policyFilter(
	ctx context.Context,
	req *services.ListAgentToolPoliciesRequest,
) (*policyFilter, error) {
	query := strings.TrimSpace(req.Query)
	multiErr := errortypes.NewMultiError()
	if utf8.RuneCountInString(query) > MaxToolPolicyQueryLength {
		multiErr.Add("query", errortypes.ErrInvalid,
			"Search must be at most 200 characters")
	}
	if req.Egress != "" && !req.Egress.IsValid() {
		multiErr.Add("egress", errortypes.ErrInvalid, "Class is not recognized")
	}
	if req.Kind != "" && !req.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Kind is not recognized")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	filter := &policyFilter{
		query:    strings.ToLower(query),
		egress:   req.Egress,
		resource: strings.TrimSpace(req.Resource),
		kind:     req.Kind,
	}
	if req.RunsWithoutPerson != nil {
		survey, err := s.survey(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		filter.byAlone = true
		filter.wantAlone = *req.RunsWithoutPerson
		filter.unattended = survey.unattended
	}

	return filter, nil
}

func (f *policyFilter) matches(entry *policyEntry) bool {
	policy := &entry.view.Policy
	if f.egress != "" && !slices.Contains(policy.Egress, f.egress) {
		return false
	}
	if f.kind != "" && policy.Kind != f.kind {
		return false
	}
	if f.resource != "" && entry.resource != f.resource {
		return false
	}
	if f.query != "" && !strings.Contains(entry.search, f.query) {
		return false
	}
	if f.byAlone {
		_, alone := f.unattended[policy.Name]
		if alone != f.wantAlone {
			return false
		}
	}

	return true
}

func (s *Service) startAfter(after string) (int, error) {
	if after == "" {
		return 0, nil
	}

	name, err := pagination.DecodeKeyCursor(toolPolicyCursorScope, after)
	if err != nil {
		return 0, errortypes.NewValidationError(
			"after",
			errortypes.ErrInvalidFormat,
			"Cursor is invalid",
		)
	}

	idx, found := slices.BinarySearchFunc(
		s.entries,
		name,
		func(entry policyEntry, target string) int {
			return cmp.Compare(entry.view.Policy.Name, target)
		},
	)
	if found {
		idx++
	}

	return idx, nil
}

func toolPolicyPageSize(first int) int {
	if first <= 0 {
		return DefaultToolPolicyPageSize
	}

	return pagination.ClampLimit(first)
}
