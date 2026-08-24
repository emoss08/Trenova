package driverportalservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

type InviteWorkerRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	Email      string
}

type InviteWorkerResult struct {
	Invitation *worker.PortalInvitation
	InviteURL  string
	EmailSent  bool
}

type PortalStatus struct {
	Linked            bool
	PortalUser        *tenant.User
	PendingInvitation *worker.PortalInvitation
	Invitations       []*worker.PortalInvitation
}

type InvitationPreview struct {
	OrganizationName string `json:"organizationName"`
	WorkerFirstName  string `json:"workerFirstName"`
	Email            string `json:"email"`
	ExpiresAt        int64  `json:"expiresAt"`
}

type AcceptInvitationRequest struct {
	Token    string
	Password string
	Timezone string
}

type AcceptInvitationResult struct {
	EmailAddress     string `json:"emailAddress"`
	OrganizationName string `json:"organizationName"`
}

func (s *Service) InviteWorker(
	ctx context.Context,
	req *InviteWorkerRequest,
	actor *serviceports.RequestActor,
) (*InviteWorkerResult, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Portal invitations require an authenticated user",
		)
	}

	if err := s.requireAssetOperations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}

	wrk, err := s.portalRepo.GetWorkerForPortalManagement(ctx, req.TenantInfo, req.WorkerID)
	if err != nil {
		return nil, err
	}
	if !wrk.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalidOperation,
			"This driver already has portal access",
		)
	}
	if wrk.Status != domaintypes.StatusActive {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalidOperation,
			"Only active drivers can be invited to the portal",
		)
	}

	inviteEmail, err := resolveInviteEmail(req.Email, wrk.Email)
	if err != nil {
		return nil, err
	}
	if err = s.ensureNoPendingInvitation(ctx, req); err != nil {
		return nil, err
	}

	token, tokenHash, err := newInvitationToken()
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	invitation := &worker.PortalInvitation{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		WorkerID:       req.WorkerID,
		Email:          inviteEmail,
		TokenHash:      tokenHash,
		Status:         worker.PortalInvitationStatusPending,
		ExpiresAt:      now + invitationTTLSeconds,
		InvitedByID:    actor.UserID,
	}
	multiErr := errortypes.NewMultiError()
	invitation.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.portalRepo.CreateInvitation(ctx, invitation)
	if err != nil {
		return nil, err
	}

	inviteURL := s.inviteURL(token)
	emailSent := s.sendInvitationEmail(ctx, &invitationEmailParams{
		TenantInfo: req.TenantInfo,
		Worker:     wrk,
		To:         inviteEmail,
		InviteURL:  inviteURL,
		ExpiresAt:  created.ExpiresAt,
		Now:        now,
	})

	s.logPortalAudit(
		created.ID,
		req.TenantInfo,
		actor.UserID,
		permission.OpCreate,
		fmt.Sprintf("Portal invitation sent to %s", inviteEmail),
		created,
	)

	return &InviteWorkerResult{
		Invitation: created,
		InviteURL:  inviteURL,
		EmailSent:  emailSent,
	}, nil
}

func resolveInviteEmail(override, workerEmail string) (string, error) {
	inviteEmail := strings.TrimSpace(strings.ToLower(override))
	if inviteEmail == "" {
		inviteEmail = strings.TrimSpace(strings.ToLower(workerEmail))
	}
	if inviteEmail == "" {
		return "", errortypes.NewValidationError(
			"email",
			errortypes.ErrRequired,
			"The driver has no email address on file; provide one to send the invitation",
		)
	}
	return inviteEmail, nil
}

func (s *Service) ensureNoPendingInvitation(
	ctx context.Context,
	req *InviteWorkerRequest,
) error {
	existing, err := s.portalRepo.GetPendingInvitationForWorker(
		ctx,
		repositories.GetPendingPortalInvitationRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if existing.IsAcceptable(timeutils.NowUnix()) {
		return errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalidOperation,
			"An invitation is already pending for this driver; revoke it before sending a new one",
		)
	}
	return nil
}

func (s *Service) RevokeAccess(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	actor *serviceports.RequestActor,
) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError("Portal revocation requires an authenticated user")
	}
	if err := s.portalRepo.RevokePortalAccess(ctx, tenantInfo, workerID); err != nil {
		return err
	}
	s.logPortalAudit(
		workerID,
		tenantInfo,
		actor.UserID,
		permission.OpCancel,
		"Portal access revoked",
		nil,
	)
	return nil
}

func (s *Service) GetPortalStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*PortalStatus, error) {
	wrk, err := s.portalRepo.GetWorkerForPortalManagement(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}

	invitations, err := s.portalRepo.ListInvitations(
		ctx,
		&repositories.ListPortalInvitationsRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
		},
	)
	if err != nil {
		return nil, err
	}

	status := &PortalStatus{
		Linked:      !wrk.UserID.IsNil(),
		Invitations: invitations,
	}
	now := timeutils.NowUnix()
	for _, invitation := range invitations {
		if invitation.IsAcceptable(now) {
			status.PendingInvitation = invitation
			break
		}
	}
	if status.Linked {
		status.PortalUser = wrk.PortalUser
	}
	return status, nil
}

func (s *Service) GetInvitationPreview(
	ctx context.Context,
	token string,
) (*InvitationPreview, error) {
	invitation, err := s.lookupAcceptableInvitation(ctx, token)
	if err != nil {
		return nil, err
	}

	preview := &InvitationPreview{
		Email:     invitation.Email,
		ExpiresAt: invitation.ExpiresAt,
	}
	if invitation.Worker != nil {
		preview.WorkerFirstName = invitation.Worker.FirstName
		if invitation.Worker.Organization != nil {
			preview.OrganizationName = invitation.Worker.Organization.Name
		}
	}
	return preview, nil
}

func (s *Service) AcceptInvitation(
	ctx context.Context,
	req *AcceptInvitationRequest,
) (*AcceptInvitationResult, error) {
	if len(req.Password) < 8 {
		return nil, errortypes.NewValidationError(
			"password",
			errortypes.ErrInvalid,
			"Password must be at least 8 characters",
		)
	}

	invitation, err := s.lookupAcceptableInvitation(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	if invitation.Worker == nil {
		return nil, errortypes.NewValidationError(
			"token",
			errortypes.ErrInvalid,
			"This invitation is no longer valid. Ask your carrier to send a new one.",
		)
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "America/New_York"
	}

	user := &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        invitation.BusinessUnitID,
		CurrentOrganizationID: invitation.OrganizationID,
		Status:                domaintypes.StatusActive,
		Name: strings.TrimSpace(
			invitation.Worker.FirstName + " " + invitation.Worker.LastName,
		),
		Username:     usernameFromEmail(invitation.Email),
		EmailAddress: invitation.Email,
		Timezone:     timezone,
	}
	user.Password, err = user.GeneratePassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash portal password: %w", err)
	}

	created, err := s.portalRepo.ActivatePortalAccess(
		ctx,
		&repositories.ActivatePortalAccessRequest{
			Invitation:      invitation,
			User:            user,
			RoleName:        DriverRoleName,
			RoleDescription: driverRoleDescription,
		},
	)
	if err != nil {
		return nil, err
	}

	result := &AcceptInvitationResult{EmailAddress: created.EmailAddress}
	if invitation.Worker.Organization != nil {
		result.OrganizationName = invitation.Worker.Organization.Name
	}
	return result, nil
}

func (s *Service) lookupAcceptableInvitation(
	ctx context.Context,
	token string,
) (*worker.PortalInvitation, error) {
	if token == "" {
		return nil, errortypes.NewValidationError(
			"token",
			errortypes.ErrRequired,
			"An invitation token is required",
		)
	}
	invitation, err := s.portalRepo.GetInvitationByTokenHash(ctx, hashInvitationToken(token))
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"token",
				errortypes.ErrInvalid,
				"This invitation is no longer valid. Ask your carrier to send a new one.",
			)
		}
		return nil, err
	}
	if !invitation.IsAcceptable(timeutils.NowUnix()) {
		return nil, errortypes.NewValidationError(
			"token",
			errortypes.ErrInvalid,
			"This invitation has expired or was revoked. Ask your carrier to send a new one.",
		)
	}
	return invitation, nil
}

func (s *Service) inviteURL(token string) string {
	base := s.cfg.Portal.GetBaseURL()
	if base == "" {
		return "/dash/accept?token=" + token
	}
	return base + "/dash/accept?token=" + token
}

// invitationEmailParams is one invitation on its way out.
type invitationEmailParams struct {
	TenantInfo pagination.TenantInfo
	Worker     *worker.Worker
	To         string
	InviteURL  string
	ExpiresAt  int64
	Now        int64
}

// sendInvitationEmail renders the invitation through the organization's template
// and mails it.
//
// The wording and the layout used to be a fmt.Sprintf in this file, which meant a
// carrier could not change a word of the first thing their drivers ever receive
// from them. It also interpolated the invite link into the HTML raw while
// hand-escaping four characters of the driver's name beside it; html/template
// escapes both correctly and by position.
//
// A render failure falls back to the shipped default rather than abandoning the
// invitation: the driver cannot onboard without this email, and the caller is
// told whether it went so an admin can share the link by hand.
func (s *Service) sendInvitationEmail(
	ctx context.Context,
	p *invitationEmailParams,
) bool {
	log := s.l.With(zap.String("workerId", p.Worker.ID.String()))

	if s.templates == nil {
		log.Warn("template rendering is not configured; share the invite link manually")
		return false
	}

	rendered, err := s.templates.RenderMessage(ctx, &serviceports.RenderMessageRequest{
		TenantInfo: p.TenantInfo,
		Kind:       documenttemplate.KindDriverPortalInvitationEmail,
		Data: invitationContext(
			p.Worker,
			p.InviteURL,
			p.ExpiresAt,
			p.Now,
		),
		ReferenceID:       p.Worker.ID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		log.Warn("failed to render the portal invitation email; share the invite link manually",
			zap.Error(err))
		return false
	}

	if _, err = s.emailService.Send(ctx, &serviceports.SendEmailRequest{
		TenantInfo:     p.TenantInfo,
		Purpose:        email.PurposeGeneral,
		To:             []string{p.To},
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: "portal-invite-" + hashInvitationToken(p.InviteURL),
	}); err != nil {
		log.Warn("failed to send portal invitation email; share the invite link manually",
			zap.Error(err))
		return false
	}

	return true
}

func newInvitationToken() (token, tokenHash string, err error) {
	return tokenutils.New()
}

func hashInvitationToken(token string) string {
	return tokenutils.Hash(token)
}

func usernameFromEmail(address string) string {
	local := address
	if at := strings.IndexByte(address, '@'); at > 0 {
		local = address[:at]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	username := b.String()
	if username == "" {
		username = "driver"
	}
	if len(username) > 20 {
		username = username[:20]
	}
	return strings.TrimRight(username, "-_.")
}

func (s *Service) logPortalAudit(
	resourceID pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
	state any,
) {
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceDriverPortal,
		ResourceID:     resourceID.String(),
		Operation:      operation,
		UserID:         userID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
	if state != nil {
		params.CurrentState = jsonutils.MustToJSON(state)
	}
	if err := s.auditService.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log driver portal audit action", zap.Error(err))
	}
}
