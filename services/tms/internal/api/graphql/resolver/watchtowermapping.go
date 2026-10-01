package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

// watchtowerCursor is the edge cursor for one row, on the same keyset the
// feed pages by, so a client can resume from any edge it holds rather than
// only from the end of a page.
func watchtowerCursor(item *watchtower.Item) string {
	encoded, err := pagination.EncodeCursor(pagination.Cursor{
		CreatedAt: item.OccurredAt,
		ID:        item.ID,
	})
	if err != nil {
		return ""
	}

	return encoded
}

// watchtowerCountsModel renders the counts, listing the kinds in the order
// the feed's filters show them rather than whatever order the aggregate
// came back in.
func watchtowerCountsModel(counts *services.WatchtowerCounts) *gqlmodel.WatchtowerCounts {
	byKind := make([]*gqlmodel.WatchtowerKindSummary, 0, len(counts.ByKind))
	for _, kind := range watchtower.AllSourceKinds() {
		count, ok := counts.ByKind[kind]
		if !ok {
			continue
		}
		byKind = append(byKind, &gqlmodel.WatchtowerKindSummary{
			Kind:  kind,
			Label: kind.Label(),
			Count: count,
		})
	}

	return &gqlmodel.WatchtowerCounts{
		Unresolved:     counts.Unresolved,
		Critical:       counts.Critical,
		Unseen:         counts.Unseen,
		UnseenCritical: counts.UnseenCritical,
		ByKind:         byKind,
		SeenAt:         int(counts.SeenAt),
	}
}

func watchtowerHandOffModel(
	result *services.HandOffWatchtowerItemResult,
) *gqlmodel.WatchtowerHandOffResult {
	templates := make([]string, 0, len(result.Templates))
	for _, template := range result.Templates {
		templates = append(templates, string(template))
	}

	return &gqlmodel.WatchtowerHandOffResult{
		Item:        result.Item,
		Run:         result.Run,
		Subscribers: definitionsOrEmpty(result.Subscribers),
		Candidates:  definitionsOrEmpty(result.Candidates),
		Templates:   templates,
	}
}

// definitionsOrEmpty keeps a non-null list non-null: a hand-off with no
// subscribers reports an empty list rather than failing the field.
func definitionsOrEmpty(definitions []*agentdefinition.Definition) []*agentdefinition.Definition {
	if definitions == nil {
		return []*agentdefinition.Definition{}
	}

	return definitions
}
