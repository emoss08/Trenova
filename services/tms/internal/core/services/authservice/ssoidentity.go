package authservice

import (
	"bytes"
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type oidcBool struct {
	set   bool
	value bool
}

func (b *oidcBool) UnmarshalJSON(data []byte) error {
	trimmed := bytes.Trim(bytes.TrimSpace(data), `"`)
	switch strings.ToLower(string(trimmed)) {
	case "true":
		b.set, b.value = true, true
	case "false":
		b.set, b.value = true, false
	default:
		b.set, b.value = false, false
	}

	return nil
}

func (b oidcBool) IsTrue() bool {
	return b.set && b.value
}

type ssoIdentity struct {
	Issuer       string
	Subject      string
	Email        string
	EmailTrusted bool
}

type ssoUserLookup struct {
	Config      *tenant.SSOConfig
	Identity    ssoIdentity
	DisplayName string
}

func identityFromOIDCClaims(
	issuer string,
	claims *oidcClaims,
	provider tenant.SSOProvider,
) ssoIdentity {
	return ssoIdentity{
		Issuer:  strings.TrimSpace(issuer),
		Subject: strings.TrimSpace(claims.Subject),
		Email:   claims.EmailAddress(provider),
		EmailTrusted: provider == tenant.SSOProviderAzureAD ||
			claims.EmailVerified.IsTrue(),
	}
}

func (s *Service) resolveSSOUser(ctx context.Context, req *ssoUserLookup) (*tenant.User, error) {
	if s.links == nil {
		return nil, errortypes.NewBusinessError("SSO is not configured")
	}

	identity := req.Identity
	if identity.Issuer == "" || identity.Subject == "" {
		return nil, errortypes.NewAuthenticationError(
			"{0} identity token did not identify the account", req.DisplayName,
		)
	}

	if identity.Email != "" {
		if err := validateAllowedDomain(identity.Email, req.Config.AllowedDomains); err != nil {
			return nil, err
		}
	}

	link, err := s.links.GetBySubject(ctx, repositories.GetSSOIdentityLinkBySubjectRequest{
		SSOConfigID: req.Config.ID,
		Issuer:      identity.Issuer,
		Subject:     identity.Subject,
	})
	switch {
	case err == nil:
		return s.userForLink(ctx, link, req.DisplayName)
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}

	return s.linkSSOIdentity(ctx, req)
}

func (s *Service) userForLink(
	ctx context.Context,
	link *tenant.SSOIdentityLink,
	displayName string,
) (*tenant.User, error) {
	usr, err := s.ur.FindByIDForLogin(ctx, link.UserID)
	if err != nil {
		return nil, errortypes.NewAuthenticationError(
			"No Trenova user exists for this {0} account", displayName,
		)
	}

	if err = s.links.RecordLogin(ctx, link, timeutils.NowUnix()); err != nil {
		s.l.Warn("failed to record sso identity login", zap.Error(err))
	}

	return usr, nil
}

func (s *Service) linkSSOIdentity(ctx context.Context, req *ssoUserLookup) (*tenant.User, error) {
	identity := req.Identity
	if identity.Email == "" {
		return nil, errortypes.NewAuthenticationError(
			"{0} account did not provide a usable email address", req.DisplayName,
		)
	}

	if !identity.EmailTrusted {
		return nil, errortypes.NewAuthenticationError(
			"{0} has not verified this account's email address, so it cannot be matched to a Trenova user",
			req.DisplayName,
		)
	}

	usr, err := s.ur.FindByEmail(ctx, identity.Email)
	if err != nil {
		return nil, errortypes.NewAuthenticationError(
			"No Trenova user exists for this {0} account", req.DisplayName,
		)
	}

	existing, err := s.links.GetByUser(ctx, repositories.GetSSOIdentityLinkByUserRequest{
		SSOConfigID: req.Config.ID,
		Issuer:      identity.Issuer,
		UserID:      usr.ID,
	})
	switch {
	case err == nil && existing.Subject != identity.Subject:
		s.l.Warn(
			"refused an sso sign-in claiming a user already linked to another identity",
			zap.String("ssoConfigID", req.Config.ID.String()),
			zap.String("userID", usr.ID.String()),
		)
		return nil, errortypes.NewAuthenticationError(
			"This Trenova user is already linked to a different {0} account", req.DisplayName,
		)
	case err == nil:
		return s.userForLink(ctx, existing, req.DisplayName)
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}

	now := timeutils.NowUnix()
	link := &tenant.SSOIdentityLink{
		OrganizationID: req.Config.OrganizationID,
		BusinessUnitID: req.Config.BusinessUnitID,
		SSOConfigID:    req.Config.ID,
		UserID:         usr.ID,
		Issuer:         identity.Issuer,
		Subject:        identity.Subject,
		EmailAtLink:    identity.Email,
		LastLoginAt:    now,
	}
	if err = s.links.Create(ctx, link); err != nil {
		if !dberror.IsUniqueConstraintViolation(err) {
			return nil, err
		}

		return s.resolveConcurrentLink(ctx, req, usr)
	}

	return usr, nil
}

func (s *Service) resolveConcurrentLink(
	ctx context.Context,
	req *ssoUserLookup,
	usr *tenant.User,
) (*tenant.User, error) {
	link, err := s.links.GetBySubject(ctx, repositories.GetSSOIdentityLinkBySubjectRequest{
		SSOConfigID: req.Config.ID,
		Issuer:      req.Identity.Issuer,
		Subject:     req.Identity.Subject,
	})
	if err != nil || link.UserID != usr.ID {
		return nil, errortypes.NewAuthenticationError(
			"This Trenova user is already linked to a different {0} account", req.DisplayName,
		)
	}

	return usr, nil
}
