package resolvercore

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/agentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"go.uber.org/zap"
)

// subsetChoices lists the records a record-subset field offers, in the order
// proposed, each named by its label when the reader may read its resource
// and it has one, by its id otherwise. Every other kind lists none.
func (r *Core) SubsetChoices(
	ctx context.Context,
	field *toolschema.Field,
) []*toolschema.Choice {
	if field.Kind != toolschema.KindRecordSubset {
		return nil
	}

	labelled := *field
	labelled.Choices = slices.Clone(field.Choices)
	agentruntimeports.LabelSubsetChoices(&labelled, r.subsetLabels(ctx, field))

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
func (r *Core) subsetLabels(
	ctx context.Context,
	field *toolschema.Field,
) agentruntimeports.RecordLabels {
	if len(field.Choices) == 0 || field.Resource == "" {
		return nil
	}

	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil
	}
	resource := permission.Resource(field.Resource)
	if !r.HasPermission(ctx, authCtx, resource, permission.OpRead) {
		return nil
	}

	ids := make([]string, len(field.Choices))
	for idx := range field.Choices {
		ids[idx] = field.Choices[idx].ID
	}
	labels, err := agentloaders.SubsetLabels(ctx, agentloaders.SubsetLabelsKey(resource, ids))
	if errors.Is(err, loadercore.ErrNotAttached) {
		return nil
	}
	if err != nil {
		r.L.Warn("could not read the labels of the records a proposal offers",
			zap.String("resource", field.Resource),
			zap.Error(err))

		return nil
	}

	return labels
}
