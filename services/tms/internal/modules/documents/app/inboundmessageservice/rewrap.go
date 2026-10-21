package inboundmessageservice

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	cols := inboundmessagecol.MailboxColumns
	mailboxColumn := func(name string, value buncolgen.Column) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:           name,
			Table:          inboundmessagecol.MailboxTable,
			ID:             cols.ID,
			OrganizationID: cols.OrganizationID,
			BusinessUnitID: cols.BusinessUnitID,
			Value:          value,
			Scopes:         unboundMailboxScopes,
		}
	}
	return []kernelports.EnvelopeColumn{
		mailboxColumn("inbound mailbox signing secret", cols.SigningSecret),
		mailboxColumn("inbound mailbox provider API key", cols.ProviderAPIKey),
	}
}

func unboundMailboxScopes(*kernelports.EnvelopeRow, string) []kernelports.SecretScope {
	return []kernelports.SecretScope{{}}
}
