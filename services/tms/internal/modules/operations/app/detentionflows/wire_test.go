package detentionflows

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentioneffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticeservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionsimulation"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionsync"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/zap"
)

type wiring struct {
	Logger            *zap.Logger
	PolicyRepo        detentionports.DetentionPolicyRepository
	OccurrenceRepo    detentionports.DetentionOccurrenceRepository
	EvidenceRepo      detentionports.DetentionEvidenceRepository
	AccessorialRepo   accessorialchargeports.AccessorialChargeRepository
	AgreementRepo     rateagreementports.RateAgreementRepository
	PayAssignmentRepo driverpayports.WorkerPayAssignmentRepository
	PayProfileRepo    driverpayports.PayProfileRepository
	Organizations     tenantports.OrganizationRepository
	NoticeRepo        detentionports.DetentionNoticeRepository
	AnalyticsRepo     detentionports.DetentionAnalyticsRepository
	CustomerRepo      customerports.CustomerRepository
	CommentRepo       shipmentcommentports.ShipmentCommentRepository
	DocumentTypeRepo  documenttypeports.DocumentTypeRepository
	EmailService      emailports.EmailService
	UploadService     documentuploadports.DocumentUploadService
	WorkflowStarter   kernelports.WorkflowStarter
	Templates         documenttemplateports.DocumentTemplateResolver
	ContextBuilder    *detentionnoticerender.ContextBuilder
	AuditService      auditports.AuditService
	Watchtower        watchtowerports.WatchtowerProjector
	Publisher         agenteventports.AgentEventPublisher
	Realtime          realtimeports.RealtimeService
	Senders           emailports.EmailSenderResolver
	DocumentStore     documentports.DocumentStore
	Documents         documentports.DocumentRepository
	Now               func() int64
}

type services struct {
	effects   *detentioneffects.Effects
	syncer    *detentionsync.Service
	simulator *detentionsimulation.Service
	notices   *detentionnoticeservice.Service
	review    *detentionreview.Service
}

func wire(p *wiring) services {
	l := p.Logger
	effects := detentioneffects.New(detentioneffects.Params{
		Logger:        l,
		EvidenceRepo:  p.EvidenceRepo,
		Organizations: p.Organizations,
		AuditService:  p.AuditService,
		Realtime:      p.Realtime,
		Now:           p.Now,
	})
	syncer := detentionsync.New(detentionsync.Params{
		Logger:            l,
		PolicyRepo:        p.PolicyRepo,
		OccurrenceRepo:    p.OccurrenceRepo,
		EvidenceRepo:      p.EvidenceRepo,
		AccessorialRepo:   p.AccessorialRepo,
		AgreementRepo:     p.AgreementRepo,
		PayAssignmentRepo: p.PayAssignmentRepo,
		PayProfileRepo:    p.PayProfileRepo,
		CommentRepo:       p.CommentRepo,
		Watchtower:        p.Watchtower,
		Publisher:         p.Publisher,
		Now:               p.Now,
		Effects:           effects,
	})
	simulator := detentionsimulation.New(detentionsimulation.Params{
		AccessorialRepo: p.AccessorialRepo,
		AnalyticsRepo:   p.AnalyticsRepo,
		Now:             p.Now,
		Effects:         effects,
	})
	notices := detentionnoticeservice.New(detentionnoticeservice.Params{
		Logger:           l,
		PolicyRepo:       p.PolicyRepo,
		OccurrenceRepo:   p.OccurrenceRepo,
		NoticeRepo:       p.NoticeRepo,
		CustomerRepo:     p.CustomerRepo,
		DocumentTypeRepo: p.DocumentTypeRepo,
		EmailService:     p.EmailService,
		UploadService:    p.UploadService,
		WorkflowStarter:  p.WorkflowStarter,
		Templates:        p.Templates,
		ContextBuilder:   p.ContextBuilder,
		Publisher:        p.Publisher,
		Senders:          p.Senders,
		DocumentStore:    p.DocumentStore,
		Documents:        p.Documents,
		Now:              p.Now,
		Effects:          effects,
	})
	review := detentionreview.New(detentionreview.Params{
		OccurrenceRepo: p.OccurrenceRepo,
		Now:            p.Now,
		Effects:        effects,
	})
	return services{
		effects:   effects,
		syncer:    syncer,
		simulator: simulator,
		notices:   notices,
		review:    review,
	}
}
