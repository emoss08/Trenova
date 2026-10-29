package reportcatalogjobs

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportnotify"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	DefinitionRepo      reportports.ReportDefinitionRepository
	CatalogVersions     reportports.ReportCatalogVersionRepository
	Engine              reportports.ReportEngine
	NotificationService *notificationservice.Service
	Templates           documenttemplateports.DocumentTemplateResolver
	AuditService        auditports.AuditService
	Logger              *zap.Logger
}

type Activities struct {
	defRepo      reportports.ReportDefinitionRepository
	versions     reportports.ReportCatalogVersionRepository
	engine       reportports.ReportEngine
	notification *notificationservice.Service
	templates    documenttemplateports.DocumentTemplateResolver
	audit        auditports.AuditService
	l            *zap.Logger
}

//nolint:gocritic // fx.In parameter structs must be passed by value
func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		defRepo:      p.DefinitionRepo,
		versions:     p.CatalogVersions,
		engine:       p.Engine,
		notification: p.NotificationService,
		templates:    p.Templates,
		audit:        p.AuditService,
		l:            p.Logger.Named("reportjobs"),
	}
}

func (a *Activities) notices() *reportnotify.Renderer {
	return reportnotify.NewRenderer(a.templates, a.l)
}
