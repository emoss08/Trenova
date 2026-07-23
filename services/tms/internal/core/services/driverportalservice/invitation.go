package driverportalservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

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
	emailSent := s.sendInvitationEmail(ctx, req.TenantInfo, wrk, inviteEmail, inviteURL)

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

func (s *Service) sendInvitationEmail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
	to string,
	inviteURL string,
) bool {
	orgName := "your carrier"
	if wrk.Organization != nil && wrk.Organization.Name != "" {
		orgName = wrk.Organization.Name
	}

	subject := fmt.Sprintf("%s invited you to Dash", orgName)
	html := invitationEmailHTML(wrk.FirstName, orgName, inviteURL)
	text := fmt.Sprintf(
		"Hi %s,\n\n%s has invited you to Dash, your driver portal. "+
			"See your loads, settlements, and pay history, and raise questions about your pay.\n\n"+
			"Set up your account: %s\n\nThis link expires in 7 days.",
		wrk.FirstName, orgName, inviteURL,
	)

	_, err := s.emailService.Send(ctx, &serviceports.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeGeneral,
		To:             []string{to},
		Subject:        subject,
		HTML:           html,
		Text:           text,
		IdempotencyKey: "portal-invite-" + hashInvitationToken(inviteURL),
	})
	if err != nil {
		s.l.Warn("failed to send portal invitation email; share the invite link manually",
			zap.String("workerId", wrk.ID.String()),
			zap.Error(err))
		return false
	}
	return true
}

func invitationEmailHTML(firstName, orgName, inviteURL string) string {
	return fmt.Sprintf(
		`<div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:520px;margin:0 auto;padding:32px 24px;color:#18181b;">
  <h1 style="font-size:20px;margin:0 0 16px;">You're invited to Dash</h1>
  <p style="font-size:15px;line-height:1.6;margin:0 0 12px;">Hi %s,</p>
  <p style="font-size:15px;line-height:1.6;margin:0 0 20px;">%s has invited you to <strong>Dash</strong>, your driver portal. See your loads, settlement statements, and pay history &mdash; and raise a question about your pay right from your phone.</p>
  <p style="margin:0 0 24px;"><a href="%s" style="display:inline-block;background:#18181b;color:#ffffff;text-decoration:none;padding:12px 24px;border-radius:8px;font-size:15px;font-weight:600;">Set up your account</a></p>
  <p style="font-size:13px;line-height:1.6;color:#71717a;margin:0;">This link expires in 7 days. If the button doesn't work, copy this link into your browser:<br>%s</p>
</div>`,
		htmlEscape(firstName),
		htmlEscape(orgName),
		inviteURL,
		inviteURL,
	)
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(s)
}

func newInvitationToken() (token, tokenHash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate invitation token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, hashInvitationToken(token), nil
}

func hashInvitationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
