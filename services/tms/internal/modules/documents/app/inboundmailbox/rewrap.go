package inboundmailbox

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	m := inboundmessagedb.Mailboxes
	mailboxColumn := func(name string, value keel.Expr, field string) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:   name,
			Value:  value,
			Rebind: true,
			Scopes: mailboxSecretScopes(field),
		}
	}
	return []kernelports.EnvelopeColumn{
		mailboxColumn(
			"inbound mailbox signing secret",
			m.SigningSecret,
			inboundmessage.SigningSecretField,
		),
		mailboxColumn(
			"inbound mailbox provider API key",
			m.ProviderAPIKey,
			inboundmessage.ProviderAPIKeyField,
		),
	}
}

func mailboxSecretScopes(field string) kernelports.EnvelopeScopes {
	return func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
		bound := encryptionservice.InboundMailboxSecretAAD(
			row.Tenant.OrgID,
			row.Tenant.BuID,
			pulid.ID(row.ID),
			field,
		)
		return []kernelports.SecretScope{bound.SecretScope(), {}}
	}
}
