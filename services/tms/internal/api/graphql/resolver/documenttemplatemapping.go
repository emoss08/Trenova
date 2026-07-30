package resolver

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/pulid"
)

func documentTemplateColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.DocumentTemplateSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func documentTemplateAssignmentColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.DocumentTemplateAssignmentSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func documentTemplateConnectionToModel(
	result *pagination.CursorListResult[*documenttemplate.DocumentTemplate],
) (*gqlmodel.DocumentTemplateConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(
			node *documenttemplate.DocumentTemplate,
			cursor string,
		) *gqlmodel.DocumentTemplateEdge {
			return &gqlmodel.DocumentTemplateEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.DocumentTemplateEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DocumentTemplateConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func documentTemplateAssignmentConnectionToModel(
	result *pagination.CursorListResult[*documenttemplate.DocumentTemplateAssignment],
) (*gqlmodel.DocumentTemplateAssignmentConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(
			node *documenttemplate.DocumentTemplateAssignment,
			cursor string,
		) *gqlmodel.DocumentTemplateAssignmentEdge {
			return &gqlmodel.DocumentTemplateAssignmentEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.DocumentTemplateAssignmentEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DocumentTemplateAssignmentConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

// documentTemplateVariableToModel flattens the registry's nested catalog for the
// editor's reference panel and autocomplete, preserving the nesting so a
// collection's element shape stays discoverable.
func documentTemplateVariableToModel(
	variable *documenttemplate.VariableDefinition,
) *gqlmodel.DocumentTemplateVariable {
	out := &gqlmodel.DocumentTemplateVariable{
		Path:        variable.Path,
		Type:        variable.Type,
		Description: variable.Description,
		Required:    variable.Required,
	}

	if len(variable.Fields) == 0 {
		return out
	}

	out.Fields = make([]*gqlmodel.DocumentTemplateVariable, 0, len(variable.Fields))
	for i := range variable.Fields {
		out.Fields = append(out.Fields, documentTemplateVariableToModel(&variable.Fields[i]))
	}

	return out
}

func documentTemplateKindToModel(
	def *documenttemplate.KindDefinition,
	requiredPaths []string,
) *gqlmodel.DocumentTemplateKind {
	variables := make([]*gqlmodel.DocumentTemplateVariable, 0, len(def.Variables))
	for i := range def.Variables {
		variables = append(variables, documentTemplateVariableToModel(&def.Variables[i]))
	}

	channels := make([]documenttemplate.Channel, 0, len(def.Channels))
	channels = append(channels, def.Channels...)

	return &gqlmodel.DocumentTemplateKind{
		Kind:           string(def.Kind),
		DisplayName:    def.DisplayName,
		Description:    def.Description,
		Category:       def.Category,
		Channels:       channels,
		Paged:          def.Paged,
		CustomerScoped: def.CustomerScoped,
		Variables:      variables,
		RequiredPaths:  requiredPaths,
		// Source defaults to the built-in and is raised to OrgDefault only when a
		// template of this kind is actually live, so the catalog can say "system
		// default in effect" truthfully rather than by absence of a row.
		Source: documenttemplate.SourceBuiltIn,
	}
}

func documentTemplateDiagnosticsToModel(
	diags templateengine.Diagnostics,
) []*gqlmodel.DocumentTemplateDiagnostic {
	out := make([]*gqlmodel.DocumentTemplateDiagnostic, 0, len(diags))
	for _, diag := range diags {
		out = append(out, &gqlmodel.DocumentTemplateDiagnostic{
			Severity: diag.Severity.String(),
			Code:     diag.Code,
			Field:    diag.Field,
			Message:  diag.Message,
			Line:     diag.Line,
			Column:   diag.Column,
		})
	}

	return out
}

func documentTemplatePreviewToModel(
	result *services.PreviewResult,
	versionID *string,
) *gqlmodel.DocumentTemplatePreview {
	out := &gqlmodel.DocumentTemplatePreview{
		Subject: result.Subject,
		// HTML travels as a string. Serving it as a document on this origin would
		// let an organization's template read app-origin cookies; the client puts
		// it in a sandboxed iframe instead.
		HTML:        result.HTML,
		Text:        result.Text,
		Source:      result.Source,
		ContentHash: result.ContentHash,
		Diagnostics: documentTemplateDiagnosticsToModel(result.Diagnostics),
	}

	// The bytes are fetched over REST. Only a saved version has a stable URL —
	// unsaved editor content has no id to address, so the client saves the draft
	// before asking for a print.
	if versionID != nil && *versionID != "" {
		url := fmt.Sprintf("/api/v1/document-templates/versions/%s/preview.pdf", *versionID)
		out.PDFURL = &url
	}

	return out
}

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
		Subject:    templateString(input.Subject),
		BodyHTML:   templateString(input.BodyHTML),
		BodyText:   templateString(input.BodyText),
		CSSContent: templateString(input.CSSContent),
		HeaderHTML: templateString(input.HeaderHTML),
		FooterHTML: templateString(input.FooterHTML),
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

// applyDocumentTemplateVersionInput copies editor input onto a version entity.
func applyDocumentTemplateVersionInput(
	version *documenttemplate.DocumentTemplateVersion,
	input *gqlmodel.DocumentTemplateVersionInput,
) {
	if input == nil {
		return
	}

	version.Subject = templateString(input.Subject)
	version.BodyHTML = templateString(input.BodyHTML)
	version.BodyText = templateString(input.BodyText)
	version.CSSContent = templateString(input.CSSContent)
	version.HeaderHTML = templateString(input.HeaderHTML)
	version.FooterHTML = templateString(input.FooterHTML)

	if input.PageSize != nil {
		version.PageSize = *input.PageSize
	}
	if input.Orientation != nil {
		version.Orientation = *input.Orientation
	}

	version.MarginTop = input.MarginTop
	version.MarginBottom = input.MarginBottom
	version.MarginLeft = input.MarginLeft
	version.MarginRight = input.MarginRight
}

func templateString(value *string) string {
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
func (r *Resolver) previewFor(
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

	return r.documentTemplateService.PreviewVersion(ctx, &services.PreviewVersionRequest{
		TenantInfo: tenantInfo(authCtx),
		Kind:       documenttemplate.Kind(input.Kind),
		VersionID:  versionID,
		Content:    documentTemplateContentFromInput(input.Content),
	})
}
