package resolvercore

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/agentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
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

	ids := make([]string, len(field.Choices))
	for idx := range field.Choices {
		ids[idx] = field.Choices[idx].ID
	}

	return r.recordLabels(ctx, permission.Resource(field.Resource), ids)
}

// SubjectLabel is the name of the record a run is on, read through the
// request's loader so a page of runs is named together. It is "" for a run
// on the organization or a conversation, for a kind of record that has no
// label, and for a reader who may not read the record. The subject's kind and
// the resource that governs it come from the record-kind registry.
func (r *Core) SubjectLabel(
	ctx context.Context,
	subjectType agent.SubjectType,
	subjectID pulid.ID,
) string {
	if subjectType == agent.SubjectOrganization || subjectType == agent.SubjectAssistantThread ||
		subjectID.IsNil() {
		return ""
	}
	kind, ok := subjectType.RecordKind()
	if !ok {
		return ""
	}
	resource := kind.Resource()

	return r.recordLabels(ctx, resource, []string{subjectID.String()}).Label(resource, subjectID)
}

// recordLabels reads the labels of records of one resource for a reader who
// may read it. A read that fails names none: a label is never worth failing
// the page that shows it.
func (r *Core) recordLabels(
	ctx context.Context,
	resource permission.Resource,
	ids []string,
) agentruntimeports.RecordLabels {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil
	}
	if !r.HasPermission(ctx, authCtx, resource, permission.OpRead) {
		return nil
	}

	labels, err := agentloaders.SubsetLabels(ctx, agentloaders.SubsetLabelsKey(resource, ids))
	if errors.Is(err, loadercore.ErrNotAttached) {
		return nil
	}
	if err != nil {
		r.L.Warn("could not read the labels of records",
			zap.String("resource", resource.String()),
			zap.Error(err))

		return nil
	}

	return labels
}
