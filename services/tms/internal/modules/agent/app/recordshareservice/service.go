// Package recordshareservice shares a record with teammates: it records who
// was told about what, tells them in the app and by email, and lists who a
// record has been shared with. It serves every kind of record the permission
// registry names, so an invoice, a shipment and a worker's case are shared the
// same way and read back from the same place.
package recordshareservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/app/deskrecords"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordshare"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordshareports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	AppConfig           *appconfig.Config
	Repo                recordshareports.Repository
	Labels              *deskrecords.Labeler
	UserRepo            tenantports.UserRepository
	OrganizationRepo    tenantports.OrganizationRepository
	Permissions         permissionports.PermissionEngine
	NotificationService *notificationservice.Service
	EmailService        emailports.EmailService
	Templates           documenttemplateports.DocumentTemplateResolver
	AuditService        auditports.AuditService
	Owners              caseownerports.Service
}

type Service struct {
	l                   *zap.Logger
	cfg                 *appconfig.Config
	repo                recordshareports.Repository
	labels              *deskrecords.Labeler
	userRepo            tenantports.UserRepository
	organizationRepo    tenantports.OrganizationRepository
	permissions         permissionports.PermissionEngine
	notificationService *notificationservice.Service
	emailService        emailports.EmailService
	templates           documenttemplateports.DocumentTemplateResolver
	auditService        auditports.AuditService
	owners              caseownerports.Service
	now                 func() int64
}

func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.record-share"),
		cfg:                 p.AppConfig,
		repo:                p.Repo,
		labels:              p.Labels,
		userRepo:            p.UserRepo,
		organizationRepo:    p.OrganizationRepo,
		permissions:         p.Permissions,
		notificationService: p.NotificationService,
		emailService:        p.EmailService,
		templates:           p.Templates,
		auditService:        p.AuditService,
		owners:              p.Owners,
		now:                 timeutils.NowUnix,
	}
}

func AsService(s *Service) recordshareports.Service { return s }

// record reads what is being shared as the actor may read it. A kind the
// registry does not name, a record that is not there and one the actor may
// not read are all refused: nobody shares what they cannot open themselves.
func (s *Service) record(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	kind recordkind.Kind,
	id pulid.ID,
	actor *permissionports.RequestActor,
) (*recordshareports.Record, error) {
	if !kind.Known() || kind.Resource() == "" {
		return nil, errortypes.NewNotFoundError("That kind of record cannot be shared")
	}
	if prefix, ok := kind.IDPrefix(); !ok || id.Prefix() != prefix {
		return nil, errortypes.NewNotFoundError("That record was not found")
	}

	found := &deskattentionports.Record{Kind: kind, ID: id}
	if err := s.labels.Label(
		ctx,
		tenantInfo,
		actor,
		[]*deskattentionports.Record{found},
	); err != nil {
		return nil, err
	}
	if !found.Readable {
		return nil, errortypes.NewAuthorizationError("You may not read this record")
	}
	if found.Label == "" {
		return nil, errortypes.NewNotFoundError("That record was not found")
	}

	return &recordshareports.Record{
		Kind:     kind,
		ID:       id,
		Label:    found.Label,
		Subtitle: found.Subtitle,
	}, nil
}

func (s *Service) List(
	ctx context.Context,
	req *recordshareports.RecordRequest,
	actor *permissionports.RequestActor,
) ([]*recordshare.Share, error) {
	if _, err := s.record(ctx, req.TenantInfo, req.Kind, req.RecordID, actor); err != nil {
		return nil, err
	}

	return s.standing(ctx, req, actor)
}

// standing is who the record is shared with now, each share marked with
// whether the actor may take it back.
func (s *Service) standing(
	ctx context.Context,
	req *recordshareports.RecordRequest,
	actor *permissionports.RequestActor,
) ([]*recordshare.Share, error) {
	shares, err := s.repo.ListByRecord(ctx, req)
	if err != nil || len(shares) == 0 {
		return shares, err
	}

	overCase, err := s.mayRemoveAny(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	for _, share := range shares {
		share.Removable = overCase || share.SharedByID == actor.UserID
	}

	return shares, nil
}

// mayRemoveAny reports whether the actor may take back any share of the
// record, not only their own: the case is theirs, or they may give cases to
// people.
func (s *Service) mayRemoveAny(
	ctx context.Context,
	req *recordshareports.RecordRequest,
	actor *permissionports.RequestActor,
) (bool, error) {
	standing, err := s.owners.Standing(ctx, caseownerports.RecordRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.RecordID,
	}, actor)
	if err != nil {
		return false, err
	}
	if standing == nil {
		return false, nil
	}

	owns := standing.Owner != nil && standing.Owner.OwnerID == actor.UserID

	return owns || standing.CanAssign, nil
}

func (s *Service) Unshare(
	ctx context.Context,
	req *recordshareports.ShareTarget,
	actor *permissionports.RequestActor,
) ([]*recordshare.Share, error) {
	record, err := s.record(ctx, req.TenantInfo, req.Kind, req.RecordID, actor)
	if err != nil {
		return nil, err
	}
	of := &recordshareports.RecordRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		RecordID:   req.RecordID,
	}

	share, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	if share.SharedByID != actor.UserID {
		overCase, mErr := s.mayRemoveAny(ctx, of, actor)
		if mErr != nil {
			return nil, mErr
		}
		if !overCase {
			return nil, errortypes.NewAuthorizationError(
				"Only the person who shared it, the case's owner or a case lead can remove this share",
			)
		}
	}

	if !share.Revoked() {
		share.Revoke(actor.UserID, s.now())
		revoked, rErr := s.repo.Revoke(ctx, share)
		if rErr != nil {
			return nil, rErr
		}
		if revoked {
			s.logUnshare(ctx, req.TenantInfo, record, share, actor)
		}
	}

	return s.standing(ctx, of, actor)
}

func (s *Service) logUnshare(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	record *recordshareports.Record,
	share *recordshare.Share,
	actor *permissionports.RequestActor,
) {
	auditActor := actor.AuditActor()

	if err := s.auditService.LogAction(ctx,
		&auditports.LogActionParams{
			Resource:       record.Kind.Resource(),
			ResourceID:     record.ID.String(),
			Operation:      permission.OpAssign,
			UserID:         auditActor.UserID,
			APIKeyID:       auditActor.APIKeyID,
			PrincipalType:  auditActor.PrincipalType,
			PrincipalID:    auditActor.PrincipalID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
		},
		auditservice.WithComment(record.Kind.Noun()+" no longer shared with a teammate"),
		auditservice.WithMetadata(map[string]any{
			"recordKind":      string(record.Kind),
			"recordLabel":     record.Label,
			"recipientUserId": share.SharedWithID.String(),
			"sharedByUserId":  share.SharedByID.String(),
		}),
	); err != nil {
		s.l.Error("failed to log record unshare", zap.Error(err))
	}
}

func (s *Service) Share(
	ctx context.Context,
	req *recordshareports.ShareRequest,
	actor *permissionports.RequestActor,
) (*recordshareports.ShareResult, error) {
	plan, err := s.planShare(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	if err = s.repo.Upsert(ctx, plan.shares); err != nil {
		return nil, err
	}

	s.logShare(ctx, req.TenantInfo, plan, actor)

	emailsQueued, emailStatus := s.deliver(ctx, req.TenantInfo, plan)

	listed, err := s.standing(ctx, &recordshareports.RecordRequest{
		TenantInfo: req.TenantInfo,
		Kind:       plan.record.Kind,
		RecordID:   plan.record.ID,
	}, actor)
	if err != nil {
		return nil, err
	}

	return &recordshareports.ShareResult{
		Shares:         listed,
		RecipientCount: len(plan.recipients),
		EmailsQueued:   emailsQueued,
		EmailStatus:    emailStatus,
	}, nil
}

func (s *Service) logShare(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	plan *sharePlan,
	actor *permissionports.RequestActor,
) {
	auditActor := actor.AuditActor()
	recipientIDs := make([]string, 0, len(plan.recipients))
	for _, recipient := range plan.recipients {
		recipientIDs = append(recipientIDs, recipient.ID.String())
	}

	if err := s.auditService.LogAction(ctx,
		&auditports.LogActionParams{
			Resource:       plan.record.Kind.Resource(),
			ResourceID:     plan.record.ID.String(),
			Operation:      permission.OpAssign,
			UserID:         auditActor.UserID,
			APIKeyID:       auditActor.APIKeyID,
			PrincipalType:  auditActor.PrincipalType,
			PrincipalID:    auditActor.PrincipalID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
		},
		auditservice.WithComment(plan.record.Kind.Noun()+" shared with teammates"),
		auditservice.WithMetadata(map[string]any{
			"recordKind":       string(plan.record.Kind),
			"recordLabel":      plan.record.Label,
			"recipientUserIds": recipientIDs,
			"path":             plan.input.path,
			"hasNote":          plan.input.note != "",
		}),
	); err != nil {
		s.l.Error("failed to log record share", zap.Error(err))
	}
}

func isShareableUser(user *tenant.User) bool {
	return user.Status == activeStatus && !user.IsLocked && user.Username != systemUsername
}
