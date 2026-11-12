package agentmemoryservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	fieldSubjectName     = "subjectName"
	subjectNameLookup    = 10
	subjectNamesReported = 5
)

func (s *Service) ResolveSubject(
	ctx context.Context,
	tenant pagination.TenantInfo,
	subjectType agentmemory.MemorySubjectType,
	name string,
) (pulid.ID, error) {
	name = strings.TrimSpace(name)
	kind := strings.ToLower(string(subjectType))
	if !subjectType.IsValid() || name == "" {
		return pulid.Nil, errortypes.NewValidationError(
			fieldSubjectName,
			errortypes.ErrInvalid,
			"Name the record with subjectType and subjectName, or leave both out",
		)
	}

	candidates, err := s.subjectNames.FindSubjectsByName(
		ctx,
		&agentmemoryports.FindMemorySubjectsRequest{
			TenantInfo: tenant,
			Type:       subjectType,
			Name:       name,
			Limit:      subjectNameLookup,
		},
	)
	if err != nil {
		return pulid.Nil, err
	}

	switch match := NamedSubject(candidates, name); {
	case match != nil:
		return match.ID, nil
	case len(candidates) == 0:
		return pulid.Nil, errortypes.NewValidationError(
			fieldSubjectName,
			errortypes.ErrInvalid,
			"No {0} is named \"{1}\". Check the name, or save the memory without a subject",
			kind, name,
		)
	default:
		return pulid.Nil, errortypes.NewValidationError(
			fieldSubjectName,
			errortypes.ErrInvalid,
			"Several {0} records match \"{1}\": {2}. Ask the person which one, then pass its "+
				"id as subjectId",
			kind, name, describeCandidates(candidates),
		)
	}
}

// NamedSubject picks the record a name means: the only candidate, or the only
// one whose name or code is exactly the name. Anything less certain is no
// match, so a memory is never filed under a guess.
func NamedSubject(
	candidates []agentmemoryports.MemorySubjectCandidate,
	name string,
) *agentmemoryports.MemorySubjectCandidate {
	if len(candidates) == 1 {
		return &candidates[0]
	}

	var exact *agentmemoryports.MemorySubjectCandidate
	for idx := range candidates {
		candidate := &candidates[idx]
		if !strings.EqualFold(candidate.Label, name) && !strings.EqualFold(candidate.Code, name) {
			continue
		}
		if exact != nil {
			return nil
		}
		exact = candidate
	}

	return exact
}

func describeCandidates(candidates []agentmemoryports.MemorySubjectCandidate) string {
	shown := candidates
	if len(shown) > subjectNamesReported {
		shown = shown[:subjectNamesReported]
	}
	parts := make([]string, 0, len(shown)+1)
	for _, candidate := range shown {
		parts = append(parts, candidate.Label+" ("+candidate.ID.String()+")")
	}
	if len(candidates) > len(shown) {
		parts = append(parts, "and more")
	}

	return strings.Join(parts, ", ")
}
