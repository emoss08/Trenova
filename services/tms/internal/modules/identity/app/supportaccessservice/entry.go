package supportaccessservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// supportAuthenticatorAAL is the assurance a support session carries: the control plane mints
	// an assertion only for a staff member signed in with Google Workspace who confirmed a
	// passkey within its recent-authentication window (managed-mode.md, "Support access").
	supportAuthenticatorAAL = 2
	supportLocale           = "en"
	noticeTimeout           = 10 * time.Second
)

type EnterRequest struct {
	Assertion string
}

type EnterResult struct {
	Token     string
	SessionID pulid.ID
	ExpiresAt int64
	Grant     *supportaccess.Grant
}

// Enter turns a support assertion into a support session. Every refusal is a
// *supportaccess.RefusalError whose reason is for the logs alone; the caller answers every one
// the same way. A refusal of an assertion that verified is also written to the named
// organization's security audit log, so the customer sees each attempt that reached it.
func (s *Service) Enter(ctx context.Context, req *EnterRequest) (*EnterResult, error) {
	if !s.enabled {
		return nil, &supportaccess.RefusalError{Reason: supportaccess.RefusalNotConfigured}
	}

	keys, err := s.keySet(ctx)
	if errors.Is(err, supportaccess.ErrNoKeySet) {
		return nil, &supportaccess.RefusalError{Reason: supportaccess.RefusalNotConfigured}
	}
	if err != nil {
		return nil, err
	}

	now := s.clock()
	assertion, err := supportaccess.VerifyAssertion(req.Assertion, keys, time.Unix(now, 0))
	if err != nil {
		s.logRefusal(err, nil)
		return nil, err
	}

	first, err := s.ledger.Use(ctx, &supportaccess.AssertionUse{
		AssertionID: assertion.ID,
		UsedAt:      now,
		ExpiresAt:   assertion.UseExpiresAt(),
	})
	if err != nil {
		return nil, err
	}
	if !first {
		return nil, s.refuseVerified(ctx, assertion, supportaccess.RefusalReplayed)
	}

	grant, err := s.grants.FindLiveForEntry(ctx, &supportaccessports.FindLiveGrantRequest{
		OrganizationID: assertion.OrganizationID,
		StaffEmail:     assertion.StaffEmail,
		Now:            now,
	})
	if errors.Is(err, supportaccess.ErrGrantNotFound) {
		return nil, s.refuseVerified(ctx, assertion, supportaccess.RefusalNoGrant)
	}
	if err != nil {
		return nil, err
	}
	if grant.OrganizationID != assertion.OrganizationID || grant.StaffEmail != assertion.StaffEmail ||
		!grant.LiveAt(now) {
		return nil, s.refuseVerified(ctx, assertion, supportaccess.RefusalNoGrant)
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  grant.OrganizationID,
		BuID:   grant.BusinessUnitID,
		UserID: grant.PrincipalUserID,
	}
	support := &authctx.Support{
		GrantID:    grant.ID,
		StaffEmail: grant.StaffEmail,
		StaffName:  assertion.StaffName,
		Scope:      grant.Scope.String(),
		EndsAt:     grant.ExpiresAt,
	}
	scoped := authctx.WithSupport(
		dbscope.WithTenant(ctx, tenantInfo.DBTenant()),
		support,
		grant.PrincipalUserID,
	)

	sess, token, err := s.openSession(scoped, tenantInfo, support, now)
	if err != nil {
		return nil, err
	}

	if err = s.grants.MarkEntered(scoped, &supportaccessports.MarkEnteredRequest{
		Grant:     grant,
		StaffName: assertion.StaffName,
		At:        now,
	}); err != nil {
		s.l.Warn("could not record when support last entered", zap.Error(err))
	}

	s.recordEntry(scoped, grant, assertion, sess, tenantInfo, now)
	s.notifyStarted(scoped, grant, assertion, tenantInfo)

	return &EnterResult{
		Token:     token,
		SessionID: sess.ID,
		ExpiresAt: sess.ExpiresAt,
		Grant:     grant,
	}, nil
}

func (s *Service) openSession(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	support *authctx.Support,
	now int64,
) (*session.Session, string, error) {
	expiresAt := support.EndsAt
	if s.maxAge > 0 {
		expiresAt = min(expiresAt, now+int64(s.maxAge/time.Second))
	}

	meta, _ := requestmeta.From(ctx)
	sess := session.NewSession(&session.NewSessionRequest{
		TenantInfo:            tenantInfo,
		ExpiresAt:             expiresAt,
		AuthProvider:          iamports.AuthEventProviderSupport,
		AuthenticatorAAL:      supportAuthenticatorAAL,
		FederationFAL:         1,
		MFAAuthenticatedAt:    now,
		LastReauthenticatedAt: now,
		Locale:                supportLocale,
		IPAddress:             meta.ClientIP,
		UserAgent:             meta.UserAgent,
	})
	sess.Support = support

	issued, err := s.tokens.Issue(sess.ID, sess.UserID)
	if err != nil {
		return nil, "", err
	}
	sess.SecretHash = issued.SecretHash
	if err = s.sessions.Create(ctx, sess); err != nil {
		return nil, "", err
	}
	return sess, issued.Token, nil
}

func (s *Service) recordEntry(
	ctx context.Context,
	grant *supportaccess.Grant,
	assertion *supportaccess.Assertion,
	sess *session.Session,
	tenantInfo pagination.TenantInfo,
	now int64,
) {
	s.record(ctx, &change{
		tenantInfo: tenantInfo,
		actor:      principalActor(grant),
		grant:      grant,
		operation:  permission.OpRead,
		action:     ActionSessionStarted,
		after: map[string]any{
			"staffEmail": grant.StaffEmail,
			"staffName":  assertion.StaffName,
			"scope":      grant.Scope.String(),
			"sessionEnd": sess.ExpiresAt,
		},
		comment: "Trenova support session started by " + assertion.StaffName +
			" (" + grant.StaffEmail + ")",
		metadata: map[string]any{
			"assertionId": assertion.ID,
			"staffName":   assertion.StaffName,
			"startedAt":   now,
		},
	})
	s.authEvents.Record(ctx, &iamports.AuthEventRecord{
		Provider:         iamports.AuthEventProviderSupport,
		Outcome:          iam.AuthEventOutcomeSuccess,
		UserID:           grant.PrincipalUserID,
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		AuthenticatorAAL: supportAuthenticatorAAL,
		FederationFAL:    1,
	})
}

func principalActor(grant *supportaccess.Grant) kernelports.AuditActor {
	return kernelports.AuditActor{
		PrincipalType: permissionports.PrincipalTypeUser,
		PrincipalID:   grant.PrincipalUserID,
		UserID:        grant.PrincipalUserID,
	}
}

func (s *Service) notifyStarted(
	ctx context.Context,
	grant *supportaccess.Grant,
	assertion *supportaccess.Assertion,
	tenantInfo pagination.TenantInfo,
) {
	noticeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), noticeTimeout)
	defer cancel()
	if err := s.notifier.SupportSessionStarted(noticeCtx, &supportaccessports.SessionStartedNotice{
		TenantInfo: tenantInfo,
		GrantID:    grant.ID,
		StaffEmail: grant.StaffEmail,
		StaffName:  assertion.StaffName,
		Scope:      grant.Scope,
		EndsAt:     grant.ExpiresAt,
	}); err != nil {
		s.l.Error(
			"could not tell the organization's administrators a support session started",
			zap.String("grantID", grant.ID.String()),
			zap.Error(err),
		)
	}
}

// refuseVerified records a refused entry whose assertion verified in the organization it names,
// when that organization exists, and returns the refusal.
func (s *Service) refuseVerified(
	ctx context.Context,
	assertion *supportaccess.Assertion,
	reason supportaccess.RefusalReason,
) error {
	refusal := &supportaccess.RefusalError{Reason: reason}
	s.logRefusal(refusal, assertion)

	buID, err := s.grants.OrganizationBusinessUnit(ctx, assertion.OrganizationID)
	if err != nil {
		if !errors.Is(err, supportaccess.ErrGrantNotFound) {
			s.l.Error("could not find the organization of a refused support entry", zap.Error(err))
		}
		return refusal
	}

	tenantInfo := pagination.TenantInfo{OrgID: assertion.OrganizationID, BuID: buID}
	scoped := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	s.auditor.RecordChange(scoped, &permissionports.SecurityChange{
		Resource:       permission.ResourceSupportAccess,
		ResourceID:     assertion.ID,
		Operation:      permission.OpRead,
		Action:         ActionEntryRefused,
		Actor:          permissionports.SystemAuditActor(),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		After: map[string]any{
			"staffEmail": assertion.StaffEmail,
			"staffName":  assertion.StaffName,
			"reason":     string(reason),
		},
		Comment: "Refused a Trenova support entry for " + assertion.StaffEmail,
		Metadata: map[string]any{
			"supportAccess": ActionEntryRefused,
			"assertionId":   assertion.ID,
			"refusal":       string(reason),
		},
	})
	s.authEvents.Record(scoped, &iamports.AuthEventRecord{
		Provider:       iamports.AuthEventProviderSupport,
		Outcome:        iam.AuthEventOutcomeDenied,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ErrorCode:      string(reason),
	})
	return refusal
}

func (s *Service) logRefusal(err error, assertion *supportaccess.Assertion) {
	fields := []zap.Field{zap.String("reason", string(supportaccess.ReasonOf(err))), zap.Error(err)}
	if assertion != nil {
		fields = append(fields,
			zap.String("assertionID", assertion.ID),
			zap.String("staffEmail", assertion.StaffEmail),
			zap.String("organizationID", assertion.OrganizationID.String()),
		)
	}
	s.l.Warn("refused a Trenova support entry", fields...)
}
