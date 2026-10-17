package watchtowerresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// watchtowerCountsModel renders the counts, listing the kinds in the order
// the feed's filters show them rather than whatever order the aggregate
// came back in.
func watchtowerCountsModel(counts *watchtowerports.WatchtowerCounts) *gqlmodel.WatchtowerCounts {
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
		Active:         counts.Active,
		ActiveCritical: counts.ActiveCritical,
		Unseen:         counts.Unseen,
		UnseenCritical: counts.UnseenCritical,
		Overdue:        counts.Overdue,
		Unassigned:     counts.Unassigned,
		Mine:           counts.Mine,
		MineOverdue:    counts.MineOverdue,
		ByKind:         byKind,
		SeenAt:         int(counts.SeenAt),
	}
}

func watchtowerHandOffModel(
	result *watchtowerports.HandOffWatchtowerItemResult,
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

func (r *MutationResolver) exceptionTarget(
	ctx context.Context,
	id string,
	expectedVersion *int,
) (*authctx.AuthContext, watchtowerports.ExceptionTarget, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceWatchtower, permission.OpUpdate)
	if err != nil {
		return nil, watchtowerports.ExceptionTarget{}, err
	}
	itemID, err := pulid.MustParse(id)
	if err != nil {
		return nil, watchtowerports.ExceptionTarget{}, err
	}

	target := watchtowerports.ExceptionTarget{TenantInfo: base.TenantInfo(authCtx), ItemID: itemID}
	if expectedVersion != nil {
		target.ExpectedVersion = int64(*expectedVersion)
	}

	return authCtx, target, nil
}

func optionalUser(ctx context.Context, id pulid.ID) (*tenant.User, error) {
	if id.IsNil() {
		return nil, nil //nolint:nilnil // no person behind the change is no user and no error
	}

	return base.OptionalMatch(loaders.UserByID(ctx, id.String()))
}

func optionalAgent(ctx context.Context, id pulid.ID) (*agentdefinition.Definition, error) {
	if id.IsNil() {
		return nil, nil //nolint:nilnil // no agent behind the work is no agent and no error
	}

	return base.OptionalMatch(loaders.AgentDefinitionByID(ctx, id.String()))
}

func suggestedAgent(
	ctx context.Context,
	item *watchtower.Item,
) (*agentdefinition.Definition, error) {
	if item.EventKind == "" || item.SubjectID.IsNil() {
		return nil, nil //nolint:nilnil // an item no agent can take has no suggestion
	}

	return loaders.SuggestedAgentByEventKind(ctx, string(item.EventKind))
}

func recordModel(item *watchtower.Item) *watchtower.RecordRef {
	kind := item.ResolvedRecordKind()
	if kind == "" && item.RecordLabel == "" {
		return nil
	}

	return &watchtower.RecordRef{Kind: kind, Label: item.RecordLabel}
}

func factsModel(facts []watchtower.Fact) []*gqlmodel.WatchtowerFact {
	out := make([]*gqlmodel.WatchtowerFact, 0, len(facts))
	for _, fact := range facts {
		out = append(out, &gqlmodel.WatchtowerFact{
			Label: fact.Label,
			Value: stringutils.NilIfEmpty(fact.Value),
			At:    intutils.PositiveIntPtr(fact.At),
		})
	}

	return out
}

func trailModel(trail []watchtower.TrailEvent) []*gqlmodel.WatchtowerTrailEvent {
	out := make([]*gqlmodel.WatchtowerTrailEvent, 0, len(trail))
	for _, event := range trail {
		out = append(out, &gqlmodel.WatchtowerTrailEvent{
			At:   intutils.PositiveIntPtr(event.At),
			Text: event.Text,
		})
	}

	return out
}
