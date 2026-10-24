package userservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/useragentutils"
	"go.uber.org/zap"
)

const (
	maxAvatarLookup     = 100
	defaultSignInWindow = 5
)

type AvatarSummary struct {
	ID           pulid.ID                  `json:"id"`
	Name         string                    `json:"name"`
	Username     string                    `json:"username"`
	AvatarStyle  tenant.AvatarStyle        `json:"avatarStyle"`
	Availability tenant.AvailabilityStatus `json:"availability"`
	AwayBackOn   *int64                    `json:"awayBackOn"`
	AwayNote     string                    `json:"awayNote"`
}

type SessionSummary struct {
	ID             pulid.ID                  `json:"id"`
	Label          string                    `json:"label"`
	Browser        string                    `json:"browser"`
	OS             string                    `json:"os"`
	Device         useragentutils.DeviceKind `json:"device"`
	IPAddress      string                    `json:"ipAddress"`
	AuthProvider   string                    `json:"authProvider"`
	CreatedAt      int64                     `json:"createdAt"`
	LastAccessedAt int64                     `json:"lastAccessedAt"`
	ExpiresAt      int64                     `json:"expiresAt"`
	Current        bool                      `json:"current"`
}

type SignInSummary struct {
	ID         pulid.ID                  `json:"id"`
	Provider   string                    `json:"provider"`
	Outcome    iam.AuthEventOutcome      `json:"outcome"`
	MFAState   string                    `json:"mfaState"`
	ErrorCode  string                    `json:"errorCode"`
	IPAddress  string                    `json:"ipAddress"`
	Label      string                    `json:"label"`
	Device     useragentutils.DeviceKind `json:"device"`
	OccurredAt int64                     `json:"occurredAt"`
}

type SessionScope struct {
	TenantInfo       pagination.TenantInfo
	CurrentSessionID pulid.ID
}

type columnUpdate struct {
	tenantInfo pagination.TenantInfo
	user       *tenant.User
	original   *tenant.User
	columns    []string
	comment    string
}

func (s *Service) updateColumns(ctx context.Context, req *columnUpdate) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "updateColumns"),
		zap.String("userID", req.user.ID.String()),
	)

	updated, err := s.repo.UpdateColumns(ctx, tenantports.UpdateUserColumnsRequest{
		Entity:  req.user,
		Columns: req.columns,
	})
	if err != nil {
		log.Error("failed to update user", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceUser,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         req.tenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(req.original),
		OrganizationID: req.tenantInfo.OrgID,
		BusinessUnitID: req.tenantInfo.BuID,
	},
		auditservice.WithComment(req.comment),
		auditservice.WithCategory(audit.CategoryUser),
		auditservice.WithDiff(req.original, updated),
		auditservice.WithRequest(ctx),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.tenantInfo.OrgID,
		BusinessUnitID: req.tenantInfo.BuID,
		ActorUserID:    req.tenantInfo.UserID,
		Resource:       permission.ResourceUser.String(),
		Action:         "updated",
		RecordID:       updated.ID,
		Entity:         updated,
	}); err != nil {
		log.Warn("failed to publish user invalidation", zap.Error(err))
	}

	return s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:         req.tenantInfo,
		LookupUserID:       updated.ID,
		IncludeMemberships: true,
	})
}

func (s *Service) loadSelf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.User, *tenant.User, error) {
	user, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, nil, err
	}
	original := *user
	return user, &original, nil
}

func (s *Service) UpdateMyProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req tenant.ProfileUpdate,
) (*tenant.User, error) {
	req.Normalize()
	if req.IsEmpty() {
		return nil, errortypes.NewValidationError(
			"profile",
			errortypes.ErrRequired,
			"Nothing to update",
		)
	}

	multiErr := errortypes.NewMultiError()
	req.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	user, original, err := s.loadSelf(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	cols := tenantdb.Users
	columns := make([]string, 0, 3)
	if req.JobTitle != nil {
		user.JobTitle = *req.JobTitle
		columns = append(columns, columnName(cols.JobTitle.Col))
	}
	if req.PhoneNumber != nil {
		user.PhoneNumber = *req.PhoneNumber
		columns = append(columns, columnName(cols.PhoneNumber.Col))
	}
	if req.AvatarStyle != nil {
		user.AvatarStyle = *req.AvatarStyle
		columns = append(columns, columnName(cols.AvatarStyle.Col))
	}

	return s.updateColumns(ctx, &columnUpdate{
		tenantInfo: tenantInfo,
		user:       user,
		original:   original,
		columns:    columns,
		comment:    "Profile updated",
	})
}

func (s *Service) SetMyAssistantHidden(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	hidden bool,
) (*tenant.User, error) {
	user, original, err := s.loadSelf(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	if user.AssistantHidden == hidden {
		return user, nil
	}

	user.AssistantHidden = hidden
	comment := "Assistant shown"
	if hidden {
		comment = "Assistant hidden"
	}

	return s.updateColumns(ctx, &columnUpdate{
		tenantInfo: tenantInfo,
		user:       user,
		original:   original,
		columns:    []string{columnName(tenantdb.Users.AssistantHidden.Col)},
		comment:    comment,
	})
}

func (s *Service) UpdateMyAvailability(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req tenant.AvailabilityUpdate,
) (*tenant.User, error) {
	req.Normalize()

	multiErr := errortypes.NewMultiError()
	req.Validate(tenantInfo.UserID, timeutils.NowUnix(), multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if req.DelegateUserID != nil {
		if err := s.ensureDelegate(ctx, tenantInfo, *req.DelegateUserID); err != nil {
			return nil, err
		}
	}

	user, original, err := s.loadSelf(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	user.ApplyAvailability(&req)

	cols := tenantdb.Users
	return s.updateColumns(ctx, &columnUpdate{
		tenantInfo: tenantInfo,
		user:       user,
		original:   original,
		columns: []string{
			columnName(cols.AvailabilityStatus.Col),
			columnName(cols.FocusUntil.Col),
			columnName(cols.AwayBackOn.Col),
			columnName(cols.AwayDelegateUserID.Col),
			columnName(cols.AwayNote.Col),
		},
		comment: "Availability updated",
	})
}

func (s *Service) ensureDelegate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	delegateID pulid.ID,
) error {
	found, err := s.repo.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    []pulid.ID{delegateID},
	})
	if err != nil {
		return err
	}
	if len(found) == 0 || !found[0].IsActive() || found[0].IsLocked {
		return errortypes.NewValidationError(
			"delegateUserId",
			errortypes.ErrInvalid,
			"Hand off to an active member of your organization",
		)
	}
	return nil
}

func (s *Service) ListAvatars(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]AvatarSummary, error) {
	unique := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if id.IsNil() || slices.Contains(unique, id) {
			continue
		}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return []AvatarSummary{}, nil
	}
	if len(unique) > maxAvatarLookup {
		return nil, errortypes.NewValidationError(
			"ids",
			errortypes.ErrInvalid,
			"Ask for at most 100 people at a time",
		)
	}

	users, err := s.repo.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    unique,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	out := make([]AvatarSummary, 0, len(users))
	for _, user := range users {
		summary := AvatarSummary{
			ID:           user.ID,
			Name:         user.Name,
			Username:     user.Username,
			AvatarStyle:  user.AvatarStyle,
			Availability: user.EffectiveAvailability(now),
		}
		if summary.AvatarStyle == "" {
			summary.AvatarStyle = tenant.AvatarStyleAuto
		}
		if summary.Availability == tenant.AvailabilityAway {
			summary.AwayBackOn = user.AwayBackOn
			summary.AwayNote = user.AwayNote
		}
		out = append(out, summary)
	}

	return out, nil
}

func (s *Service) ListMySessions(ctx context.Context, scope SessionScope) ([]SessionSummary, error) {
	sessions, err := s.sr.ListForUser(ctx, scope.TenantInfo.UserID)
	if err != nil {
		return nil, err
	}

	out := make([]SessionSummary, 0, len(sessions))
	for _, sess := range sessions {
		described := useragentutils.Describe(sess.UserAgent)
		out = append(out, SessionSummary{
			ID:             sess.ID,
			Label:          described.Label(),
			Browser:        described.Browser,
			OS:             described.OS,
			Device:         described.Device,
			IPAddress:      sess.IPAddress,
			AuthProvider:   sess.AuthProvider,
			CreatedAt:      sess.CreatedAt,
			LastAccessedAt: sess.LastAccessedAt,
			ExpiresAt:      sess.ExpiresAt,
			Current:        sess.ID == scope.CurrentSessionID,
		})
	}

	slices.SortStableFunc(out, func(a, b SessionSummary) int {
		switch {
		case a.Current != b.Current:
			if a.Current {
				return -1
			}
			return 1
		case a.LastAccessedAt != b.LastAccessedAt:
			if a.LastAccessedAt > b.LastAccessedAt {
				return -1
			}
			return 1
		default:
			return 0
		}
	})

	return out, nil
}

func (s *Service) RevokeMySession(
	ctx context.Context,
	scope SessionScope,
	sessionID pulid.ID,
) error {
	if sessionID == scope.CurrentSessionID {
		return errortypes.NewValidationError(
			"sessionId",
			errortypes.ErrInvalid,
			"Use Log out to end the session you are using",
		)
	}

	sessions, err := s.sr.ListForUser(ctx, scope.TenantInfo.UserID)
	if err != nil {
		return err
	}
	owned := slices.ContainsFunc(sessions, func(sess *session.Session) bool {
		return sess.ID == sessionID
	})
	if !owned {
		return errortypes.NewNotFoundError("Session not found")
	}

	if err = s.sr.Delete(ctx, sessionID); err != nil {
		return err
	}

	s.recordRevocation(ctx, scope.TenantInfo, 1)
	return nil
}

func (s *Service) RevokeMyOtherSessions(ctx context.Context, scope SessionScope) (int, error) {
	sessions, err := s.sr.ListForUser(ctx, scope.TenantInfo.UserID)
	if err != nil {
		return 0, err
	}

	revoked := 0
	for _, sess := range sessions {
		if sess.ID == scope.CurrentSessionID {
			continue
		}
		if err = s.sr.Delete(ctx, sess.ID); err != nil {
			s.l.Warn(
				"failed to revoke session",
				zap.String("sessionID", sess.ID.String()),
				zap.Error(err),
			)
			continue
		}
		revoked++
	}

	if revoked > 0 {
		s.recordRevocation(ctx, scope.TenantInfo, revoked)
	}
	return revoked, nil
}

func (s *Service) recordRevocation(ctx context.Context, tenantInfo pagination.TenantInfo, count int) {
	if s.authEvents == nil {
		return
	}
	mfaState := "single"
	if count > 1 {
		mfaState = "others"
	}
	s.authEvents.Record(ctx, &iamports.AuthEventRecord{
		Provider:       iamports.AuthEventProviderSessionRevoked,
		Outcome:        iam.AuthEventOutcomeSuccess,
		UserID:         tenantInfo.UserID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		MFAState:       mfaState,
	})
}

func (s *Service) ListMySignIns(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]SignInSummary, error) {
	if s.signIns == nil {
		return []SignInSummary{}, nil
	}
	if limit <= 0 {
		limit = defaultSignInWindow
	}

	events, err := s.signIns.ListSignInsForUser(ctx, iamports.ListUserSignInsRequest{
		TenantInfo: tenantInfo,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]SignInSummary, 0, len(events))
	for _, event := range events {
		described := useragentutils.Describe(event.UserAgent)
		out = append(out, SignInSummary{
			ID:         event.ID,
			Provider:   event.Provider,
			Outcome:    event.Outcome,
			MFAState:   event.MFAState,
			ErrorCode:  event.ErrorCode,
			IPAddress:  event.IPAddress,
			Label:      described.Label(),
			Device:     described.Device,
			OccurredAt: event.OccurredAt,
		})
	}

	return out, nil
}
