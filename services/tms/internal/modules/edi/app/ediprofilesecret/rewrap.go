package ediprofilesecret

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	return []kernelports.EnvelopeColumn{{
		Name:    "EDI communication profile secrets",
		Value:   edidb.EDICommunicationProfiles.EncryptedSecrets,
		JSONMap: true,
		Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
			return []kernelports.SecretScope{
				ProfileSecretAAD(row.Tenant, pulid.ID(row.ID), field).SecretScope(),
			}
		},
	}}
}
