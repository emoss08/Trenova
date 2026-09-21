package agentruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ContextBuilderParams struct {
	fx.In

	Logger        *zap.Logger
	Organizations repositories.OrganizationRepository
	Users         repositories.UserRepository
	Runtime       serviceports.AgentRuntime
	Memories      serviceports.AgentMemoryService `optional:"true"`
}

type ContextBuilder struct {
	logger        *zap.Logger
	organizations repositories.OrganizationRepository
	users         repositories.UserRepository
	runtime       serviceports.AgentRuntime
	memories      serviceports.AgentMemoryService
}

func NewContextBuilder(p ContextBuilderParams) serviceports.RuntimeContextBuilder {
	return &ContextBuilder{
		logger:        p.Logger.Named("service.agentruntime.context"),
		organizations: p.Organizations,
		users:         p.Users,
		runtime:       p.Runtime,
		memories:      p.Memories,
	}
}

func (b *ContextBuilder) Build(
	ctx context.Context,
	req *serviceports.RuntimeContextRequest,
) (agentdefinition.RuntimeContext, error) {
	definition := req.Definition
	rc := agentdefinition.RuntimeContext{
		Now:     timeutils.NowUnix(),
		Trigger: req.Trigger,
		Subject: req.Subject,
		Page:    req.Page,
		Tools:   b.runtime.ToolSummaries(definition),
	}

	tenant := req.Actor.TenantInfo()

	// The organization is read on every build, not only when the prompt wants
	// its name. Its timezone is what every tool means by "today", and a tool
	// draws the day boundary in UTC unless it is told otherwise — so an agent
	// whose prompt providers happened not to include the organization got a
	// clock a few hours off in every date filter it ran.
	describe := definition.HasContextProvider(agentdefinition.ContextOrganization) ||
		definition.HasContextProvider(agentdefinition.ContextClock)
	org, err := b.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenant,
		IncludeBU:  describe,
	})
	if err != nil {
		b.logger.Warn("agent context: organization lookup failed",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err),
		)
	} else {
		rc.Timezone = org.Timezone
		if describe {
			rc.OrganizationName = org.Name
			if org.BusinessUnit != nil {
				rc.BusinessUnitName = org.BusinessUnit.Name
			}
		}
	}

	if definition.HasContextProvider(agentdefinition.ContextUser) && req.Actor.UserID.IsNotNil() {
		user, err := b.users.GetByID(ctx, repositories.GetUserByIDRequest{
			TenantInfo:   tenant,
			LookupUserID: req.Actor.UserID,
		})
		if err != nil {
			b.logger.Warn("agent context: user lookup failed",
				zap.String("user", req.Actor.UserID.String()),
				zap.Error(err),
			)
		} else {
			roles := make([]string, 0, len(user.Assignments))
			for _, assignment := range user.Assignments {
				if assignment != nil && assignment.Role != nil && assignment.Role.Name != "" {
					roles = append(roles, assignment.Role.Name)
				}
			}
			rc.User = &agentdefinition.RuntimeUser{
				Name:  user.Name,
				Email: user.EmailAddress,
				Roles: roles,
			}
			if rc.Timezone == "" {
				rc.Timezone = user.Timezone
			}
		}
	}

	// Memory is read for every agent that asks for it, scoped to the
	// organization and to the agent's own tools: a correction to assign_move
	// belongs in the prompt of an agent that can assign, and nowhere else.
	if definition.HasContextProvider(agentdefinition.ContextMemory) && b.memories != nil {
		memories, err := b.memories.ForContext(ctx, serviceports.MemoryContextRequest{
			TenantInfo: tenant,
			ToolNames:  definition.ToolNames,
		})
		if err != nil {
			b.logger.Warn("agent context: memory lookup failed",
				zap.String("organization", tenant.OrgID.String()),
				zap.Error(err),
			)
		} else {
			rc.Memories = memories
		}
	}

	return rc, nil
}
