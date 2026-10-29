package accountingconnectionwebhook

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const appCredentialPrefix = "acctapp_"

type webhookSender struct {
	identity accountingconnection.AppIdentity
	tenant   *pagination.TenantInfo
}

func (w *webhookSender) owns(conn *accountingconnection.AccountingConnection) bool {
	if w.tenant != nil &&
		(conn.OrganizationID != w.tenant.OrgID || conn.BusinessUnitID != w.tenant.BuID) {
		return false
	}
	return conn.ConnectedThrough(w.identity)
}

func errWebhookUnverified() error {
	return errortypes.NewAuthenticationError("The webhook signature did not verify")
}

func webhookRealms(
	provider accountingconnectorports.AccountingProvider,
	body []byte,
) ([]string, error) {
	realms, err := provider.WebhookRealmIDs(body)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"The webhook body could not be read",
		)
	}
	return realms, nil
}
