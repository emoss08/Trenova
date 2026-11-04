package paletteanswer

import (
	"context"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Permissions permissionports.PermissionEngine
	Converser   llmports.TaskConverser
	Nova        llmports.TaskRunner
	QueryTools  agenttoolports.AgentQueryToolRegistry
	Search      kernelports.GlobalSearchService
	Zones       agenttoolports.TenantTimezoneReader
}

// Service answers a question typed into the command palette: a short answer
// from a few of the person's own reads, streamed as it is written.
type Service struct {
	l           *zap.Logger
	permissions permissionports.PermissionEngine
	converser   llmports.TaskConverser
	nova        llmports.TaskRunner
	catalog     *catalog
	validator   *toolschema.Validator
	search      kernelports.GlobalSearchService
	zones       agenttoolports.TenantTimezoneReader
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.palette-answer"),
		permissions: p.Permissions,
		converser:   p.Converser,
		nova:        p.Nova,
		catalog:     newCatalog(p.QueryTools),
		validator:   toolschema.NewValidator(),
		search:      p.Search,
		zones:       p.Zones,
	}
}

// Available reports whether Nova answers palette questions for the tenant.
func (s *Service) Available(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	return s.nova.Available(ctx, tenantInfo)
}

// Answer reads what the question needs as the person asking and streams the
// answer to events.
func (s *Service) Answer(
	ctx context.Context,
	askID pulid.ID,
	question *paletteports.Question,
	actor *permissionports.RequestActor,
	events llmports.NovaEvents,
) (*llmports.NovaAnswer, error) {
	tenantInfo := actor.TenantInfo()
	reads := &reader{
		catalog:     s.catalog,
		validator:   s.validator,
		search:      s.search,
		permissions: s.permissions,
		actor:       actor,
		timezone:    s.timezone(ctx, tenantInfo),
	}

	return s.converser.Converse(ctx, &llmports.NovaConversation{
		Name:         aitask.TaskPaletteAnswer,
		Feature:      aiusage.FeaturePaletteAnswer,
		Instructions: instructions,
		Question:     strings.TrimSpace(question.Content),
		Context:      questionContext(question),
		Tools:        reads.tools(),
		MaxCalls:     maxCalls,
		MaxTokens:    maxTokens,
		TenantInfo:   tenantInfo,
		UserID:       actor.UserID,
		OwnerKind:    agentruntimeports.RunStepOwnerPaletteQuestion,
		OwnerID:      askID,
	}, events)
}

func (s *Service) timezone(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	zone, err := s.zones.TenantTimezone(ctx, tenantInfo)
	if err != nil {
		s.l.Warn("could not read the organization's timezone; using UTC", zap.Error(err))

		return ""
	}

	return zone
}

type pageSummary struct {
	Path   string                     `json:"path"`
	Record string                     `json:"record,omitempty"`
	ID     string                     `json:"id,omitempty"`
	Title  string                     `json:"title,omitempty"`
	View   *agentpagecontext.PageView `json:"view,omitempty"`
}

func questionContext(question *paletteports.Question) llmports.DelimitedContext {
	sections := make([]llmports.ContextSection, 0, 2)
	if page := question.Page.Normalized(); page != nil {
		if encoded, err := sonic.MarshalString(pageSummary{
			Path:   page.Path,
			Record: page.EntityType,
			ID:     page.EntityID,
			Title:  page.Title,
			View:   page.View,
		}); err == nil {
			sections = append(sections, llmports.ContextSection{
				Title: "The page under the palette", Content: encoded,
			})
		}
	}
	if mentions := agentpagecontext.NormalizeEntityRefs(question.Mentions); len(mentions) > 0 {
		if encoded, err := sonic.MarshalString(mentions); err == nil {
			sections = append(sections, llmports.ContextSection{
				Title: "Mentioned records", Content: encoded,
			})
		}
	}

	return llmports.DelimitedContext{Sections: sections}
}
