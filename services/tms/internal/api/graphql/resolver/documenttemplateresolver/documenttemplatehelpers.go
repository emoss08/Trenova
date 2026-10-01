package documenttemplateresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// documentTemplateContentFromInput turns editor input into the service's content
// shape. A nil input previews the built-in, which is what the catalog shows for
// a kind nobody has customized.
func documentTemplateContentFromInput(
	input *gqlmodel.DocumentTemplateVersionInput,
) *services.TemplateContent {
	if input == nil {
		return nil
	}

	content := &services.TemplateContent{
		Subject:    TemplateString(input.Subject),
		BodyHTML:   TemplateString(input.BodyHTML),
		BodyText:   TemplateString(input.BodyText),
		CSSContent: TemplateString(input.CSSContent),
		HeaderHTML: TemplateString(input.HeaderHTML),
		FooterHTML: TemplateString(input.FooterHTML),
	}

	if input.PageSize != nil {
		content.PageSize = *input.PageSize
	}
	if input.Orientation != nil {
		content.Orientation = *input.Orientation
	}

	content.Margins = documenttemplate.Margins{
		Top:    templateFloat(input.MarginTop, content.Margins.Top),
		Bottom: templateFloat(input.MarginBottom, content.Margins.Bottom),
		Left:   templateFloat(input.MarginLeft, content.Margins.Left),
		Right:  templateFloat(input.MarginRight, content.Margins.Right),
	}

	return content
}

func (r *Deps) DocumentTemplateVersionKind(
	ctx context.Context,
	version *documenttemplate.DocumentTemplateVersion,
	tenant pagination.TenantInfo,
) documenttemplate.Kind {
	if version.Template != nil && version.Template.Kind != "" {
		return version.Template.Kind
	}

	l, ok := loaders.FromContext(ctx)
	if !ok || l == nil {
		return r.DocumentTemplateService.KindOf(ctx, version, tenant)
	}

	kind, err := l.DocumentTemplateKindByTemplateID.Load(ctx, version.TemplateID.String())
	if err != nil {
		return ""
	}

	return kind
}

func TemplateString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func templateFloat(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

// previewFor renders whatever the editor is showing: unsaved content, a stored
// version, or the built-in.
//
// includePdf from the input is deliberately not honoured here. The bytes never
// travel over GraphQL, so printing during a keystroke-debounced query would burn
// a conversion nobody reads; the REST route owns the print tier.
func (r *Deps) PreviewFor(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	input *gqlmodel.DocumentTemplatePreviewInput,
) (*services.PreviewResult, error) {
	var versionID *pulid.ID
	if input.VersionID != nil && *input.VersionID != "" {
		parsed, err := pulid.MustParse(*input.VersionID)
		if err != nil {
			return nil, err
		}
		versionID = &parsed
	}

	return r.DocumentTemplateService.PreviewVersion(ctx, &services.PreviewVersionRequest{
		TenantInfo: base.TenantInfo(authCtx),
		Kind:       documenttemplate.Kind(input.Kind),
		VersionID:  versionID,
		Content:    documentTemplateContentFromInput(input.Content),
	})
}
