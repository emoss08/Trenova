//go:build integration

package accountingtest

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const Realm = "9341452431742015"

func NewConnection(
	tenant pagination.TenantInfo,
	userID pulid.ID,
	realmID string,
	now int64,
) *accountingconnection.AccountingConnection {
	conn := &accountingconnection.AccountingConnection{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		IntegrationType: integration.TypeQuickBooksOnline,
		ExternalRealmID: realmID,
	}
	conn.Connect(userID, accountingconnection.TokenGrant{
		AccessTokenCiphertext:  "cipher-access",
		AccessTokenExpiresAt:   now + 3600,
		RefreshTokenCiphertext: "cipher-refresh",
		RefreshTokenExpiresAt:  now + 8_640_000,
	}, now)
	conn.ApplyCompanyFacts(
		&accountingconnection.CompanyFacts{CompanyName: "Acme", HomeCurrency: "USD"},
	)
	return conn
}
