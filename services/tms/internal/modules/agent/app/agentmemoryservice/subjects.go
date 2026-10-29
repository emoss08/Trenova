package agentmemoryservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxSubjectHops = 3

type SubjectResolver struct {
	links agentmemoryports.AgentMemorySubjectRepository
}

func NewSubjectResolver(links agentmemoryports.AgentMemorySubjectRepository) SubjectResolver {
	return SubjectResolver{links: links}
}

func (r SubjectResolver) Resolve(
	ctx context.Context,
	tenant pagination.TenantInfo,
	records []agentpagecontext.EntityRef,
) ([]agentmemory.MemorySubject, error) {
	subjects := newSubjectSet(agentmemory.MaxMemorySubjectsPerTurn)
	visited := make(map[agentmemory.MemoryRecordRef]struct{}, len(records))
	frontier := make([]agentmemory.MemoryRecordRef, 0, len(records))

	for _, record := range records {
		ref, ok := agentmemory.MemoryRecordRefOf(record.Type, record.ID)
		if !ok {
			continue
		}
		if subjectType, isSubject := ref.Kind.Subject(); isSubject {
			subjects.add(subjectType, ref.ID, agentmemory.MemoryRelationDirect)
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

		next := make([]agentmemory.MemoryRecordRef, 0, len(frontier))
		for _, from := range frontier {
			for _, link := range named[from] {
				if subjectType, isSubject := link.Kind.Subject(); isSubject {
					subjects.add(subjectType, link.ID, agentmemory.MemoryRelationRelated)
					continue
				}
				ref := agentmemory.MemoryRecordRef{Kind: link.Kind, ID: link.ID}
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
	frontier []agentmemory.MemoryRecordRef,
) (map[agentmemory.MemoryRecordRef][]agentmemoryports.MemoryRecordLink, error) {
	named := make(
		map[agentmemory.MemoryRecordRef][]agentmemoryports.MemoryRecordLink,
		len(frontier),
	)
	if r.links == nil {
		return named, nil
	}

	kinds := make([]agentmemory.MemoryRecordKind, 0, len(frontier))
	byKind := make(map[agentmemory.MemoryRecordKind][]pulid.ID, len(frontier))
	for _, ref := range frontier {
		if _, ok := byKind[ref.Kind]; !ok {
			kinds = append(kinds, ref.Kind)
		}
		byKind[ref.Kind] = append(byKind[ref.Kind], ref.ID)
	}

	for _, kind := range kinds {
		links, err := r.links.ListRecordLinks(ctx, &agentmemoryports.ListMemoryRecordLinksRequest{
			TenantInfo: tenant,
			Kind:       kind,
			IDs:        byKind[kind],
		})
		if err != nil {
			return named, fmt.Errorf("read the records a %s names: %w", kind, err)
		}
		for _, link := range links {
			from := agentmemory.MemoryRecordRef{Kind: kind, ID: link.From}
			named[from] = append(named[from], link)
		}
	}

	return named, nil
}

type subjectSet struct {
	limit    int
	seen     map[agentmemoryports.MemorySubjectRef]struct{}
	subjects []agentmemory.MemorySubject
}

func newSubjectSet(limit int) *subjectSet {
	return &subjectSet{
		limit:    limit,
		seen:     make(map[agentmemoryports.MemorySubjectRef]struct{}, limit),
		subjects: make([]agentmemory.MemorySubject, 0, limit),
	}
}

func (s *subjectSet) full() bool { return len(s.subjects) >= s.limit }

func (s *subjectSet) add(
	subjectType agentmemory.MemorySubjectType,
	id pulid.ID,
	relation agentmemory.MemoryRelation,
) {
	if id.IsNil() || s.full() {
		return
	}
	key := agentmemoryports.MemorySubjectRef{Type: subjectType, ID: id}
	if _, ok := s.seen[key]; ok {
		return
	}
	s.seen[key] = struct{}{}
	s.subjects = append(s.subjects, agentmemory.MemorySubject{
		Type:     subjectType,
		ID:       id,
		Relation: relation,
	})
}

func (s *subjectSet) list() []agentmemory.MemorySubject { return s.subjects }

func subjectRefs(subjects []agentmemory.MemorySubject) []agentmemoryports.MemorySubjectRef {
	refs := make([]agentmemoryports.MemorySubjectRef, 0, len(subjects))
	for _, subject := range subjects {
		refs = append(refs, agentmemoryports.MemorySubjectRef{Type: subject.Type, ID: subject.ID})
	}

	return refs
}
