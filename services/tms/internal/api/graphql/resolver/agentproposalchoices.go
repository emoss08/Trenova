package resolver

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/toolschema"
	"go.uber.org/zap"
)

// subsetChoices lists the records a record-subset field offers, in the order
// proposed, each named by its label when the reader may read its resource
// and it has one, by its id otherwise. Every other kind lists none.
func (r *Resolver) subsetChoices(
	ctx context.Context,
	field *toolschema.Field,
) []*toolschema.Choice {
	if field.Kind != toolschema.KindRecordSubset {
		return nil
	}

	labelled := *field
	labelled.Choices = slices.Clone(field.Choices)
	services.LabelSubsetChoices(&labelled, r.subsetLabels(ctx, field))

	out := make([]*toolschema.Choice, len(labelled.Choices))
	for idx := range labelled.Choices {
		out[idx] = &labelled.Choices[idx]
	}

	return out
}

// subsetLabels reads the labels of a field's records through the request's
// loader, so a page of proposals is labelled together. A reader who may not
// read the resource is given none, and a read that fails leaves the records
// named by their ids.
func (r *Resolver) subsetLabels(
	ctx context.Context,
	field *toolschema.Field,
) services.RecordLabels {
	if len(field.Choices) == 0 || field.Resource == "" {
		return nil
	}

	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil
	}
	resource := permission.Resource(field.Resource)
	if !r.hasPermission(ctx, authCtx, resource, permission.OpRead) {
		return nil
	}

	l, ok := loaders.FromContext(ctx)
	if !ok || l.SubsetLabels == nil {
		return nil
	}

	ids := make([]string, len(field.Choices))
	for idx := range field.Choices {
		ids[idx] = field.Choices[idx].ID
	}
	labels, err := l.SubsetLabels.Load(ctx, loaders.SubsetLabelsKey(resource, ids))
	if err != nil {
		r.l.Warn("could not read the labels of the records a proposal offers",
			zap.String("resource", field.Resource),
			zap.Error(err))

		return nil
	}

	return labels
}
