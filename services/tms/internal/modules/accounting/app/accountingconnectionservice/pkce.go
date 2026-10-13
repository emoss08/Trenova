package accountingconnectionservice

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/tokenutils"
)

var errVerifierMissing = errors.New("the authorization state holds no code verifier")

func (s *Service) verifierAAD(
	tenantInfo pagination.TenantInfo,
	stateHash string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingOAuthVerifier,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     stateHash,
	}
}

func (s *Service) newCodeVerifier(
	tenantInfo pagination.TenantInfo,
	stateHash string,
	typ integration.Type,
) (challenge, sealed string, err error) {
	if !accountingsync.MustProfile(typ).PKCE {
		return "", "", nil
	}
	verifier, err := tokenutils.PKCEVerifier()
	if err != nil {
		return "", "", err
	}
	sealed, err = s.encryption.EncryptStringWithAAD(verifier, s.verifierAAD(tenantInfo, stateHash))
	if err != nil {
		return "", "", errortypes.NewBusinessError("Could not start the connection. Try again.").
			WithInternal(err)
	}
	return tokenutils.PKCEChallenge(verifier), sealed, nil
}

func (s *Service) openCodeVerifier(
	tenantInfo pagination.TenantInfo,
	stateHash string,
	state *accountingsyncports.AccountingOAuthState,
) (string, error) {
	if !accountingsync.MustProfile(state.IntegrationType).PKCE {
		return "", nil
	}
	if state.SealedVerifier == "" {
		return "", errVerifierMissing
	}
	return s.encryption.DecryptStringWithAAD(
		state.SealedVerifier,
		s.verifierAAD(tenantInfo, stateHash),
	)
}
