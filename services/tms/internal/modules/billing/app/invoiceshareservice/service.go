package invoiceshareservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceshare"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	Config              *config.Config
	Repo                invoiceports.InvoiceShareRepository
	InvoiceRepo         invoiceports.InvoiceRepository
	UserRepo            tenantports.UserRepository
	OrganizationRepo    tenantports.OrganizationRepository
	Permissions         permissionports.PermissionEngine
	NotificationService *notificationservice.Service
	EmailService        emailports.EmailService
	Templates           documenttemplateports.DocumentTemplateResolver
	AuditService        auditports.AuditService
}

type Service struct {
	l                   *zap.Logger
	cfg                 *config.Config
	repo                invoiceports.InvoiceShareRepository
	invoiceRepo         invoiceports.InvoiceRepository
	userRepo            tenantports.UserRepository
	organizationRepo    tenantports.OrganizationRepository
	permissions         permissionports.PermissionEngine
	notificationService *notificationservice.Service
	emailService        emailports.EmailService
	templates           documenttemplateports.DocumentTemplateResolver
	auditService        auditports.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.invoice-share"),
		cfg:                 p.Config,
		repo:                p.Repo,
		invoiceRepo:         p.InvoiceRepo,
		userRepo:            p.UserRepo,
		organizationRepo:    p.OrganizationRepo,
		permissions:         p.Permissions,
		notificationService: p.NotificationService,
		emailService:        p.EmailService,
		templates:           p.Templates,
		auditService:        p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *invoiceports.ListInvoiceSharesRequest,
) ([]*invoiceshare.InvoiceShare, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	if _, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	return s.repo.ListByInvoiceID(ctx, req)
}

func (s *Service) Share(
	ctx context.Context,
	req *invoiceports.ShareInvoiceRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.ShareInvoiceResult, error) {
	plan, err := s.planShare(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	entity, sharer, recipients, shares, input := plan.invoice, plan.sharer, plan.recipients,
		plan.shares, &plan.input

	if err = s.repo.Upsert(ctx, shares); err != nil {
		return nil, err
	}

	s.logShare(entity, actor, recipients, input)

	emailsQueued, emailStatus := s.deliver(ctx, &deliveryParams{
		invoice:    entity,
		sharer:     sharer,
		recipients: recipients,
		note:       input.note,
		tab:        input.tab,
	})

	listed, err := s.repo.ListByInvoiceID(ctx, &invoiceports.ListInvoiceSharesRequest{
		TenantInfo: req.TenantInfo,
		InvoiceID:  entity.ID,
	})
	if err != nil {
		return nil, err
	}

	return &invoiceports.ShareInvoiceResult{
		Shares:         listed,
		RecipientCount: len(recipients),
		EmailsQueued:   emailsQueued,
		EmailStatus:    emailStatus,
	}, nil
}

func buildShares(
	entity *invoice.Invoice,
	sharerID pulid.ID,
	recipients []*tenant.User,
	input *shareInput,
) ([]*invoiceshare.InvoiceShare, error) {
	now := timeutils.NowUnix()
	shares := make([]*invoiceshare.InvoiceShare, 0, len(recipients))
	multiErr := errortypes.NewMultiError()

	for _, recipient := range recipients {
		share := &invoiceshare.InvoiceShare{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			InvoiceID:      entity.ID,
			SharedWithID:   recipient.ID,
			SharedByID:     sharerID,
			Note:           input.note,
			Tab:            input.tab,
			ShareCount:     1,
			FirstSharedAt:  now,
			LastSharedAt:   now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		share.Validate(multiErr)
		shares = append(shares, share)
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return shares, nil
}

func (s *Service) logShare(
	entity *invoice.Invoice,
	actor *permissionports.RequestActor,
	recipients []*tenant.User,
	input *shareInput,
) {
	auditActor := actor.AuditActor()
	recipientIDs := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		recipientIDs = append(recipientIDs, recipient.ID.String())
	}

	if err := s.auditService.LogAction(
		&auditports.LogActionParams{
			Resource:       permission.ResourceInvoice,
			ResourceID:     entity.ID.String(),
			Operation:      permission.OpAssign,
			UserID:         auditActor.UserID,
			APIKeyID:       auditActor.APIKeyID,
			PrincipalType:  auditActor.PrincipalType,
			PrincipalID:    auditActor.PrincipalID,
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		auditservice.WithComment("Invoice shared with teammates"),
		auditservice.WithMetadata(map[string]any{
			"invoiceNumber":    entity.Number,
			"recipientUserIds": recipientIDs,
			"tab":              string(input.tab),
			"hasNote":          input.note != "",
		}),
	); err != nil {
		s.l.Error("failed to log invoice share", zap.Error(err))
	}
}
