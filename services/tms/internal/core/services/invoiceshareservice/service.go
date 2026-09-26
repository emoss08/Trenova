package invoiceshareservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
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
	Repo                repositories.InvoiceShareRepository
	InvoiceRepo         repositories.InvoiceRepository
	UserRepo            repositories.UserRepository
	OrganizationRepo    repositories.OrganizationRepository
	Permissions         servicesports.PermissionEngine
	NotificationService *notificationservice.Service
	EmailService        servicesports.EmailService
	Templates           servicesports.DocumentTemplateResolver
	AuditService        servicesports.AuditService
}

type Service struct {
	l                   *zap.Logger
	cfg                 *config.Config
	repo                repositories.InvoiceShareRepository
	invoiceRepo         repositories.InvoiceRepository
	userRepo            repositories.UserRepository
	organizationRepo    repositories.OrganizationRepository
	permissions         servicesports.PermissionEngine
	notificationService *notificationservice.Service
	emailService        servicesports.EmailService
	templates           servicesports.DocumentTemplateResolver
	auditService        servicesports.AuditService
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
	req *repositories.ListInvoiceSharesRequest,
) ([]*invoice.InvoiceShare, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	if _, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	return s.repo.ListByInvoiceID(ctx, req)
}

func (s *Service) Share(
	ctx context.Context,
	req *servicesports.ShareInvoiceRequest,
	actor *servicesports.RequestActor,
) (*servicesports.ShareInvoiceResult, error) {
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

	listed, err := s.repo.ListByInvoiceID(ctx, &repositories.ListInvoiceSharesRequest{
		TenantInfo: req.TenantInfo,
		InvoiceID:  entity.ID,
	})
	if err != nil {
		return nil, err
	}

	return &servicesports.ShareInvoiceResult{
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
) ([]*invoice.InvoiceShare, error) {
	now := timeutils.NowUnix()
	shares := make([]*invoice.InvoiceShare, 0, len(recipients))
	multiErr := errortypes.NewMultiError()

	for _, recipient := range recipients {
		share := &invoice.InvoiceShare{
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
	actor *servicesports.RequestActor,
	recipients []*tenant.User,
	input *shareInput,
) {
	auditActor := actor.AuditActor()
	recipientIDs := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		recipientIDs = append(recipientIDs, recipient.ID.String())
	}

	if err := s.auditService.LogAction(
		&servicesports.LogActionParams{
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
