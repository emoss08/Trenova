package services

import (
	"context"
	"html/template"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/pulid"
)

// ContextBuilderGroup is the fx group every domain drops its context builder
// into.
//
// The template service depends on the group, never on the domains: an invoice
// knows how to describe itself and the template service knows how to render, and
// neither has to import the other. Adding a kind is adding a builder.
const ContextBuilderGroup = `group:"documenttemplate.contextbuilders"`

// TemplateContent is one version's renderable source, whichever tier it came
// from.
//
// A resolved database version and an embedded starter both land in this shape,
// which is what keeps a single render path: there is no branch anywhere that
// says "if no template then use the hardcoded string".
type TemplateContent struct {
	Subject    string
	BodyHTML   string
	BodyText   string
	CSSContent string
	HeaderHTML string
	FooterHTML string

	PageSize    documenttemplate.PageSize
	Orientation documenttemplate.Orientation
	Margins     documenttemplate.Margins
}

// Channel returns the source stored for one channel.
func (c *TemplateContent) Channel(channel documenttemplate.Channel) string {
	switch channel {
	case documenttemplate.ChannelPDF, documenttemplate.ChannelEmailHTML:
		return c.BodyHTML
	case documenttemplate.ChannelEmailText, documenttemplate.ChannelNotificationBody:
		return c.BodyText
	case documenttemplate.ChannelSubject, documenttemplate.ChannelNotificationTitle:
		return c.Subject
	default:
		return ""
	}
}

// ResolvedTemplate is the outcome of walking assigned → org default → built-in.
type ResolvedTemplate struct {
	Kind    documenttemplate.Kind
	Source  documenttemplate.TemplateSource
	Content TemplateContent

	// TemplateID and VersionID are nil for the built-in, which has no row. A
	// generated document records that as "rendered from the shipped default".
	TemplateID *pulid.ID
	VersionID  *pulid.ID

	// ContentHash identifies the bytes that will render, for both tiers. It is
	// what makes a preview cacheable and a past render reproducible.
	ContentHash string
}

// FromBuiltIn reports whether the embedded starter produced this.
func (r *ResolvedTemplate) FromBuiltIn() bool {
	return r.Source == documenttemplate.SourceBuiltIn
}

type ResolveTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       documenttemplate.Kind

	// CustomerID is optional. A kind that is not CustomerScoped ignores it
	// rather than skipping the assignment tier by accident.
	CustomerID *pulid.ID
}

// BuildContextRequest is what a domain needs to assemble its own context.
type BuildContextRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       documenttemplate.Kind

	// ReferenceID names the entity being rendered — the invoice, the detention
	// occurrence, the report run.
	ReferenceID pulid.ID
	CustomerID  *pulid.ID

	// UserID is the person who asked, for the kinds whose context names them.
	UserID pulid.ID
}

// ContextBuilder turns a domain entity into the context its kind declares.
//
// The builder owns the mapping, so the variable catalog and the data that fills
// it are maintained in the same place as the domain they describe.
type ContextBuilder interface {
	Kind() documenttemplate.Kind
	Build(ctx context.Context, req *BuildContextRequest) (any, error)
}

type RenderDocumentRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       documenttemplate.Kind
	CustomerID *pulid.ID

	// Data is the already-built context. Callers that have the entity but not
	// the context pass ReferenceID instead and let the registered builder run.
	Data        any
	ReferenceID pulid.ID
	UserID      pulid.ID

	// Title becomes the PDF document title, which is what a reader's viewer tab
	// and a saved file are named.
	Title   string
	TraceID string

	// SkipPDF renders only the HTML, for the editor preview tier that runs on
	// every debounce and must not pay for a print.
	SkipPDF bool
}

type RenderedDocument struct {
	PDF  []byte
	HTML string

	Source      documenttemplate.TemplateSource
	TemplateID  *pulid.ID
	VersionID   *pulid.ID
	ContentHash string

	// Warnings records what the render dropped — a blocked image, an asset over
	// budget — so an omission reaches the generated document rather than
	// vanishing.
	Warnings []documenttemplate.GeneratedDocumentWarning
}

type RenderMessageRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       documenttemplate.Kind
	CustomerID *pulid.ID

	Data        any
	ReferenceID pulid.ID
	UserID      pulid.ID

	// FallbackToBuiltIn renders the shipped default when the organization's own
	// template fails.
	//
	// A notification sets this: a lost notification is worse than an unstyled
	// one. An invoice email does not — a customer must never receive a document
	// that silently differs from the one the organization authored, so the
	// failure surfaces to the sender instead.
	FallbackToBuiltIn bool
}

type RenderedMessage struct {
	Subject string
	HTML    string
	Text    string

	Source      documenttemplate.TemplateSource
	TemplateID  *pulid.ID
	VersionID   *pulid.ID
	ContentHash string

	Warnings []documenttemplate.GeneratedDocumentWarning
}

// PreviewVersionRequest previews content that may not be saved.
//
// Content wins over VersionID: the editor previews what is on screen, which by
// definition has not been written yet. With neither, the built-in is previewed,
// which is what the catalog shows for a kind nobody has customized.
type PreviewVersionRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       documenttemplate.Kind

	VersionID *pulid.ID
	Content   *TemplateContent

	// Data overrides the kind's sample context. Nil means preview against the
	// sample, which is the only data available before a real entity is chosen.
	Data any

	// IncludePDF gates the expensive tier. The editor drives it from a longer
	// debounce and an explicit button, not from every keystroke.
	IncludePDF bool
}

type PreviewResult struct {
	Subject string
	HTML    string
	Text    string
	PDF     []byte

	Source      documenttemplate.TemplateSource
	ContentHash string

	// Diagnostics are advisory here rather than blocking. A draft is allowed to
	// be wrong; the author needs to see why, not be refused a preview.
	Diagnostics templateengine.Diagnostics
}

// RecordGeneratedDocumentRequest logs one rendered artifact against the template
// version that produced it.
//
// This is what makes "what exactly did we send that customer in March"
// answerable after a template has been edited — which is precisely when an
// invoice or detention dispute asks. The version row cannot be deleted while a
// record points at it.
type RecordGeneratedDocumentRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       documenttemplate.Kind

	// Rendered carries the source, ids, and content hash from the render that
	// produced the bytes, so the caller does not restate them and cannot get
	// them wrong.
	Rendered *RenderedDocument

	// ReferenceType and ReferenceID name the business object the document is
	// about — the invoice, the detention occurrence.
	ReferenceType string
	ReferenceID   pulid.ID

	// DocumentID is the stored document the bytes were uploaded as, when the
	// caller uploaded them.
	DocumentID *pulid.ID

	FileName string
	FilePath string
	FileSize int64
	Checksum string

	UserID pulid.ID
}

// PublishVersionRequest is the gate a version passes to go live.
type PublishVersionRequest struct {
	TenantInfo pagination.TenantInfo
	VersionID  pulid.ID
	UserID     pulid.ID
	Notes      string
}

// DocumentTemplateResolver is what every render path in the system talks to.
type DocumentTemplateResolver interface {
	// Resolve reports which template governs a render and why, without
	// rendering anything.
	Resolve(ctx context.Context, req *ResolveTemplateRequest) (*ResolvedTemplate, error)

	RenderDocument(ctx context.Context, req *RenderDocumentRequest) (*RenderedDocument, error)

	RenderMessage(ctx context.Context, req *RenderMessageRequest) (*RenderedMessage, error)

	PreviewVersion(ctx context.Context, req *PreviewVersionRequest) (*PreviewResult, error)

	// RecordGeneratedDocument logs a rendered artifact against its template
	// version. Callers that persist or send a document call it after the bytes
	// are stored.
	RecordGeneratedDocument(
		ctx context.Context,
		req *RecordGeneratedDocumentRequest,
	) (*documenttemplate.GeneratedDocument, error)
}

// AssetInlineRequest is one rendered document whose images need resolving.
type AssetInlineRequest struct {
	// HTML is rendered output, not template source.
	HTML string
	// Field names the editor pane, so a diagnostic routes back to it.
	Field string
}

type AssetInlineResult struct {
	HTML  string
	Diags templateengine.Diagnostics
	// Inlined counts the assets that made it in; Bytes is their decoded size.
	Inlined int
	Bytes   int64
}

// AssetInliner resolves document image references to data: URIs.
//
// This is a port because resolving an organization-controlled URL is a network
// egress decision — the implementation routes through the SSRF-safe client — and
// the core has no business knowing how that is enforced.
type AssetInliner interface {
	InlineAssets(ctx context.Context, req *AssetInlineRequest) (*AssetInlineResult, error)

	// ResolveImageDataURI turns one image reference — a storage key, an https
	// URL, or an existing data: URI — into a validated data: URI.
	//
	// A context builder needs this because a logo arrives as a context *field*,
	// not as markup: by the time InlineAssets sees the rendered document the
	// value is already interpolated, and interpolating a raw organization URL
	// into a template.URL field would hand an organization the unescaped-URL
	// slot. Going through here means the field only ever carries base64 of bytes
	// this decoded and validated.
	//
	// A reference that cannot be resolved returns an empty string and no error:
	// a missing logo must not stop an invoice going out.
	ResolveImageDataURI(ctx context.Context, src string) (string, error)
}

// ResolveLogoDataURI turns an organization's configured logo reference into the
// template.URL a document context carries.
//
// It exists so exactly one place in the system types an inliner result as a
// template.URL. That typing is what makes a data: URI survive html/template at
// all, and it is only safe because the value is base64 of bytes the inliner
// decoded and validated — never text anyone typed. Every context that shows a
// logo goes through here rather than repeating that reasoning.
//
// A reference that cannot be resolved yields an empty string and no error: the
// template's {{ if }} drops the image, and no document is worth losing over a
// logo.
func ResolveLogoDataURI(
	ctx context.Context,
	inliner AssetInliner,
	src string,
) (template.URL, error) {
	if inliner == nil || src == "" {
		return "", nil
	}

	dataURI, err := inliner.ResolveImageDataURI(ctx, src)
	if err != nil {
		return "", err
	}

	//nolint:gosec // G203: base64 the inliner produced from bytes it decoded and
	// validated. See this function's doc comment.
	return template.URL(dataURI), nil
}
