package authservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	authErrorUnknownAccount     = "unknown_account"
	authErrorRejectedLogin      = "invalid_credentials"
	authErrorAccountUnavailable = "account_unavailable"
	authErrorOrganizationAccess = "organization_access_denied"
	authErrorSSORequired        = "sso_required"
	authErrorSSOState           = "sso_state_invalid"
	authErrorSSOConfig          = "sso_configuration_unavailable"
	authErrorSSOExchange        = "sso_code_exchange_failed"
	authErrorSSOAssertion       = "sso_id_token_invalid"
	authErrorSSONonce           = "sso_nonce_mismatch"
	authErrorSSOTenant          = "sso_tenant_mismatch"
	authErrorSSOIdentity        = "sso_identity_unresolved"
	authErrorAuthentication     = "authentication_failed"
	authErrorAuthorization      = "access_denied"
	authErrorInternal           = "internal_error"
)

type authAttempt struct {
	provider       string
	userID         pulid.ID
	organizationID pulid.ID
	businessUnitID pulid.ID
	aal            int
	fal            int
	mfaState       string
	errorCode      string
}

func newAuthAttempt(provider string) *authAttempt {
	return &authAttempt{provider: provider, aal: 1, fal: 1}
}

func (a *authAttempt) forUser(usr *tenant.User) {
	if usr == nil {
		return
	}
	a.userID = usr.ID
	if a.organizationID.IsNil() {
		a.organizationID = usr.CurrentOrganizationID
		a.businessUnitID = usr.BusinessUnitID
	}
}

func (a *authAttempt) forOrganization(orgID, buID pulid.ID) {
	if orgID.IsNil() || buID.IsNil() {
		return
	}
	a.organizationID = orgID
	a.businessUnitID = buID
}

func (a *authAttempt) fail(code string) {
	a.errorCode = code
}

func (s *Service) recordAuthAttempt(ctx context.Context, attempt *authAttempt, err error) {
	if s.authEvents == nil {
		return
	}

	rec := services.AuthEventRecord{
		Provider:         attempt.provider,
		Outcome:          authOutcome(err),
		UserID:           attempt.userID,
		OrganizationID:   attempt.organizationID,
		BusinessUnitID:   attempt.businessUnitID,
		AuthenticatorAAL: attempt.aal,
		FederationFAL:    attempt.fal,
		MFAState:         attempt.mfaState,
		RiskOutcome:      iam.RiskOutcomeAllow,
	}
	if err != nil {
		rec.ErrorCode = attempt.errorCode
		if rec.ErrorCode == "" {
			rec.ErrorCode = authErrorCode(err)
		}
		if rec.Outcome == iam.AuthEventOutcomeDenied {
			rec.RiskOutcome = iam.RiskOutcomeDeny
		}
	}

	s.authEvents.Record(ctx, &rec)
}

func authOutcome(err error) iam.AuthEventOutcome {
	switch {
	case err == nil:
		return iam.AuthEventOutcomeSuccess
	case errors.Is(err, errSSORequired), errortypes.IsAuthorizationError(err):
		return iam.AuthEventOutcomeDenied
	default:
		return iam.AuthEventOutcomeFailed
	}
}

func authErrorCode(err error) string {
	switch {
	case errors.Is(err, errSSORequired):
		return authErrorSSORequired
	case errortypes.IsAuthorizationError(err):
		return authErrorAuthorization
	case errortypes.IsAuthenticationError(err):
		return authErrorAuthentication
	default:
		return authErrorInternal
	}
}
