package inboundmessageservice

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	cols := inboundmessagecol.MailboxColumns
	mailboxColumn := func(
		name string,
		value buncolgen.Column,
		field string,
	) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:           name,
			Table:          inboundmessagecol.MailboxTable,
			ID:             cols.ID,
			OrganizationID: cols.OrganizationID,
			BusinessUnitID: cols.BusinessUnitID,
			Value:          value,
			Rebind:         true,
			Scopes:         mailboxSecretScopes(field),
		}
	}
	return []kernelports.EnvelopeColumn{
		mailboxColumn(
			"inbound mailbox signing secret",
			cols.SigningSecret,
			inboundmessage.SigningSecretField,
		),
		mailboxColumn(
			"inbound mailbox provider API key",
			cols.ProviderAPIKey,
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
