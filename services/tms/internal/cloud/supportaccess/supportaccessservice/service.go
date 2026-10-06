package supportaccessservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	supportctx "github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessrepository"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	touchInterval    = 60
	customerListSize = 25
	historyListSize  = 10
)

var (
	errNotAvailable = errortypes.NewNotFoundError("Trenova support access is not available")
	errNotStaff     = errortypes.NewAuthorizationError(
		"Only Trenova platform staff can open support sessions",
	)
	errMFARequired = errortypes.NewAuthorizationError(
		"Sign in with two-factor authentication before opening a support session",
	)
	errNoGrant = errortypes.NewAuthorizationError(
		"This organization has not allowed Trenova support access, or its access has ended",
	)
	errReadOnlyGrant = errortypes.NewBusinessError(
		"This organization allowed read-only support access, so the session cannot be elevated",
	)
	errNoOpenGrant = errortypes.NewBusinessError("Trenova support access is not currently allowed")

	presetDurationHours = []int{24, 72, 168, 336}
)

type Notifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

type Params struct {
	fx.In

	Repository *supportaccessrepository.Repository
	Users      repositories.UserRepository
	MFA        services.MFAService
	Auditor    services.SecurityAuditor
	Limiter    StartLimiter
	Notifier   *notificationservice.Service `optional:"true"`
	Config     *config.Config
	Logger     *zap.Logger
}

type Service struct {
	repo     store
	users    repositories.UserRepository
	mfa      services.MFAService
	auditor  services.SecurityAuditor
	limiter  StartLimiter
	notifier Notifier
	platform *config.PlatformConfig
	settings *cloudconfig.CloudSupportAccessConfig
	l        *zap.Logger
	now      func() time.Time
}

//nolint:gocritic // fx parameter objects are passed by value
func New(p Params) *Service {
	svc := &Service{
		repo:     p.Repository,
		users:    p.Users,
		mfa:      p.MFA,
		auditor:  p.Auditor,
		limiter:  p.Limiter,
		platform: &p.Config.Platform,
		settings: &cloudconfig.From(p.Config).Cloud.SupportAccess,
		l:        p.Logger.Named("service.support-access"),
		now:      time.Now,
	}
	if p.Notifier != nil {
		svc.notifier = p.Notifier
	}

	return svc
}

func (s *Service) Enabled() bool {
	return s.platform.IsCloud() && s.settings.IsEnabled()
}

func (s *Service) CookieName() string {
	return s.settings.GetCookieName()
}

func (s *Service) nowUnix() int64 {
	return s.now().Unix()
}

func (s *Service) requireEnabled() error {
	if !s.Enabled() {
		return errNotAvailable
	}
	return nil
}

func (s *Service) requireStaff(ctx context.Context, staff *StaffContext) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}
	if staff == nil {
		return errNotStaff
	}

	member, err := s.repo.GetActiveStaffMember(staff.scope(ctx), staff.UserID)
	if err != nil {
		return err
	}
	if member == nil {
		return errNotStaff
	}

	return nil
}

func (s *Service) StaffProfile(ctx context.Context, staff *StaffContext) (*StaffProfile, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}

	profile := &StaffProfile{
		OpenSessions:     []*SessionView{},
		MaxSessionHours:  s.settings.GetMaxSessionDuration().Hours(),
		ElevationMinutes: s.settings.GetElevationDuration().Minutes(),
	}

	member, err := s.repo.GetActiveStaffMember(staff.scope(ctx), staff.UserID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return profile, nil
	}

	profile.IsStaff = true
	profile.Role = member.Role.String()
	profile.SessionVerified = staff.MFAVerified()

	if profile.MFAEnrolled, err = s.mfa.HasActiveFactor(
		staff.scope(ctx),
		staff.UserID,
	); err != nil {
		return nil, err
	}

	now := s.nowUnix()
	open, err := s.repo.ListOpenSessionsForStaff(ctx, staff.UserID, now)
	if err != nil {
		return nil, err
	}
	for _, session := range open {
		profile.OpenSessions = append(profile.OpenSessions, s.sessionView(session, now))
	}

	return profile, nil
}

func (s *Service) ListGrantedOrganizations(
	ctx context.Context,
	staff *StaffContext,
) ([]*supportaccessrepository.GrantedOrganization, error) {
	if err := s.requireStaff(ctx, staff); err != nil {
		return nil, err
	}

	return s.repo.ListGrantedOrganizations(ctx, s.nowUnix())
}

func (s *Service) requireAssurance(ctx context.Context, staff *StaffContext) error {
	if !staff.MFAVerified() {
		return errMFARequired
	}

	enrolled, err := s.mfa.HasActiveFactor(staff.scope(ctx), staff.UserID)
	if err != nil {
		return err
	}
	if !enrolled {
		return errMFARequired
	}

	return nil
}

func (s *Service) checkStart(ctx context.Context, req *StartSessionRequest) error {
	if err := s.requireStaff(ctx, req.Staff); err != nil {
		return err
	}
	if err := s.requireAssurance(ctx, req.Staff); err != nil {
		return err
	}
	if err := validateStart(req); err != nil {
		return err
	}

	allowed, retryAfter, err := s.limiter.Allow(
		ctx,
		req.Staff.UserID,
		s.settings.GetSessionStartsPerHour(),
	)
	if err != nil {
		return err
	}
	if !allowed {
		return errortypes.NewRateLimitError(
			"organizationId",
			"Too many support sessions were opened in the last hour. Try again later.",
		).WithRetryAfter(retryAfter)
	}

	return nil
}

func (s *Service) preparePrincipal(
	ctx, targetCtx context.Context,
	req *StartSessionRequest,
	now int64,
) (staffUser, principal *tenant.User, err error) {
	staffUser, err = s.users.FindByIDForLogin(ctx, req.Staff.UserID)
	if err != nil {
		return nil, nil, err
	}

	principalUser, err := newPrincipalUser(&principalUserParams{
		Staff:          staffUser,
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		EmailDomain:    s.settings.GetPrincipalEmailDomain(),
		Now:            now,
	})
	if err != nil {
		return nil, nil, err
	}

	principal, err = s.repo.EnsurePrincipal(
		targetCtx,
		&supportaccessrepository.EnsurePrincipalRequest{
			TenantInfo:  pagination.TenantInfo{OrgID: req.OrganizationID, BuID: req.BusinessUnitID},
			StaffUserID: req.Staff.UserID,
			NewUser:     principalUser,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	return staffUser, principal, nil
}

func (s *Service) StartSession(
	ctx context.Context,
	req *StartSessionRequest,
) (*StartedSession, error) {
	if err := s.checkStart(ctx, req); err != nil {
		return nil, err
	}

	now := s.nowUnix()
	targetCtx := tenantScope(ctx, req.OrganizationID, req.BusinessUnitID, req.Staff.UserID)
	grant, err := s.repo.GetOpenGrant(targetCtx, pagination.TenantInfo{
		OrgID: req.OrganizationID,
		BuID:  req.BusinessUnitID,
	}, now)
	if err != nil {
		return nil, err
	}
	if !grant.IsActive(now) {
		return nil, errNoGrant
	}

	staffUser, principal, err := s.preparePrincipal(ctx, targetCtx, req, now)
	if err != nil {
		return nil, err
	}

	if _, err = s.repo.EndSessionsForStaff(ctx, supportaccessrepository.EndStaffSessionsRequest{
		StaffUserID: req.Staff.UserID,
		Reason:      supportaccess.EndReasonReplaced,
		Now:         now,
	}); err != nil {
		return nil, err
	}

	sessionID := pulid.MustNew("sps_")
	token, err := supportctx.IssueToken(req.OrganizationID, req.BusinessUnitID, sessionID)
	if err != nil {
		return nil, err
	}

	session := &supportaccess.Session{
		ID:              sessionID,
		OrganizationID:  req.OrganizationID,
		BusinessUnitID:  req.BusinessUnitID,
		GrantID:         grant.ID,
		StaffUserID:     req.Staff.UserID,
		StaffName:       truncate(staffUser.Name, maxNameLength),
		PrincipalUserID: principal.ID,
		BaseSessionID:   req.Staff.SessionID,
		SecretHash:      token.SecretHash,
		Reason:          strings.TrimSpace(req.Reason),
		TicketReference: strings.TrimSpace(req.TicketReference),
		StartedAt:       now,
		ExpiresAt: min(
			now+int64(s.settings.GetMaxSessionDuration().Seconds()),
			grant.ExpiresAt,
		),
		LastSeenAt: now,
		ClientIP:   truncate(req.ClientIP, supportaccess.MaxClientIPLength),
		UserAgent:  truncate(req.UserAgent, supportaccess.MaxUserAgentLength),
	}

	multiErr := errortypes.NewMultiError()
	session.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if err = s.repo.CreateSession(targetCtx, session); err != nil {
		return nil, err
	}

	session.GrantAccessMode = grant.AccessMode.String()
	session.GrantExpiresAt = grant.ExpiresAt

	s.recordSessionEvent(ctx, &sessionEvent{
		session:   session,
		operation: sessionStarted,
		comment: "Trenova support session started by " + session.StaffName + ": " +
			session.Reason + ticketSuffix(session.TicketReference),
	})
	s.notifyAdmins(ctx, session, grant)

	return &StartedSession{Session: s.sessionView(session, now), Token: token.Value}, nil
}

func (s *Service) Resolve(
	ctx context.Context,
	staff *StaffContext,
	rawToken string,
) (*supportctx.Active, *supportaccess.Session, error) {
	if !s.Enabled() {
		return nil, nil, &SessionEndedError{Reason: supportaccess.EndReasonExpired}
	}

	token, err := supportctx.ParseToken(rawToken)
	if err != nil {
		return nil, nil, &SessionEndedError{Reason: supportaccess.EndReasonExpired}
	}

	key := supportaccessrepository.SessionKey{
		OrganizationID: token.OrganizationID,
		BusinessUnitID: token.BusinessUnitID,
		SessionID:      token.SessionID,
	}
	targetCtx := tenantScope(ctx, token.OrganizationID, token.BusinessUnitID, staff.UserID)

	session, err := s.repo.GetSession(targetCtx, key)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil, &SessionEndedError{Reason: supportaccess.EndReasonExpired}
		}
		return nil, nil, err
	}
	if !token.Matches(session.SecretHash) || session.StaffUserID != staff.UserID {
		return nil, nil, &SessionEndedError{Reason: supportaccess.EndReasonExpired}
	}
	if session.EndedAt != nil {
		return nil, nil, &SessionEndedError{Reason: session.EndReason}
	}

	now := s.nowUnix()
	if reason, ended := s.endReason(ctx, staff, session, now); ended {
		s.endSession(targetCtx, session, reason, now)
		return nil, nil, &SessionEndedError{Reason: reason}
	}

	if now-session.LastSeenAt >= touchInterval {
		if touchErr := s.repo.TouchSession(targetCtx, key, now); touchErr != nil {
			s.l.Warn("failed to record support session activity", zap.Error(touchErr))
		}
	}

	active := &supportctx.Active{
		SessionID:       session.ID,
		OrganizationID:  session.OrganizationID,
		BusinessUnitID:  session.BusinessUnitID,
		GrantID:         session.GrantID,
		StaffUserID:     session.StaffUserID,
		StaffName:       session.StaffName,
		PrincipalUserID: session.PrincipalUserID,
		BaseSessionID:   session.BaseSessionID,
		GrantMode:       supportaccess.AccessMode(session.GrantAccessMode),
		ExpiresAt:       session.ExpiresAt,
		Now:             now,
	}
	if session.IsElevated(now) {
		active.ElevatedUntil = *session.ElevatedUntil
	}

	return active, session, nil
}

func (s *Service) endReason(
	ctx context.Context,
	staff *StaffContext,
	session *supportaccess.Session,
	now int64,
) (supportaccess.EndReason, bool) {
	switch {
	case session.BaseSessionID != staff.SessionID:
		return supportaccess.EndReasonSignedOut, true
	case !staff.MFAVerified():
		return supportaccess.EndReasonAssuranceLost, true
	case session.GrantRevokedAt != nil:
		return supportaccess.EndReasonGrantRevoked, true
	case now >= session.GrantExpiresAt:
		return supportaccess.EndReasonGrantExpired, true
	case now >= session.ExpiresAt:
		return supportaccess.EndReasonExpired, true
	}

	member, err := s.repo.GetActiveStaffMember(staff.scope(ctx), staff.UserID)
	if err != nil {
		s.l.Error("failed to confirm a support session's staff member; ending it", zap.Error(err))
		return supportaccess.EndReasonStaffRemoved, true
	}
	if member == nil {
		return supportaccess.EndReasonStaffRemoved, true
	}

	return "", false
}

func (s *Service) endSession(
	ctx context.Context,
	session *supportaccess.Session,
	reason supportaccess.EndReason,
	now int64,
) {
	ended, err := s.repo.EndSession(ctx, supportaccessrepository.EndSessionRequest{
		Key: supportaccessrepository.SessionKey{
			OrganizationID: session.OrganizationID,
			BusinessUnitID: session.BusinessUnitID,
			SessionID:      session.ID,
		},
		Reason: reason,
		Now:    now,
	})
	if err != nil {
		s.l.Error("failed to end a support session", zap.Error(err))
		return
	}
	if !ended {
		return
	}

	s.recordSessionEvent(ctx, &sessionEvent{
		session:   session,
		operation: sessionEnded,
		comment: "Trenova support session of " + session.StaffName + " ended: " + endReasonText(
			reason,
		),
		metadata: map[string]any{metadataEndReason: reason.String()},
	})
}

func (s *Service) CurrentSession(
	ctx context.Context,
	staff *StaffContext,
	rawToken string,
) (*CurrentSessionResponse, error) {
	if rawToken == "" {
		return &CurrentSessionResponse{}, nil
	}

	_, session, err := s.Resolve(ctx, staff, rawToken)
	if err != nil {
		var ended *SessionEndedError
		if asEnded(err, &ended) {
			return &CurrentSessionResponse{Ended: ended.Reason.String()}, nil
		}
		return nil, err
	}

	return &CurrentSessionResponse{Active: true, Session: s.sessionView(session, s.nowUnix())}, nil
}

func (s *Service) Elevate(
	ctx context.Context,
	staff *StaffContext,
	rawToken string,
	req *ElevateRequest,
) (*SessionView, error) {
	active, session, err := s.Resolve(ctx, staff, rawToken)
	if err != nil {
		return nil, err
	}
	if !active.GrantMode.AllowsWrite() {
		return nil, errReadOnlyGrant
	}
	if err = validateElevation(req); err != nil {
		return nil, err
	}

	if _, err = s.mfa.Reauthenticate(staff.scope(ctx), &services.ReauthenticateRequest{
		UserID:       staff.UserID,
		Password:     req.Password,
		Code:         req.Code,
		RecoveryCode: req.RecoveryCode,
	}); err != nil {
		return nil, err
	}

	now := s.nowUnix()
	until := min(now+int64(s.settings.GetElevationDuration().Seconds()), session.ExpiresAt)
	reason := strings.TrimSpace(req.Reason)
	ticket := strings.TrimSpace(req.TicketReference)
	targetCtx := tenantScope(ctx, session.OrganizationID, session.BusinessUnitID, staff.UserID)

	if err = s.repo.ElevateSession(targetCtx, &supportaccessrepository.ElevateSessionRequest{
		Key:           keyOf(session),
		ElevatedAt:    now,
		ElevatedUntil: until,
		Reason:        reason,
		Ticket:        ticket,
	}); err != nil {
		return nil, err
	}

	session.ElevatedAt = &now
	session.ElevatedUntil = &until
	session.ElevationReason = reason
	session.ElevationTicket = ticket

	s.recordSessionEvent(ctx, &sessionEvent{
		session:   session,
		operation: sessionElevated,
		comment: "Trenova support (" + session.StaffName + ") elevated to read-write until " +
			time.Unix(until, 0).UTC().Format(time.RFC3339) + ": " + reason + ticketSuffix(ticket),
		metadata: map[string]any{"elevatedUntil": until},
	})

	return s.sessionView(session, now), nil
}

func (s *Service) DropElevation(
	ctx context.Context,
	staff *StaffContext,
	rawToken string,
) (*SessionView, error) {
	active, session, err := s.Resolve(ctx, staff, rawToken)
	if err != nil {
		return nil, err
	}

	now := s.nowUnix()
	if !active.WriteActive() {
		return s.sessionView(session, now), nil
	}

	targetCtx := tenantScope(ctx, session.OrganizationID, session.BusinessUnitID, staff.UserID)
	if err = s.repo.DropElevation(targetCtx, keyOf(session), now); err != nil {
		return nil, err
	}
	session.ElevatedUntil = nil

	s.recordSessionEvent(ctx, &sessionEvent{
		session:   session,
		operation: sessionDemoted,
		comment:   "Trenova support (" + session.StaffName + ") returned to read-only",
	})

	return s.sessionView(session, now), nil
}

func (s *Service) EndSession(ctx context.Context, staff *StaffContext, rawToken string) error {
	token, err := supportctx.ParseToken(rawToken)
	if err != nil {
		return nil //nolint:nilerr // an unreadable cookie has nothing left to end
	}

	targetCtx := tenantScope(ctx, token.OrganizationID, token.BusinessUnitID, staff.UserID)
	session, err := s.repo.GetSession(targetCtx, supportaccessrepository.SessionKey{
		OrganizationID: token.OrganizationID,
		BusinessUnitID: token.BusinessUnitID,
		SessionID:      token.SessionID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if !token.Matches(session.SecretHash) || session.StaffUserID != staff.UserID ||
		session.EndedAt != nil {
		return nil
	}

	s.endSession(targetCtx, session, supportaccess.EndReasonExited, s.nowUnix())

	return nil
}

func (s *Service) GrantState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*GrantState, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}

	now := s.nowUnix()
	grant, err := s.repo.GetOpenGrant(ctx, tenantInfo, now)
	if err != nil {
		return nil, err
	}

	history, err := s.repo.ListGrants(ctx, tenantInfo, historyListSize)
	if err != nil {
		return nil, err
	}

	sessions, err := s.repo.ListSessions(ctx, tenantInfo, customerListSize)
	if err != nil {
		return nil, err
	}

	views := make([]*CustomerSessionView, 0, len(sessions))
	for _, session := range sessions {
		views = append(views, customerSessionView(session, now))
	}

	return &GrantState{
		Grant:            grant,
		Sessions:         views,
		History:          history,
		DurationHours:    s.durationChoices(),
		MaxDurationHours: int(s.settings.GetMaxGrantDuration().Hours()),
		AccessModes:      supportaccess.AllAccessModes(),
		ServerTime:       now,
	}, nil
}

func (s *Service) durationChoices() []int {
	maxHours := int(s.settings.GetMaxGrantDuration().Hours())
	choices := make([]int, 0, len(presetDurationHours))
	for _, hours := range presetDurationHours {
		if hours <= maxHours {
			choices = append(choices, hours)
		}
	}
	return choices
}

func (s *Service) CreateGrant(
	ctx context.Context,
	req *CreateGrantRequest,
) (*supportaccess.Grant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}

	maxDuration := s.settings.GetMaxGrantDuration()
	if req.DurationHours <= 0 || time.Duration(req.DurationHours)*time.Hour > maxDuration {
		return nil, errortypes.NewValidationError(
			"durationHours",
			errortypes.ErrInvalid,
			"Choose how long support access lasts, up to {0} days",
			int(maxDuration.Hours()/24),
		)
	}

	now := s.nowUnix()
	grant := &supportaccess.Grant{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		GrantedByID:    req.TenantInfo.UserID,
		AccessMode:     req.AccessMode,
		Note:           strings.TrimSpace(req.Note),
		StartsAt:       now,
		ExpiresAt:      now + int64(req.DurationHours)*int64(time.Hour.Seconds()),
	}

	multiErr := errortypes.NewMultiError()
	grant.Validate(multiErr, maxDuration)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	result, err := s.repo.ReplaceGrant(ctx, grant)
	if err != nil {
		return nil, err
	}

	s.recordGrantEvent(ctx, &grantEvent{
		tenantInfo: req.TenantInfo,
		grantID:    grant.ID,
		operation:  grantCreated,
		after:      grant,
		comment: "Allowed Trenova support access (" + accessModeText(grant.AccessMode) +
			") until " + time.Unix(grant.ExpiresAt, 0).UTC().Format(time.RFC3339),
		endedSessions: result.EndedSessionIDs,
	})

	return grant, nil
}

func (s *Service) RevokeGrant(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}

	now := s.nowUnix()
	open, err := s.repo.GetOpenGrant(ctx, tenantInfo, now)
	if err != nil {
		return err
	}
	if open == nil {
		return errNoOpenGrant
	}

	ended, err := s.repo.RevokeGrants(ctx, supportaccessrepository.RevokeGrantsRequest{
		TenantInfo: tenantInfo,
		RevokedBy:  tenantInfo.UserID,
		Now:        now,
	})
	if err != nil {
		return err
	}

	s.recordGrantEvent(ctx, &grantEvent{
		tenantInfo:    tenantInfo,
		grantID:       open.ID,
		operation:     grantRevoked,
		before:        open,
		comment:       "Revoked Trenova support access",
		endedSessions: ended,
	})

	return nil
}

func keyOf(session *supportaccess.Session) supportaccessrepository.SessionKey {
	return supportaccessrepository.SessionKey{
		OrganizationID: session.OrganizationID,
		BusinessUnitID: session.BusinessUnitID,
		SessionID:      session.ID,
	}
}

func (s *Service) sessionView(session *supportaccess.Session, now int64) *SessionView {
	grantMode := supportaccess.AccessMode(session.GrantAccessMode)
	view := &SessionView{
		ID:               session.ID,
		OrganizationID:   session.OrganizationID,
		BusinessUnitID:   session.BusinessUnitID,
		OrganizationName: session.OrganizationName,
		StaffName:        session.StaffName,
		Mode:             supportaccess.AccessModeReadOnly,
		GrantMode:        grantMode,
		Reason:           session.Reason,
		TicketReference:  session.TicketReference,
		StartedAt:        session.StartedAt,
		ExpiresAt:        session.ExpiresAt,
		CanElevate:       grantMode.AllowsWrite(),
		ServerTime:       now,
	}
	if grantMode.AllowsWrite() && session.IsElevated(now) {
		view.Mode = supportaccess.AccessModeReadWrite
		view.ElevatedUntil = *session.ElevatedUntil
	}

	return view
}

func customerSessionView(session *supportaccess.Session, now int64) *CustomerSessionView {
	view := &CustomerSessionView{
		ID:              session.ID,
		StaffName:       session.StaffName,
		Status:          session.Status(now),
		Mode:            supportaccess.AccessModeReadOnly,
		Reason:          session.Reason,
		TicketReference: session.TicketReference,
		StartedAt:       session.StartedAt,
		ExpiresAt:       session.ExpiresAt,
		LastSeenAt:      session.LastSeenAt,
		ElevationCount:  session.ElevationCount,
		ElevationReason: session.ElevationReason,
		EndReason:       session.EndReason.String(),
	}
	if session.EndedAt != nil {
		view.EndedAt = *session.EndedAt
	}
	if session.EndedAt == nil && session.IsElevated(now) {
		view.Mode = supportaccess.AccessModeReadWrite
		view.ElevatedUntil = *session.ElevatedUntil
	}

	return view
}
