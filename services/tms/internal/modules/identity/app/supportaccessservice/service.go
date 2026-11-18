package supportaccessservice

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	keySetTTL = 30 * time.Second
	keySetKey = supportaccess.CurrentKeySetID
)

var errNotAvailable = errortypes.NewNotFoundError(
	"Trenova support access is not available on this instance",
)

type Params struct {
	fx.In

	Grants     supportaccessports.GrantRepository
	Keys       supportaccessports.KeySetRepository
	Ledger     supportaccessports.AssertionLedger
	Sessions   sessionports.SessionRepository
	Tokens     *sessiontoken.Issuer
	Principals *Principals
	Auditor    permissionports.SecurityAuditor
	AuthEvents iamports.AuthEventRecorder
	Notifier   supportaccessports.Notifier
	Managed    *managedconfig.Config
	Security   *securityconfig.Config
	Clock      timeutils.Clock
	Logger     *zap.Logger
}

// Service is support access on a managed instance: administrators grant and revoke it, the
// operator delivers the control plane's keys, and staff enter with an assertion. With
// managed.enabled off every method refuses before reading anything.
type Service struct {
	grants     supportaccessports.GrantRepository
	keys       supportaccessports.KeySetRepository
	ledger     supportaccessports.AssertionLedger
	sessions   sessionports.SessionRepository
	tokens     *sessiontoken.Issuer
	principals *Principals
	auditor    permissionports.SecurityAuditor
	authEvents iamports.AuthEventRecorder
	notifier   supportaccessports.Notifier
	enabled    bool
	maxAge     time.Duration
	clock      timeutils.Clock
	keyCache   *otter.Cache[string, cachedKeySet]
	l          *zap.Logger
}

type cachedKeySet struct {
	keys *supportaccess.KeySet
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Service {
	return &Service{
		grants:     p.Grants,
		keys:       p.Keys,
		ledger:     p.Ledger,
		sessions:   p.Sessions,
		tokens:     p.Tokens,
		principals: p.Principals,
		auditor:    p.Auditor,
		authEvents: p.AuthEvents,
		notifier:   p.Notifier,
		enabled:    p.Managed.Enabled,
		maxAge:     p.Security.Session.MaxAge,
		clock:      p.Clock,
		keyCache: otter.Must(&otter.Options[string, cachedKeySet]{
			MaximumSize:      1,
			ExpiryCalculator: otter.ExpiryWriting[string, cachedKeySet](keySetTTL),
		}),
		l: p.Logger.Named("service.support-access"),
	}
}

// Enabled reports whether support access exists on this instance at all.
func (s *Service) Enabled() bool {
	return s.enabled
}

type Settings struct {
	Available     bool
	StaffDomain   string
	Durations     []int
	MaxLiveGrants int
	Limits        []supportaccess.Limit
}

// Settings says whether grants can be made here and what an administrator may choose. It reads
// nothing when support access is off.
func (s *Service) Settings(ctx context.Context) (*Settings, error) {
	settings := &Settings{
		Durations:     supportaccess.Durations,
		MaxLiveGrants: supportaccess.MaxLiveGrants,
		Limits:        supportaccess.Limits(),
	}
	if !s.enabled {
		return settings, nil
	}
	keys, err := s.keySet(ctx)
	if errors.Is(err, supportaccess.ErrNoKeySet) {
		return settings, nil
	}
	if err != nil {
		return nil, err
	}
	settings.Available = true
	settings.StaffDomain = keys.StaffDomain
	return settings, nil
}

type GrantRequest struct {
	TenantInfo      pagination.TenantInfo
	StaffEmail      string
	Reason          string
	DurationHours   int
	Scope           supportaccess.Scope
	AllowsAssistant bool
}

func (s *Service) Grant(ctx context.Context, req *GrantRequest) (*supportaccess.Grant, error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	if supportaccess.IsPrincipalID(req.TenantInfo.UserID) || req.TenantInfo.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Only a person in this organization may grant support access",
		)
	}

	keys, err := s.keySet(ctx)
	if errors.Is(err, supportaccess.ErrNoKeySet) {
		return nil, errortypes.NewBusinessError(
			"Support access is not ready on this instance yet. Try again in a few minutes",
		)
	}
	if err != nil {
		return nil, err
	}

	email, reason, err := validateGrant(req, keys.StaffDomain)
	if err != nil {
		return nil, err
	}

	now := s.clock()
	unusable, err := (&tenant.User{}).GenerateUnusablePassword()
	if err != nil {
		return nil, err
	}
	principal := supportaccess.NewPrincipalUser(supportaccess.PrincipalUserParams{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		StaffEmail:     email,
		Password:       unusable,
		Assistant:      req.AllowsAssistant,
	})
	grant := &supportaccess.Grant{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		StaffEmail:      email,
		Reason:          reason,
		Scope:           req.Scope,
		AllowsAssistant: req.AllowsAssistant,
		DurationHours:   req.DurationHours,
		PrincipalUserID: principal.ID,
		GrantedByID:     req.TenantInfo.UserID,
		ExpiresAt:       now + int64(req.DurationHours)*int64(time.Hour/time.Second),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	created, err := s.grants.Create(ctx, &supportaccessports.CreateGrantRequest{
		Grant:     grant,
		Principal: principal,
		Now:       now,
	})
	if err != nil {
		return nil, err
	}

	s.record(ctx, &change{
		tenantInfo: req.TenantInfo,
		actor:      permissionports.UserActor(req.TenantInfo).AuditActor(),
		grant:      created,
		operation:  permission.OpCreate,
		action:     ActionGranted,
		after:      grantView(created, now),
		comment:    "Granted Trenova support access to " + created.StaffEmail + ": " + created.Reason,
	})
	s.notify(ctx, &supportaccessports.Notice{
		Kind:       supportaccessports.NoticeGranted,
		TenantInfo: req.TenantInfo,
		ActorID:    req.TenantInfo.UserID,
		Grant:      created,
	})

	return s.grants.Get(ctx, req.TenantInfo, created.ID)
}

type ExtendRequest struct {
	TenantInfo    pagination.TenantInfo
	GrantID       pulid.ID
	DurationHours int
}

func (s *Service) Extend(ctx context.Context, req *ExtendRequest) (*supportaccess.Grant, error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	if supportaccess.IsPrincipalID(req.TenantInfo.UserID) || req.TenantInfo.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Only a person in this organization may extend support access",
		)
	}

	before, err := s.grants.Get(ctx, req.TenantInfo, req.GrantID)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	next := *before
	if err = next.Extend(req.TenantInfo.UserID, req.DurationHours, now); err != nil {
		return nil, err
	}

	extended, err := s.grants.Extend(ctx, &next, now)
	if errors.Is(err, supportaccess.ErrGrantNotFound) {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalid,
			"This support access has already ended. Grant access again instead",
		)
	}
	if err != nil {
		return nil, err
	}
	s.principals.Forget(extended.PrincipalUserID)

	s.record(ctx, &change{
		tenantInfo: req.TenantInfo,
		actor:      permissionports.UserActor(req.TenantInfo).AuditActor(),
		grant:      extended,
		operation:  permission.OpCreate,
		action:     ActionExtended,
		before:     grantView(before, now),
		after:      grantView(extended, now),
		comment:    "Extended Trenova support access for " + extended.StaffEmail,
	})
	s.notify(ctx, &supportaccessports.Notice{
		Kind:       supportaccessports.NoticeExtended,
		TenantInfo: req.TenantInfo,
		ActorID:    req.TenantInfo.UserID,
		Grant:      extended,
	})

	return s.grants.Get(ctx, req.TenantInfo, extended.ID)
}

func validateGrant(req *GrantRequest, staffDomain string) (email, reason string, err error) {
	multiErr := errortypes.NewMultiError()
	email, emailErr := supportaccess.NormalizeStaffEmail(req.StaffEmail, staffDomain)
	switch {
	case errors.Is(emailErr, supportaccess.ErrEmailNotStaff):
		multiErr.Add(
			"staffEmail",
			errortypes.ErrInvalid,
			"Support access can be granted only to a Trenova staff address at {0}",
			staffDomain,
		)
	case emailErr != nil:
		multiErr.Add(
			"staffEmail",
			errortypes.ErrInvalid,
			"Enter a plain email address with no special characters",
		)
	}

	reason = strings.TrimSpace(req.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > supportaccess.MaxReasonLength {
		multiErr.Add(
			"reason",
			errortypes.ErrRequired,
			"Say why Trenova support needs access, in at most 500 characters",
		)
	}
	if !supportaccess.ValidDuration(req.DurationHours) {
		multiErr.Add("durationHours", errortypes.ErrInvalid, "Choose 1, 4, 24 or 72 hours")
	}
	if !req.Scope.IsValid() {
		multiErr.Add("scope", errortypes.ErrInvalid, "Choose read-only or full access")
	}

	if multiErr.HasErrors() {
		return "", "", multiErr
	}
	return email, reason, nil
}

type RevokeRequest struct {
	TenantInfo pagination.TenantInfo
	GrantID    pulid.ID
	Reason     string
}

// Revoke ends a live grant and every session under it at once. Two revocations at once cannot
// both succeed: the write carries the precondition.
func (s *Service) Revoke(ctx context.Context, req *RevokeRequest) (*supportaccess.Grant, error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	if supportaccess.IsPrincipalID(req.TenantInfo.UserID) || req.TenantInfo.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Only a person in this organization may revoke support access",
		)
	}

	before, err := s.grants.Get(ctx, req.TenantInfo, req.GrantID)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	next := *before
	if err = next.Revoke(req.TenantInfo.UserID, req.Reason, now); err != nil {
		return nil, err
	}

	revoked, err := s.grants.Revoke(ctx, &next, now)
	if errors.Is(err, supportaccess.ErrGrantNotFound) {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalid,
			"This support access has already ended",
		)
	}
	if err != nil {
		return nil, err
	}

	s.endSessions(ctx, revoked.PrincipalUserID)

	s.record(ctx, &change{
		tenantInfo: req.TenantInfo,
		actor:      permissionports.UserActor(req.TenantInfo).AuditActor(),
		grant:      revoked,
		operation:  permission.OpDelete,
		action:     ActionRevoked,
		before:     grantView(before, now),
		after:      grantView(revoked, now),
		comment:    "Revoked Trenova support access for " + revoked.StaffEmail + ": " + revoked.RevokeReason,
	})
	s.notify(ctx, &supportaccessports.Notice{
		Kind:       supportaccessports.NoticeRevoked,
		TenantInfo: req.TenantInfo,
		ActorID:    req.TenantInfo.UserID,
		Grant:      revoked,
	})

	return s.grants.Get(ctx, req.TenantInfo, revoked.ID)
}

func (s *Service) endSessions(ctx context.Context, principalUserID pulid.ID) {
	s.principals.Forget(principalUserID)
	if err := s.sessions.DeleteAllForUser(ctx, principalUserID); err != nil {
		s.l.Error(
			"could not delete the sessions of a revoked support grant; each ends at its next request",
			zap.String("principalUserID", principalUserID.String()),
			zap.Error(err),
		)
	}
}

const activeWindow = 5 * 60

func (s *Service) ActiveNow(ctx context.Context, grants []*supportaccess.Grant) map[pulid.ID]int64 {
	out := make(map[pulid.ID]int64, len(grants))
	if !s.enabled {
		return out
	}
	now := s.clock()
	for _, grant := range grants {
		if !grant.LiveAt(now) {
			continue
		}
		sessions, err := s.sessions.ListForUser(ctx, grant.PrincipalUserID)
		if err != nil {
			s.l.Warn("could not read whether support is in the instance now", zap.Error(err))
			continue
		}
		for _, sess := range sessions {
			if sess.Support == nil || sess.ExpiresAt <= now ||
				now-sess.LastAccessedAt > activeWindow {
				continue
			}
			out[grant.ID] = max(out[grant.ID], sess.LastAccessedAt)
		}
	}
	return out
}

func (s *Service) List(
	ctx context.Context,
	req *supportaccessports.ListGrantsRequest,
) (*pagination.CursorListResult[*supportaccess.Grant], error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	return s.grants.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*supportaccess.Grant, error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	return s.grants.Get(ctx, tenantInfo, id)
}

// SetKeys stores the key set the operator delivered. An unchanged set writes nothing.
func (s *Service) SetKeys(ctx context.Context, keys *supportaccess.KeySet) error {
	if !s.enabled {
		return errNotAvailable
	}
	if err := keys.Normalize(); err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("keys", errortypes.ErrInvalid, err.Error())
		return multiErr
	}

	held, err := s.keys.Get(ctx)
	if err != nil && !errors.Is(err, supportaccess.ErrNoKeySet) {
		return err
	}
	if held != nil && held.Same(keys) {
		return nil
	}

	keys.ReceivedAt = s.clock()
	if err = s.keys.Save(ctx, keys); err != nil {
		return err
	}
	s.keyCache.Invalidate(keySetKey)

	keyIDs := make([]string, 0, len(keys.Keys))
	for _, key := range keys.Keys {
		keyIDs = append(keyIDs, key.KeyID)
	}
	s.l.Info(
		"the control plane set the support assertion keys",
		zap.String("instanceID", keys.InstanceID),
		zap.String("staffDomain", keys.StaffDomain),
		zap.Strings("keyIDs", keyIDs),
	)
	return nil
}

func (s *Service) LiveGrants(ctx context.Context) ([]*supportaccessports.LiveGrant, error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	return s.grants.ListLive(ctx, &supportaccessports.ListLiveRequest{
		Now:   s.clock(),
		Limit: supportaccessports.MaxReportedGrants,
	})
}

// KeySet answers what the instance holds, for the operator to compare with what it sends.
func (s *Service) KeySet(ctx context.Context) (*supportaccess.KeySet, error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	keys, err := s.keys.Get(ctx)
	if errors.Is(err, supportaccess.ErrNoKeySet) {
		return nil, nil
	}
	return keys, err
}

func (s *Service) keySet(ctx context.Context) (*supportaccess.KeySet, error) {
	cached, err := s.keyCache.Get(ctx, keySetKey, otter.LoaderFunc[string, cachedKeySet](
		func(ctx context.Context, _ string) (cachedKeySet, error) {
			keys, loadErr := s.keys.Get(ctx)
			if errors.Is(loadErr, supportaccess.ErrNoKeySet) {
				return cachedKeySet{}, nil
			}
			if loadErr != nil {
				return cachedKeySet{}, loadErr
			}
			return cachedKeySet{keys: keys}, nil
		},
	))
	if err != nil {
		return nil, err
	}
	if cached.keys == nil {
		return nil, supportaccess.ErrNoKeySet
	}
	return cached.keys, nil
}
