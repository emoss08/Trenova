package documenttemplateservice

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/templateengine"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Repo           repositories.DocumentTemplateRepository
	VersionRepo    repositories.DocumentTemplateVersionRepository
	AssignmentRepo repositories.DocumentTemplateAssignmentRepository
	GeneratedRepo  repositories.GeneratedDocumentRepository
	Registry       *documenttemplate.Registry
	Engine         *templateengine.Engine
	Inliner        services.AssetInliner
	PDFRenderer    services.PDFRenderer
	Validator      *Validator
	AuditService   services.AuditService

	ContextBuilders []services.ContextBuilder `group:"documenttemplate.contextbuilders"`
}

type Service struct {
	l              *zap.Logger
	repo           repositories.DocumentTemplateRepository
	versionRepo    repositories.DocumentTemplateVersionRepository
	assignmentRepo repositories.DocumentTemplateAssignmentRepository
	generatedRepo  repositories.GeneratedDocumentRepository
	registry       *documenttemplate.Registry
	engine         *templateengine.Engine
	inliner        services.AssetInliner
	pdf            services.PDFRenderer
	validator      *Validator
	auditService   services.AuditService

	builders map[documenttemplate.Kind]services.ContextBuilder
}

var _ services.DocumentTemplateResolver = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	builders := make(map[documenttemplate.Kind]services.ContextBuilder, len(p.ContextBuilders))
	for _, builder := range p.ContextBuilders {
		if builder == nil {
			continue
		}
		builders[builder.Kind()] = builder
	}

	return &Service{
		l:              p.Logger.Named("service.document-template"),
		repo:           p.Repo,
		versionRepo:    p.VersionRepo,
		assignmentRepo: p.AssignmentRepo,
		generatedRepo:  p.GeneratedRepo,
		registry:       p.Registry,
		engine:         p.Engine,
		inliner:        p.Inliner,
		pdf:            p.PDFRenderer,
		validator:      p.Validator,
		auditService:   p.AuditService,
		builders:       builders,
	}
}

// Registry exposes the kind catalog, which the admin variable picker reads and
// the GraphQL layer publishes.
func (s *Service) Registry() *documenttemplate.Registry { return s.registry }

// Resolve walks assigned → organization default → built-in.
//
// The built-in is a real tier, not a fallback branch: it produces the same
// ResolvedTemplate shape as a database row, so every caller downstream has one
// render path and no caller ever holds a hardcoded string.
func (s *Service) Resolve(
	ctx context.Context,
	req *services.ResolveTemplateRequest,
) (*services.ResolvedTemplate, error) {
	def, ok := s.registry.Get(req.Kind)
	if !ok {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"Template kind %q is not registered", req.Kind,
		))
	}

	// A customer only participates when the kind says a per-customer override
	// means something. Passing one to a scheduled report would look like it
	// worked and never fire.
	customerID := req.CustomerID
	if !def.CustomerScoped {
		customerID = nil
	}

	row, err := s.repo.Resolve(ctx, &repositories.ResolveDocumentTemplateRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, err
	}

	if row == nil {
		return s.builtIn(req.Kind)
	}

	content := contentFromVersion(row.Version)

	return &services.ResolvedTemplate{
		Kind:        req.Kind,
		Source:      row.Source,
		Content:     content,
		TemplateID:  &row.Template.ID,
		VersionID:   &row.Version.ID,
		ContentHash: contentHashOr(row.Version.ContentHash, &content),
	}, nil
}

// builtIn loads the embedded starter for a kind.
func (s *Service) builtIn(kind documenttemplate.Kind) (*services.ResolvedTemplate, error) {
	starter, err := starters.For(kind)
	if err != nil {
		// A missing starter is a packaging fault, not an organization's problem,
		// but it must not render a blank page to a customer either.
		return nil, fmt.Errorf("load built-in template for %s: %w", kind, err)
	}

	content := services.TemplateContent{
		Subject:     starter.Subject,
		BodyHTML:    starter.BodyHTML,
		BodyText:    starter.BodyText,
		CSSContent:  starter.CSS,
		PageSize:    starter.PageSize,
		Orientation: starter.Orientation,
		Margins:     starter.Margins,
	}

	return &services.ResolvedTemplate{
		Kind:        kind,
		Source:      documenttemplate.SourceBuiltIn,
		Content:     content,
		ContentHash: starter.ContentHash(),
	}, nil
}

func contentFromVersion(
	version *documenttemplate.DocumentTemplateVersion,
) services.TemplateContent {
	return services.TemplateContent{
		Subject:     version.Subject,
		BodyHTML:    version.BodyHTML,
		BodyText:    version.BodyText,
		CSSContent:  version.CSSContent,
		HeaderHTML:  version.HeaderHTML,
		FooterHTML:  version.FooterHTML,
		PageSize:    version.ResolvedPageSize(),
		Orientation: version.ResolvedOrientation(),
		Margins:     version.Margins(),
	}
}

// ContentHash identifies a version's renderable bytes.
//
// The separator means moving text from the body to the subject changes the hash
// rather than producing the same concatenation, which matters because the hash
// is what a preview cache and a generated document both key on.
func ContentHash(content *services.TemplateContent) string {
	return templateengine.ContentHash(strings.Join([]string{
		content.Subject,
		content.BodyHTML,
		content.BodyText,
		content.CSSContent,
		content.HeaderHTML,
		content.FooterHTML,
	}, "\x00"))
}

func contentHashOr(stored string, content *services.TemplateContent) string {
	if strings.TrimSpace(stored) != "" {
		return stored
	}
	return ContentHash(content)
}

// compiledVersion is every channel of one version, compiled together.
//
// Compiling as a unit is what makes the required-path check possible: a subject
// line and a body each reference part of the catalog, and only their union can
// be judged against what the kind demands.
type compiledVersion struct {
	compiled map[documenttemplate.Channel]*templateengine.Compiled
	paths    []string
	diags    templateengine.Diagnostics
}

func (c *compiledVersion) compiledFor(
	channel documenttemplate.Channel,
) *templateengine.Compiled {
	return c.compiled[channel]
}

// compileOptions vary between drafting and publishing.
type compileOptions struct {
	// Strict turns advisory findings into blocking ones. False while an author
	// is typing, true at the gate.
	Strict bool
	// CacheKeyPrefix enables compiled reuse. Published versions are immutable so
	// their id is a safe key; a draft changes under the same id, so the editor
	// passes nothing.
	CacheKeyPrefix string
}

// compile parses every channel a kind declares and collects the union of paths.
func (s *Service) compile(
	def *documenttemplate.KindDefinition,
	content *services.TemplateContent,
	opts compileOptions,
) *compiledVersion {
	result := &compiledVersion{
		compiled: make(map[documenttemplate.Channel]*templateengine.Compiled, len(def.Channels)),
	}

	allowed := s.registry.Paths(def.Kind)
	seen := make(map[string]struct{}, len(allowed))

	for _, channel := range def.Channels {
		source := content.Channel(channel)
		if strings.TrimSpace(source) == "" {
			continue
		}

		cacheKey := ""
		if opts.CacheKeyPrefix != "" {
			cacheKey = opts.CacheKeyPrefix + ":" + string(channel)
		}

		compiled, diags := s.engine.Parse(&templateengine.ParseRequest{
			Field:        channel.Field(),
			Kind:         string(def.Kind),
			Channel:      channel.Engine(),
			Source:       source,
			AllowedPaths: allowed,
			Strict:       opts.Strict,
			CacheKey:     cacheKey,
		})

		result.diags = append(result.diags, diags...)
		if compiled == nil {
			continue
		}

		result.compiled[channel] = compiled
		for _, path := range compiled.Paths {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			result.paths = append(result.paths, path)
		}
	}

	sort.Strings(result.paths)

	return result
}

// diagnosticsToMultiError turns engine findings into the field errors a form can
// show, so a bad template reads as a validation failure rather than a 500.
func diagnosticsToMultiError(diags templateengine.Diagnostics) *errortypes.MultiError {
	errs := diags.Errors()
	if len(errs) == 0 {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	for _, diag := range errs.Sorted() {
		field := diag.Field
		if field == "" {
			field = "bodyHtml"
		}

		message := diag.Message
		if diag.Line > 0 {
			message = fmt.Sprintf("Line %d: %s", diag.Line, message)
		}

		multiErr.Add(field, errortypes.ErrInvalid, message)
	}

	return multiErr
}

// warningsFrom converts advisory diagnostics into the record a generated
// document carries, so an author can see later what a render quietly dropped.
func warningsFrom(diags templateengine.Diagnostics) []documenttemplate.GeneratedDocumentWarning {
	warnings := diags.Warnings()
	if len(warnings) == 0 {
		return nil
	}

	out := make([]documenttemplate.GeneratedDocumentWarning, 0, len(warnings))
	for _, diag := range warnings {
		out = append(out, documenttemplate.GeneratedDocumentWarning{
			Code:    diag.Code,
			Field:   diag.Field,
			Message: diag.Message,
		})
	}

	return out
}

// buildContext asks the kind's registered builder for its data.
//
// A kind with no builder is a wiring mistake rather than a runtime condition:
// the render was asked for by a caller that should have supplied data.
func (s *Service) buildContext(
	ctx context.Context,
	kind documenttemplate.Kind,
	req *services.BuildContextRequest,
) (any, error) {
	builder, ok := s.builders[kind]
	if !ok {
		return nil, fmt.Errorf("no context builder is registered for template kind %s", kind)
	}

	return builder.Build(ctx, req)
}
