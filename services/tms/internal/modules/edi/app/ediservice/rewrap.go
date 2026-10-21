package ediservice

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/edicol"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	cols := edicol.EDICommunicationProfileColumns
	return []kernelports.EnvelopeColumn{{
		Name:           "EDI communication profile secrets",
		Table:          edicol.EDICommunicationProfileTable,
		ID:             cols.ID,
		OrganizationID: cols.OrganizationID,
		BusinessUnitID: cols.BusinessUnitID,
		Value:          cols.EncryptedSecrets,
		JSONMap:        true,
		Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
			return []kernelports.SecretScope{
				profileSecretAAD(row.Tenant, pulid.ID(row.ID), field).SecretScope(),
			}
		},
	}}
}
