package accountingconnectionapps

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func validateSaveApp(req *accountingconnectionports.SaveAccountingAppRequest) error {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(req.ClientID) == "" {
		multiErr.Add("clientId", errortypes.ErrRequired, "Client ID is required")
	}
	profile := accountingprovider.MustProfile(req.IntegrationType)
	if !req.Environment.IsValid() || !profile.HasEnvironment(req.Environment) {
		multiErr.Add(
			"environment",
			errortypes.ErrInvalid,
			"{0} apps run in {1}",
			profile.Name,
			environmentList(profile.Environments),
		)
	}
	if req.ClearWebhookVerifierToken && strings.TrimSpace(req.WebhookVerifierToken) != "" {
		multiErr.Add(
			"webhookVerifierToken",
			errortypes.ErrInvalid,
			"Enter a verifier token or clear it, not both",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func environmentList(environments []accountingprovider.AppEnvironment) string {
	names := make([]string, 0, len(environments))
	for _, env := range environments {
		names = append(names, string(env))
	}
	return strings.Join(names, " or ")
}

type appChange struct {
	req      *accountingconnectionports.SaveAccountingAppRequest
	existing *accountingconnection.AccountingAppCredential
	conn     *accountingconnection.AccountingConnection
	provider string
}

func (c *appChange) guardActiveConnection() error {
	if c.conn == nil || !c.conn.IsActive() {
		return nil
	}
	if c.conn.AppSource != accountingconnection.AppSourceTenant {
		return errortypes.NewBusinessError(
			"{0} company {1} is connected through this server's app. Disconnect it before switching to your own app.",
			c.provider,
			c.conn.ExternalCompanyName,
		)
	}
	fingerprint := accountingconnection.AppFingerprint(c.req.Environment, c.req.ClientID)
	if c.conn.AppFingerprint != fingerprint {
		return errortypes.NewBusinessError(
			"Disconnect {0} company {1} before changing the client ID or environment. Its authorization belongs to the app it was connected through; you can still replace the client secret or the webhook verifier token.",
			c.provider,
			c.conn.ExternalCompanyName,
		)
	}
	return nil
}

func sameCredentialVersion(before, after *accountingconnection.AccountingAppCredential) bool {
	if before == nil || after == nil {
		return before == nil && after == nil
	}
	return before.ID == after.ID && before.Version == after.Version
}
