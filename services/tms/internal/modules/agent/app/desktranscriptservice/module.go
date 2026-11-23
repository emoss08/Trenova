package desktranscriptservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/desktranscriptexport"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Conversations desktranscriptports.ConversationRepository
	ExportRepo    desktranscriptports.ExportRepository
	Renderer      desktranscriptports.TranscriptRenderer
	Users         tenantports.UserRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Organizations tenantports.OrganizationRepository
	Permissions   permissionports.PermissionEngine
	Auditor       permissionports.SecurityAuditor
	Storage       kernelports.Client
	Workflows     kernelports.WorkflowStarter
	Notifications *notificationservice.Service
	Realtime      realtimeports.RealtimeService
	Logger        *zap.Logger
}

type Components struct {
	fx.Out

	Service desktranscriptports.Service
	Exports *desktranscriptexport.Exports
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) Components {
	assembler := desktranscriptexport.NewAssembler(&desktranscriptexport.AssemblerParams{
		Conversations: p.Conversations,
		Users:         p.Users,
		Definitions:   p.Definitions,
		Renderer:      p.Renderer,
	})
	exports := desktranscriptexport.NewExports(&desktranscriptexport.ExportsParams{
		Exports:       p.ExportRepo,
		Conversations: p.Conversations,
		Assembler:     assembler,
		Organizations: p.Organizations,
		Storage:       p.Storage,
		Notifier:      p.Notifications,
		Realtime:      p.Realtime,
		Logger:        p.Logger,
	})

	return Components{
		Service: NewService(&ServiceParams{
			Conversations: p.Conversations,
			ExportRepo:    p.ExportRepo,
			Assembler:     assembler,
			Exports:       exports,
			Organizations: p.Organizations,
			Permissions:   p.Permissions,
			Auditor:       p.Auditor,
			Storage:       p.Storage,
			Workflows:     p.Workflows,
			Logger:        p.Logger,
		}),
		Exports: exports,
	}
}

var Module = fx.Module("desk-transcript-service", fx.Provide(New))
