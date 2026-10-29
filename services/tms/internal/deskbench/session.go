package deskbench

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/deskbench/benchcell"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultUserEmail = "admin@trenova.app"
)

var defaultAgentTemplates = []agenttemplate.Template{
	agenttemplate.TemplateGeneralAssistant,
	agenttemplate.TemplateDispatchAssistant,
}

type Session struct {
	bench  *Bench
	User   *tenant.User
	Actor  permissionports.RequestActor
	Tenant pagination.TenantInfo
}

func (b *Bench) SessionFor(ctx context.Context, email string) (*Session, error) {
	if strings.TrimSpace(email) == "" {
		email = DefaultUserEmail
	}

	user, err := b.Users.FindByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		return nil, fmt.Errorf("find the bench user %s: %w", email, err)
	}

	info := pagination.TenantInfo{
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
		UserID: user.ID,
	}

	return &Session{
		bench:  b,
		User:   user,
		Actor:  *permissionports.UserActor(info),
		Tenant: info,
	}, nil
}

func (s *Session) Context(ctx context.Context) context.Context {
	return dbscope.WithTenant(ctx, s.Tenant.DBTenant())
}

func (s *Session) Agents(ctx context.Context) ([]*agentdefinition.Definition, error) {
	result, err := s.bench.Definitions.List(
		s.Context(ctx),
		&agentdefinitionports.ListAgentDefinitionRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: s.Tenant,
				Pagination: pagination.Info{Limit: pagination.MaxLimit},
			},
			EnabledOnly: true,
			ChatOnly:    true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list chat agents: %w", err)
	}

	return result.Items, nil
}

func (s *Session) Agent(ctx context.Context, ref string) (*agentdefinition.Definition, error) {
	agents, err := s.Agents(ctx)
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("the organization has no enabled chat agent")
	}

	ref = strings.TrimSpace(ref)
	if ref == "" {
		for _, template := range defaultAgentTemplates {
			for _, definition := range agents {
				if definition.Template == template {
					return definition, nil
				}
			}
		}

		return agents[0], nil
	}

	for _, definition := range agents {
		if definition.ID.String() == ref ||
			strings.EqualFold(definition.Name, ref) ||
			strings.EqualFold(string(definition.Template), ref) {
			return definition, nil
		}
	}

	return nil, fmt.Errorf("no enabled chat agent named %q (try `trenova desk agents`)", ref)
}

func (s *Session) Providers(ctx context.Context) ([]*aiprovider.Provider, error) {
	providers, err := s.bench.Providers.ListForTask(
		s.Context(ctx),
		aiproviderports.ListAIProvidersForTaskRequest{
			Task:       aitask.TaskAssistantChat,
			TenantInfo: s.Tenant,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list chat providers: %w", err)
	}

	return providers, nil
}

func (s *Session) Provider(ctx context.Context, ref string) (*aiprovider.Provider, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.EqualFold(ref, benchcell.DefaultProviderLabel) {
		return nil, nil
	}

	providers, err := s.Providers(ctx)
	if err != nil {
		return nil, err
	}

	for _, provider := range providers {
		if provider.ID.String() == ref ||
			strings.EqualFold(provider.Name, ref) ||
			strings.EqualFold(provider.Model, ref) {
			return provider, nil
		}
	}

	return nil, fmt.Errorf(
		"no enabled chat provider named %q (try `trenova desk providers`)", ref,
	)
}

const cleanPageSize = 100

func (s *Session) CleanBenchThreads(ctx context.Context) (int, error) {
	ctx = s.Context(ctx)

	doomed := make([]pulid.ID, 0, cleanPageSize)
	for cursor := ""; ; {
		page, err := s.bench.Assistant.ListThreads(ctx, conversationports.ListThreadsRequest{
			UserID:          s.User.ID,
			TenantInfo:      s.Tenant,
			Limit:           cleanPageSize,
			Cursor:          cursor,
			IncludeUnlisted: true,
		})
		if err != nil {
			return 0, fmt.Errorf("list conversations: %w", err)
		}
		for _, thread := range page.Items {
			if strings.HasPrefix(thread.Title, BenchTitlePrefix) {
				doomed = append(doomed, thread.ID)
			}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	for idx, threadID := range doomed {
		if err := s.bench.Assistant.DeleteThread(ctx, s.threadRequest(threadID)); err != nil {
			return idx, fmt.Errorf("delete conversation %s: %w", threadID, err)
		}
	}

	return len(doomed), nil
}

func (s *Session) Cell(ctx context.Context, ref string) (benchcell.Cell, error) {
	parsed, err := benchcell.ParseRef(ref)
	if err != nil {
		return benchcell.Cell{}, err
	}
	provider, err := s.Provider(ctx, parsed.Name)
	if err != nil {
		return benchcell.Cell{}, err
	}
	if err = benchcell.CheckAdvisor(provider, parsed.Advisor); err != nil {
		return benchcell.Cell{}, fmt.Errorf("%s: %w", benchcell.ProviderLabel(provider), err)
	}

	return benchcell.Cell{Provider: provider, Effort: parsed.Effort, Advisor: parsed.Advisor}, nil
}
