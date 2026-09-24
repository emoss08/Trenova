package agentmemoryservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxSubjectHops = 3

type SubjectResolver struct {
	links repositories.AgentMemorySubjectRepository
}

func NewSubjectResolver(links repositories.AgentMemorySubjectRepository) SubjectResolver {
	return SubjectResolver{links: links}
}

func (r SubjectResolver) Resolve(
	ctx context.Context,
	tenant pagination.TenantInfo,
	records []agent.EntityRef,
) ([]agent.MemorySubject, error) {
	subjects := newSubjectSet(agent.MaxMemorySubjectsPerTurn)
	visited := make(map[agent.MemoryRecordRef]struct{}, len(records))
	frontier := make([]agent.MemoryRecordRef, 0, len(records))

	for _, record := range records {
		ref, ok := agent.MemoryRecordRefOf(record.Type, record.ID)
		if !ok {
			continue
		}
		if subjectType, isSubject := ref.Kind.Subject(); isSubject {
			subjects.add(subjectType, ref.ID, agent.MemoryRelationDirect)
			continue
		}
		if _, seen := visited[ref]; seen || !ref.Kind.NamesOthers() {
			continue
		}
		visited[ref] = struct{}{}
		frontier = append(frontier, ref)
	}

	for hop := 0; hop < maxSubjectHops && len(frontier) > 0 && !subjects.full(); hop++ {
		named, err := r.linksOf(ctx, tenant, frontier)
		if err != nil {
			return subjects.list(), err
		}

		next := make([]agent.MemoryRecordRef, 0, len(frontier))
		for _, from := range frontier {
			for _, link := range named[from] {
				if subjectType, isSubject := link.Kind.Subject(); isSubject {
					subjects.add(subjectType, link.ID, agent.MemoryRelationRelated)
					continue
				}
				ref := agent.MemoryRecordRef{Kind: link.Kind, ID: link.ID}
				if _, seen := visited[ref]; seen || !ref.Kind.NamesOthers() {
					continue
				}
				visited[ref] = struct{}{}
				next = append(next, ref)
			}
		}
		frontier = next
	}

	return subjects.list(), nil
}

func (r SubjectResolver) linksOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	frontier []agent.MemoryRecordRef,
) (map[agent.MemoryRecordRef][]repositories.MemoryRecordLink, error) {
	named := make(map[agent.MemoryRecordRef][]repositories.MemoryRecordLink, len(frontier))
	if r.links == nil {
		return named, nil
	}

	kinds := make([]agent.MemoryRecordKind, 0, len(frontier))
	byKind := make(map[agent.MemoryRecordKind][]pulid.ID, len(frontier))
	for _, ref := range frontier {
		if _, ok := byKind[ref.Kind]; !ok {
			kinds = append(kinds, ref.Kind)
		}
		byKind[ref.Kind] = append(byKind[ref.Kind], ref.ID)
	}

	for _, kind := range kinds {
		links, err := r.links.ListRecordLinks(ctx, repositories.ListMemoryRecordLinksRequest{
			TenantInfo: tenant,
			Kind:       kind,
			IDs:        byKind[kind],
		})
		if err != nil {
			return named, fmt.Errorf("read the records a %s names: %w", kind, err)
		}
		for _, link := range links {
			from := agent.MemoryRecordRef{Kind: kind, ID: link.From}
			named[from] = append(named[from], link)
		}
	}

	return named, nil
}

type subjectSet struct {
	limit    int
	seen     map[repositories.MemorySubjectRef]struct{}
	subjects []agent.MemorySubject
}

func newSubjectSet(limit int) *subjectSet {
	return &subjectSet{
		limit:    limit,
		seen:     make(map[repositories.MemorySubjectRef]struct{}, limit),
		subjects: make([]agent.MemorySubject, 0, limit),
	}
}

func (s *subjectSet) full() bool { return len(s.subjects) >= s.limit }

func (s *subjectSet) add(
	subjectType agent.MemorySubjectType,
	id pulid.ID,
	relation agent.MemoryRelation,
) {
	if id.IsNil() || s.full() {
		return
	}
	key := repositories.MemorySubjectRef{Type: subjectType, ID: id}
	if _, ok := s.seen[key]; ok {
		return
	}
	s.seen[key] = struct{}{}
	s.subjects = append(s.subjects, agent.MemorySubject{
		Type:     subjectType,
		ID:       id,
		Relation: relation,
	})
}

func (s *subjectSet) list() []agent.MemorySubject { return s.subjects }

func subjectRefs(subjects []agent.MemorySubject) []repositories.MemorySubjectRef {
	refs := make([]repositories.MemorySubjectRef, 0, len(subjects))
	for _, subject := range subjects {
		refs = append(refs, repositories.MemorySubjectRef{Type: subject.Type, ID: subject.ID})
	}

	return refs
}
