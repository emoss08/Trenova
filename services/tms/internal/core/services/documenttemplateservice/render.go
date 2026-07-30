package documenttemplateservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/templateengine"
	"go.uber.org/zap"
)

// RenderDocument produces a printable artifact from the template governing the
// caller's kind and customer.
func (s *Service) RenderDocument(
	ctx context.Context,
	req *services.RenderDocumentRequest,
) (*services.RenderedDocument, error) {
	def, ok := s.registry.Get(req.Kind)
	if !ok {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"Template kind %q is not registered", req.Kind,
		))
	}

	if !def.HasChannel(documenttemplate.ChannelPDF) {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"Template kind %q does not produce a document", req.Kind,
		))
	}

	resolved, err := s.Resolve(ctx, &services.ResolveTemplateRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		CustomerID: req.CustomerID,
	})
	if err != nil {
		return nil, err
	}

	data, err := s.contextFor(ctx, req.Kind, req.Data, &services.BuildContextRequest{
		TenantInfo:  req.TenantInfo,
		Kind:        req.Kind,
		ReferenceID: req.ReferenceID,
		CustomerID:  req.CustomerID,
		UserID:      req.UserID,
	})
	if err != nil {
		return nil, err
	}

	// Content that reaches a live render already passed the publish gate, or is
	// a built-in the tests cover. Re-judging it strictly here would let a rule
	// added after publication break an invoice mid-billing-run.
	rendered, diags, err := s.renderDocumentHTML(ctx, def, resolved, data, req.Title,
		compileOptions{CacheKeyPrefix: s.cacheKeyFor(resolved)})
	if err != nil {
		return nil, err
	}

	out := &services.RenderedDocument{
		HTML:        rendered.body,
		Source:      resolved.Source,
		TemplateID:  resolved.TemplateID,
		VersionID:   resolved.VersionID,
		ContentHash: resolved.ContentHash,
		Warnings:    warningsFrom(diags),
	}

	if req.SkipPDF {
		return out, nil
	}

	pdf, err := s.printPDF(ctx, resolved, rendered, req.Title, req.TraceID)
	if err != nil {
		return nil, err
	}
	out.PDF = pdf

	return out, nil
}

// renderedDocument is the HTML side of a print: the wrapped body plus the
// separately-staged running header and footer.
type renderedDocument struct {
	body   string
	header string
	footer string
}

func (s *Service) renderDocumentHTML(
	ctx context.Context,
	def *documenttemplate.KindDefinition,
	resolved *services.ResolvedTemplate,
	data any,
	title string,
	opts compileOptions,
) (*renderedDocument, templateengine.Diagnostics, error) {
	compiled := s.compile(def, &resolved.Content, opts)

	if multiErr := diagnosticsToMultiError(compiled.diags); multiErr != nil {
		return nil, nil, multiErr
	}

	body, err := s.engine.Render(ctx, compiled.compiledFor(documenttemplate.ChannelPDF), data)
	if err != nil {
		return nil, nil, fmt.Errorf("render %s document body: %w", def.Kind, err)
	}

	document := templateengine.WrapDocument(templateengine.WrapRequest{
		Title:    title,
		BodyHTML: body,
		CSS:      resolved.Content.CSSContent,
	})

	diags := compiled.diags

	// Images are resolved after rendering, not before: the source may build a
	// src from context, and only the rendered markup knows the final URL.
	inlined, err := s.inliner.InlineAssets(ctx, &services.AssetInlineRequest{
		HTML:  document,
		Field: documenttemplate.ChannelPDF.Field(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("inline %s document assets: %w", def.Kind, err)
	}
	document = inlined.HTML
	diags = append(diags, inlined.Diags...)

	header, err := s.renderRunningPart(
		ctx, resolved.Content.HeaderHTML, def, data, "headerHtml",
	)
	if err != nil {
		return nil, nil, err
	}

	footer, err := s.renderRunningPart(
		ctx, resolved.Content.FooterHTML, def, data, "footerHtml",
	)
	if err != nil {
		return nil, nil, err
	}

	return &renderedDocument{body: document, header: header, footer: footer}, diags, nil
}

// renderRunningPart renders a header or footer.
//
// The renderer stages these as their own documents and substitutes page numbers
// into them, so they are wrapped separately and never carry the body's
// stylesheet — a header that inherited the body's page rules would repeat the
// body's margins on every page.
func (s *Service) renderRunningPart(
	ctx context.Context,
	source string,
	def *documenttemplate.KindDefinition,
	data any,
	field string,
) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", nil
	}

	compiled, diags := s.engine.Parse(&templateengine.ParseRequest{
		Field:        field,
		Kind:         string(def.Kind),
		Channel:      documenttemplate.ChannelPDF.Engine(),
		Source:       source,
		AllowedPaths: s.registry.Paths(def.Kind),
	})
	if multiErr := diagnosticsToMultiError(diags); multiErr != nil {
		return "", multiErr
	}

	rendered, err := s.engine.Render(ctx, compiled, data)
	if err != nil {
		return "", fmt.Errorf("render %s %s: %w", def.Kind, field, err)
	}

	return templateengine.WrapDocument(templateengine.WrapRequest{BodyHTML: rendered}), nil
}

func (s *Service) printPDF(
	ctx context.Context,
	resolved *services.ResolvedTemplate,
	rendered *renderedDocument,
	title string,
	traceID string,
) ([]byte, error) {
	if !s.pdf.Enabled() {
		return nil, services.ErrPDFRendererUnavailable
	}

	pdf, err := s.pdf.Render(ctx, &services.PDFRenderRequest{
		HTML:        rendered.body,
		HeaderHTML:  rendered.header,
		FooterHTML:  rendered.footer,
		Title:       title,
		TraceID:     traceID,
		PageSize:    resolved.Content.PageSize,
		Orientation: resolved.Content.Orientation,
		Margins:     resolved.Content.Margins,
	})
	if err != nil {
		return nil, err
	}

	return pdf, nil
}

// RenderMessage produces the subject, HTML, and text of one outbound message.
//
// Every channel the kind declares is rendered, so a caller never has to know
// which of them a given kind uses.
func (s *Service) RenderMessage(
	ctx context.Context,
	req *services.RenderMessageRequest,
) (*services.RenderedMessage, error) {
	def, ok := s.registry.Get(req.Kind)
	if !ok {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"Template kind %q is not registered", req.Kind,
		))
	}

	resolved, err := s.Resolve(ctx, &services.ResolveTemplateRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		CustomerID: req.CustomerID,
	})
	if err != nil {
		return nil, err
	}

	data, err := s.contextFor(ctx, req.Kind, req.Data, &services.BuildContextRequest{
		TenantInfo:  req.TenantInfo,
		Kind:        req.Kind,
		ReferenceID: req.ReferenceID,
		CustomerID:  req.CustomerID,
		UserID:      req.UserID,
	})
	if err != nil {
		return nil, err
	}

	message, err := s.renderMessageWith(ctx, def, resolved, data)
	if err == nil || !req.FallbackToBuiltIn || resolved.FromBuiltIn() {
		return message, err
	}

	// A notification that renders as nothing is worse than one that renders
	// unstyled, so the caller can ask for the built-in rather than the failure.
	// An invoice email does not: it leaves this false and the error surfaces.
	s.l.Warn("falling back to the built-in template after a render failure",
		zap.String("kind", string(req.Kind)),
		zap.String("source", resolved.Source.String()),
		zap.Error(err),
	)

	fallback, builtInErr := s.builtIn(req.Kind)
	if builtInErr != nil {
		return nil, err
	}

	return s.renderMessageWith(ctx, def, fallback, data)
}

func (s *Service) renderMessageWith(
	ctx context.Context,
	def *documenttemplate.KindDefinition,
	resolved *services.ResolvedTemplate,
	data any,
) (*services.RenderedMessage, error) {
	compiled := s.compile(def, &resolved.Content, compileOptions{
		CacheKeyPrefix: s.cacheKeyFor(resolved),
	})

	if multiErr := diagnosticsToMultiError(compiled.diags); multiErr != nil {
		return nil, multiErr
	}

	out := &services.RenderedMessage{
		Source:      resolved.Source,
		TemplateID:  resolved.TemplateID,
		VersionID:   resolved.VersionID,
		ContentHash: resolved.ContentHash,
		Warnings:    warningsFrom(compiled.diags),
	}

	for _, channel := range def.Channels {
		target := compiled.compiledFor(channel)
		if target == nil {
			continue
		}

		rendered, err := s.engine.Render(ctx, target, data)
		if err != nil {
			return nil, fmt.Errorf("render %s %s: %w", def.Kind, channel, err)
		}

		switch channel {
		case documenttemplate.ChannelSubject, documenttemplate.ChannelNotificationTitle:
			// A subject travels in a mail header, where a newline would split
			// the header and let the rest be read as new headers.
			out.Subject = sanitizeHeaderLine(rendered)
		case documenttemplate.ChannelEmailHTML:
			out.HTML = rendered
		case documenttemplate.ChannelEmailText, documenttemplate.ChannelNotificationBody:
			out.Text = rendered
		case documenttemplate.ChannelPDF:
			// A kind with both a document and a message renders its document
			// through RenderDocument, which owns the print path.
			continue
		}
	}

	if out.HTML != "" {
		inlined, err := s.inliner.InlineAssets(ctx, &services.AssetInlineRequest{
			HTML:  out.HTML,
			Field: documenttemplate.ChannelEmailHTML.Field(),
		})
		if err != nil {
			return nil, fmt.Errorf("inline %s message assets: %w", def.Kind, err)
		}
		out.HTML = inlined.HTML
		out.Warnings = append(out.Warnings, warningsFrom(inlined.Diags)...)
	}

	return out, nil
}

// PreviewVersion renders whatever the editor is showing, saved or not.
//
// Diagnostics come back alongside the output rather than instead of it: a draft
// is allowed to be wrong, and an author fixing it needs to see the render and
// the complaint at the same time.
func (s *Service) PreviewVersion(
	ctx context.Context,
	req *services.PreviewVersionRequest,
) (*services.PreviewResult, error) {
	resolved, err := s.previewSource(ctx, req)
	if err != nil {
		return nil, err
	}

	def, ok := s.registry.Get(resolved.Kind)
	if !ok {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"Template kind %q is not registered", resolved.Kind,
		))
	}

	data := req.Data
	if data == nil {
		// Before a real entity is chosen there is only the sample, which is also
		// the data the publish gate judges against — so a clean preview and a
		// publishable version mean the same thing.
		data, err = s.registry.SampleContext(resolved.Kind)
		if err != nil {
			return nil, err
		}
	}

	// A draft changes under a stable id, so previews are never cached by key.
	compiled := s.compile(def, &resolved.Content, compileOptions{})

	result := &services.PreviewResult{
		Source:      resolved.Source,
		ContentHash: resolved.ContentHash,
		Diagnostics: compiled.diags.Sorted(),
	}

	if compiled.diags.HasErrors() {
		return result, nil
	}

	if err = s.fillPreviewChannels(ctx, def, resolved, compiled, data, result); err != nil {
		return nil, err
	}

	if !req.IncludePDF || !def.HasChannel(documenttemplate.ChannelPDF) {
		return result, nil
	}

	// A draft changes under a stable id, so the print tier is not cached either.
	rendered, diags, err := s.renderDocumentHTML(
		ctx, def, resolved, data, def.DisplayName, compileOptions{},
	)
	if err != nil {
		return nil, err
	}
	result.HTML = rendered.body
	// The inliner's findings only exist once assets have been resolved, which
	// the HTML tier skips — a dropped logo must still reach the author.
	result.Diagnostics = diags.Sorted()

	pdf, err := s.printPDF(ctx, resolved, rendered, def.DisplayName, "")
	if err != nil {
		return nil, err
	}
	result.PDF = pdf

	return result, nil
}

func (s *Service) fillPreviewChannels(
	ctx context.Context,
	def *documenttemplate.KindDefinition,
	resolved *services.ResolvedTemplate,
	compiled *compiledVersion,
	data any,
	result *services.PreviewResult,
) error {
	for _, channel := range def.Channels {
		target := compiled.compiledFor(channel)
		if target == nil {
			continue
		}

		rendered, err := s.engine.Render(ctx, target, data)
		if err != nil {
			return fmt.Errorf("preview %s %s: %w", def.Kind, channel, err)
		}

		switch channel {
		case documenttemplate.ChannelSubject, documenttemplate.ChannelNotificationTitle:
			result.Subject = sanitizeHeaderLine(rendered)
		case documenttemplate.ChannelEmailText, documenttemplate.ChannelNotificationBody:
			result.Text = rendered
		case documenttemplate.ChannelEmailHTML:
			result.HTML = rendered
		case documenttemplate.ChannelPDF:
			// The document tier wraps and inlines; doing it here too would
			// double-wrap the body.
			result.HTML = templateengine.WrapDocument(templateengine.WrapRequest{
				Title:    def.DisplayName,
				BodyHTML: rendered,
				CSS:      resolved.Content.CSSContent,
			})
		}
	}

	return nil
}

// previewSource decides what is being previewed: unsaved editor content, a
// stored version, or the built-in.
func (s *Service) previewSource(
	ctx context.Context,
	req *services.PreviewVersionRequest,
) (*services.ResolvedTemplate, error) {
	if req.Content != nil {
		kind := req.Kind
		if kind == "" {
			return nil, errortypes.NewBusinessError(
				"A preview of unsaved content must name the template kind it belongs to",
			)
		}

		content := *req.Content
		applyPageDefaults(&content)

		return &services.ResolvedTemplate{
			Kind: kind,
			// Unsaved content belongs to no tier yet. Reporting the tier it
			// would land in would be a guess the author could act on wrongly.
			Source:      documenttemplate.SourceOrgDefault,
			Content:     content,
			ContentHash: ContentHash(&content),
		}, nil
	}

	if req.VersionID != nil && !req.VersionID.IsNil() {
		version, err := s.versionRepo.GetByID(
			ctx,
			&repositories.GetDocumentTemplateVersionByIDRequest{
				VersionID:  *req.VersionID,
				TenantInfo: req.TenantInfo,
			},
		)
		if err != nil {
			return nil, err
		}

		kind := req.Kind
		if version.Template != nil {
			kind = version.Template.Kind
		}

		content := contentFromVersion(version)
		source := documenttemplate.SourceOrgDefault
		if version.Template != nil && !version.Template.IsOrgDefault {
			source = documenttemplate.SourceAssigned
		}

		return &services.ResolvedTemplate{
			Kind:        kind,
			Source:      source,
			Content:     content,
			TemplateID:  &version.TemplateID,
			VersionID:   &version.ID,
			ContentHash: contentHashOr(version.ContentHash, &content),
		}, nil
	}

	if req.Kind == "" {
		return nil, errortypes.NewBusinessError(
			"A preview needs a version, unsaved content, or a template kind",
		)
	}

	return s.builtIn(req.Kind)
}

// applyPageDefaults fills the page box a draft has not chosen, so a preview
// never asks the renderer for a page with no area.
func applyPageDefaults(content *services.TemplateContent) {
	if !content.PageSize.IsValid() {
		content.PageSize = documenttemplate.PageSizeLetter
	}
	if !content.Orientation.IsValid() {
		content.Orientation = documenttemplate.OrientationPortrait
	}
	if content.Margins == (documenttemplate.Margins{}) {
		content.Margins = documenttemplate.DefaultMargins()
	}
}

// cacheKeyFor reuses a compilation only for content that cannot change under
// its own identity: a published version and a built-in are both immutable.
func (s *Service) cacheKeyFor(resolved *services.ResolvedTemplate) string {
	if resolved.ContentHash == "" {
		return ""
	}
	return string(resolved.Kind) + ":" + resolved.ContentHash
}

func (s *Service) contextFor(
	ctx context.Context,
	kind documenttemplate.Kind,
	supplied any,
	req *services.BuildContextRequest,
) (any, error) {
	if supplied != nil {
		return supplied, nil
	}
	return s.buildContext(ctx, kind, req)
}

// sanitizeHeaderLine collapses a rendered subject to one line.
//
// A subject is interpolated into a mail header, where an embedded newline ends
// the header and lets whatever follows be parsed as new headers — a template
// author must not be able to inject one through a customer name.
func sanitizeHeaderLine(value string) string {
	if !strings.ContainsAny(value, "\r\n") {
		return strings.TrimSpace(value)
	}

	replacer := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ")

	return strings.TrimSpace(strings.Join(strings.Fields(replacer.Replace(value)), " "))
}
