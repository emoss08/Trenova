package accountingconnectionauth

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const CompanyChoiceTTL = 10 * time.Minute

type StateOwner struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
}

type pendingGrant struct {
	AccessToken     string        `json:"accessToken"`
	RefreshToken    string        `json:"refreshToken"`
	AccessTokenTTL  time.Duration `json:"accessTokenTtl"`
	RefreshTokenTTL time.Duration `json:"refreshTokenTtl"`
	CertificateID   string        `json:"certificateId,omitempty"`
}

type AuthorizedApp struct {
	Connector accountingconnectorports.AccountingConnector
	Identity  accountingconnection.AppIdentity
	Provider  string
}

type CompanyConnect struct {
	Owner       *StateOwner
	App         *AuthorizedApp
	Grant       *accountingconnectorports.AccountingTokenGrant
	Company     accountingconnectorports.AccountingCompany
	Certificate *accountingconnectionkit.HeldCertificate
}

func validateCompletion(
	req *accountingconnectionports.CompleteAccountingAuthorizationRequest,
	profile *accountingprovider.ProviderProfile,
) error {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(req.State) == "" {
		multiErr.Add("state", errortypes.ErrRequired, "The connection request is missing its state")
	}
	if strings.TrimSpace(req.Code) == "" {
		multiErr.Add("code", errortypes.ErrRequired, "The authorization code is missing")
	}
	if (profile.CallbackCarriesCompany || req.RealmID != "") &&
		!accountingconnectionkit.RealmIDPattern.MatchString(req.RealmID) {
		multiErr.Add(
			"realmId",
			errortypes.ErrInvalid,
			"The company id returned by the provider is not valid",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type CompanyChoiceOffer struct {
	Owner         *StateOwner
	State         *accountingconnectionports.AccountingOAuthState
	App           *AuthorizedApp
	Grant         *accountingconnectorports.AccountingTokenGrant
	Companies     []accountingconnectorports.AccountingCompany
	GrantRealm    string
	CertificateID string
}

func grantRealmOf(
	callbackRealmID string,
	companies []accountingconnectorports.AccountingCompany,
) string {
	if callbackRealmID != "" || len(companies) == 0 {
		return callbackRealmID
	}
	return companies[0].ID
}

func validateChoice(req *accountingconnectionports.ChooseAccountingCompanyRequest) error {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(req.ChoiceToken) == "" {
		multiErr.Add(
			"choiceToken",
			errortypes.ErrRequired,
			"The company choice is missing its token",
		)
	}
	if !accountingconnectionkit.RealmIDPattern.MatchString(req.CompanyID) {
		multiErr.Add("companyId", errortypes.ErrInvalid, "Choose one of the offered companies")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func splitChoice(
	choices []accountingconnectionports.AccountingOAuthCompany,
	companyID string,
) (chosen *accountingconnectorports.AccountingCompany, others []accountingconnectorports.AccountingCompany) {
	others = make([]accountingconnectorports.AccountingCompany, 0, len(choices))
	for _, choice := range choices {
		company := accountingconnectorports.AccountingCompany(choice)
		if chosen == nil && choice.ID == companyID {
			chosen = &company
			continue
		}
		others = append(others, company)
	}
	return chosen, others
}

func errStateExpired() error {
	return errortypes.NewValidationError(
		"state",
		errortypes.ErrInvalid,
		"This connection request expired or was already used. Start the connection again.",
	)
}

func (g *pendingGrant) tokenGrant() *accountingconnectorports.AccountingTokenGrant {
	return &accountingconnectorports.AccountingTokenGrant{
		AccessToken:     g.AccessToken,
		RefreshToken:    g.RefreshToken,
		AccessTokenTTL:  g.AccessTokenTTL,
		RefreshTokenTTL: g.RefreshTokenTTL,
	}
}

type connectionSave struct {
	owner       *StateOwner
	realmID     string
	grant       *accountingconnectorports.AccountingTokenGrant
	facts       *accountingconnection.CompanyFacts
	app         accountingconnection.AppIdentity
	provider    string
	now         int64
	certificate *accountingconnectionkit.HeldCertificate
}

func errRealmTaken(provider string) error {
	return errortypes.NewBusinessError(
		"This {0} company is already connected to another Trenova organization. Disconnect it there first.",
		provider,
	)
}
