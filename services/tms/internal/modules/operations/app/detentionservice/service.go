package detentionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func (s *Service) assemble() {
	s.effects = detentioneffects.New(detentioneffects.Deps{
		Logger:        s.l,
		EvidenceRepo:  s.evidenceRepo,
		Organizations: s.organizations,
		AuditService:  s.auditService,
		Realtime:      s.realtime,
		Now:           s.now,
	})
	s.syncer = detentionsync.New(detentionsync.Deps{
		Logger:            s.l,
		PolicyRepo:        s.policyRepo,
		OccurrenceRepo:    s.occurrenceRepo,
		EvidenceRepo:      s.evidenceRepo,
		AccessorialRepo:   s.accessorialRepo,
		AgreementRepo:     s.agreementRepo,
		PayAssignmentRepo: s.payAssignmentRepo,
		PayProfileRepo:    s.payProfileRepo,
		CommentRepo:       s.commentRepo,
		Watchtower:        s.watchtower,
		Publisher:         s.publisher,
		Now:               s.now,
		Effects:           s.effects,
	})
	s.simulator = detentionsimulation.New(detentionsimulation.Deps{
		AccessorialRepo: s.accessorialRepo,
		AnalyticsRepo:   s.analyticsRepo,
		Now:             s.now,
		Effects:         s.effects,
	})
	s.notices = detentionnoticeservice.New(detentionnoticeservice.Deps{
		Logger:           s.l,
		PolicyRepo:       s.policyRepo,
		OccurrenceRepo:   s.occurrenceRepo,
		NoticeRepo:       s.noticeRepo,
		CustomerRepo:     s.customerRepo,
		DocumentTypeRepo: s.documentTypeRepo,
		EmailService:     s.emailService,
		UploadService:    s.uploadService,
		WorkflowStarter:  s.workflowStarter,
		Templates:        s.templates,
		ContextBuilder:   s.contextBuilder,
		Publisher:        s.publisher,
		Senders:          s.senders,
		DocumentStore:    s.documentStore,
		Documents:        s.documents,
		Now:              s.now,
		Effects:          s.effects,
	})
	s.review = detentionreview.New(detentionreview.Deps{
		OccurrenceRepo: s.occurrenceRepo,
		Now:            s.now,
		Effects:        s.effects,
	})
}

type Params struct {
	fx.In

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
	// Watchtower puts a running detention clock on the feed and takes it
	// off once the occurrence closes.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
	// Publisher wakes whichever agent covers detention when a clock starts
	// and when a notice is left for a person to send.
	Publisher agentports.AgentEventPublisher `optional:"true"`
	// Realtime refreshes the billing queue when a charge starts or stops
	// holding its shipment off an invoice.
	Realtime realtimeports.RealtimeService `optional:"true"`
	// Senders names who a notice would go out as, for its preview.
	Senders       emailports.EmailSenderResolver   `optional:"true"`
	DocumentStore documentports.DocumentStore      `optional:"true"`
	Documents     documentports.DocumentRepository `optional:"true"`
}

type Service struct {
	l                 *zap.Logger
	policyRepo        detentionports.DetentionPolicyRepository
	occurrenceRepo    detentionports.DetentionOccurrenceRepository
	evidenceRepo      detentionports.DetentionEvidenceRepository
	accessorialRepo   accessorialchargeports.AccessorialChargeRepository
	agreementRepo     rateagreementports.RateAgreementRepository
	payAssignmentRepo driverpayports.WorkerPayAssignmentRepository
	payProfileRepo    driverpayports.PayProfileRepository
	organizations     tenantports.OrganizationRepository
	noticeRepo        detentionports.DetentionNoticeRepository
	analyticsRepo     detentionports.DetentionAnalyticsRepository
	customerRepo      customerports.CustomerRepository
	commentRepo       shipmentcommentports.ShipmentCommentRepository
	documentTypeRepo  documenttypeports.DocumentTypeRepository
	emailService      emailports.EmailService
	uploadService     documentuploadports.DocumentUploadService
	workflowStarter   kernelports.WorkflowStarter
	templates         documenttemplateports.DocumentTemplateResolver
	contextBuilder    *detentionnoticerender.ContextBuilder
	auditService      auditports.AuditService
	watchtower        watchtowerports.WatchtowerProjector
	publisher         agentports.AgentEventPublisher
	realtime          realtimeports.RealtimeService
	senders           emailports.EmailSenderResolver
	documentStore     documentports.DocumentStore
	documents         documentports.DocumentRepository
	now               func() int64
	effects           *detentioneffects.Effects
	syncer            *detentionsync.Service
	simulator         *detentionsimulation.Service
	notices           *detentionnoticeservice.Service
	review            *detentionreview.Service
}

var _ detentionports.DetentionNoticeService = (*Service)(nil)

func New(p Params) *Service {
	s := &Service{
		l:                 p.Logger.Named("service.detention"),
		policyRepo:        p.PolicyRepo,
		occurrenceRepo:    p.OccurrenceRepo,
		evidenceRepo:      p.EvidenceRepo,
		accessorialRepo:   p.AccessorialRepo,
		agreementRepo:     p.AgreementRepo,
		payAssignmentRepo: p.PayAssignmentRepo,
		payProfileRepo:    p.PayProfileRepo,
		organizations:     p.Organizations,
		noticeRepo:        p.NoticeRepo,
		analyticsRepo:     p.AnalyticsRepo,
		customerRepo:      p.CustomerRepo,
		commentRepo:       p.CommentRepo,
		documentTypeRepo:  p.DocumentTypeRepo,
		emailService:      p.EmailService,
		uploadService:     p.UploadService,
		workflowStarter:   p.WorkflowStarter,
		templates:         p.Templates,
		contextBuilder:    p.ContextBuilder,
		watchtower:        p.Watchtower,
		publisher:         p.Publisher,
		realtime:          p.Realtime,
		senders:           p.Senders,
		documentStore:     p.DocumentStore,
		documents:         p.Documents,
		auditService:      p.AuditService,
		now:               timeutils.NowUnix,
	}
	s.assemble()
	return s
}

func (s *Service) SyncShipment(
	ctx context.Context,
	entity *shipment.Shipment,
) (*detentionsync.SyncShipmentResult, error) {
	return s.syncer.SyncShipment(ctx, entity)
}

func (s *Service) Backtest(
	ctx context.Context,
	req detentionsimulation.BacktestRequest,
	tenantInfo pagination.TenantInfo,
) (*detentionsimulation.BacktestResult, error) {
	return s.simulator.Backtest(ctx, req, tenantInfo)
}

func (s *Service) FacilityProfiles(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.FacilityDetentionStat, error) {
	return s.simulator.FacilityProfiles(ctx, req)
}

func (s *Service) CustomerProfiles(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.CustomerDetentionStat, error) {
	return s.simulator.CustomerProfiles(ctx, req)
}

func (s *Service) WaiverLeakage(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.WaiverLeakageStat, error) {
	return s.simulator.WaiverLeakage(ctx, req)
}

func (s *Service) PreviewPolicy(
	ctx context.Context,
	policy *detentionpolicy.DetentionPolicy,
	scenario detentionsimulation.PreviewScenario,
	tenantInfo pagination.TenantInfo,
) (*detentionsimulation.PreviewResult, error) {
	return s.simulator.PreviewPolicy(ctx, policy, scenario, tenantInfo)
}

func (s *Service) BuildNotice(
	ctx context.Context,
	p *detentionnoticerender.BuildNoticeParams,
) (detentionnoticerender.NoticeContent, error) {
	return s.notices.BuildNotice(ctx, p)
}

func (s *Service) ScheduleNotice(
	ctx context.Context,
	p detentionnoticeservice.ScheduleNoticeParams,
) (*detentionnoticeservice.ScheduledNotice, error) {
	return s.notices.ScheduleNotice(ctx, p)
}

func (s *Service) SendNotice(
	ctx context.Context,
	p *detentionnoticeservice.SendNoticeParams,
) (*detentionnotice.DetentionNotice, error) {
	return s.notices.SendNotice(ctx, p)
}

func (s *Service) SendOccurrenceNotice(
	ctx context.Context,
	p detentionnoticeservice.SendOccurrenceNoticeParams,
) (*detention.DetentionOccurrence, error) {
	return s.notices.SendOccurrenceNotice(ctx, p)
}

func (s *Service) SweepNoticesDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*detentionports.NoticeSweepResult, error) {
	return s.notices.SweepNoticesDue(ctx, tenantInfo)
}

func (s *Service) RecordNoticeDelivery(
	ctx context.Context,
	notice *detentionnotice.DetentionNotice,
	status detentionnotice.NoticeDeliveryStatus,
	reason string,
) (*detentionnotice.DetentionNotice, error) {
	return s.notices.RecordNoticeDelivery(ctx, notice, status, reason)
}

func (s *Service) AttachNoticePDFDocument(
	ctx context.Context,
	p *detentionports.AttachNoticePDFDocumentParams,
) error {
	return s.notices.AttachNoticePDFDocument(ctx, p)
}

func (s *Service) PreviewOccurrenceNotice(
	ctx context.Context,
	p *detentionnoticeservice.SendOccurrenceNoticeParams,
) (*detentionnoticeservice.NoticePreview, error) {
	return s.notices.PreviewOccurrenceNotice(ctx, p)
}

func (s *Service) DisputeRecord(
	ctx context.Context,
	p *detentionnoticeservice.DisputeRecordParams,
) (*detentionnoticeservice.DisputeRecord, error) {
	return s.notices.DisputeRecord(ctx, p)
}

func (s *Service) GetOccurrenceDetail(
	ctx context.Context,
	req *detentionports.GetDetentionOccurrenceByIDRequest,
) (*detentionreview.OccurrenceDetail, error) {
	return s.review.GetOccurrenceDetail(ctx, req)
}

func (s *Service) ListOccurrences(
	ctx context.Context,
	req *detentionports.ListDetentionOccurrencesRequest,
) (*pagination.ListResult[*detention.DetentionOccurrence], error) {
	return s.review.ListOccurrences(ctx, req)
}

func (s *Service) ListByShipment(
	ctx context.Context,
	req *detentionports.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	return s.review.ListByShipment(ctx, req)
}

func (s *Service) ListDesk(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*detentionreview.DeskEntry, error) {
	return s.review.ListDesk(ctx, tenantInfo)
}

func (s *Service) PriceOpenClock(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	occurrence *detention.DetentionOccurrence,
) (*detentionreview.ClockPrice, error) {
	return s.review.PriceOpenClock(ctx, tenantInfo, occurrence)
}

func (s *Service) ListAwaitingApproval(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*detentionreview.DeskEntry, error) {
	return s.review.ListAwaitingApproval(ctx, tenantInfo)
}

func (s *Service) ListDeskEntries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	filter *detentionreview.DeskFilter,
) ([]*detentionreview.DeskEntry, error) {
	return s.review.ListDeskEntries(ctx, tenantInfo, filter)
}

func (s *Service) Waive(
	ctx context.Context,
	p detentionreview.WaiveParams,
) (*detention.DetentionOccurrence, error) {
	return s.review.Waive(ctx, p)
}

func (s *Service) WaiveUninvoiced(
	ctx context.Context,
	p *detentionreview.WaiveParams,
) (*detention.DetentionOccurrence, error) {
	return s.review.WaiveUninvoiced(ctx, p)
}

func (s *Service) Approve(
	ctx context.Context,
	p detentionreview.ApproveParams,
) (*detention.DetentionOccurrence, error) {
	return s.review.Approve(ctx, p)
}

func (s *Service) Dispute(
	ctx context.Context,
	p detentionreview.DisputeParams,
) (*detention.DetentionOccurrence, error) {
	return s.review.Dispute(ctx, p)
}

func (s *Service) Escalate(
	ctx context.Context,
	p detentionreview.EscalateParams,
) (*detention.DetentionOccurrence, error) {
	return s.review.Escalate(ctx, p)
}

func (s *Service) BuildDisputePacket(
	ctx context.Context,
	occurrenceID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*detentionreview.DisputePacket, error) {
	return s.review.BuildDisputePacket(ctx, occurrenceID, tenantInfo)
}

func (s *Service) PreviewApprove(
	ctx context.Context,
	p *detentionreview.ApproveParams,
) (*detentionreview.OccurrenceChange, error) {
	return s.review.PreviewApprove(ctx, p)
}

func (s *Service) PreviewWaive(
	ctx context.Context,
	p *detentionreview.WaiveParams,
) (*detentionreview.OccurrenceChange, error) {
	return s.review.PreviewWaive(ctx, p)
}

func (s *Service) PreviewDispute(
	ctx context.Context,
	p *detentionreview.DisputeParams,
) (*detentionreview.OccurrenceChange, error) {
	return s.review.PreviewDispute(ctx, p)
}
