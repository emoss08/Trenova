package factoringservice

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	cols := factoringcol.TransportColumns
	return []kernelports.EnvelopeColumn{{
		Name:           "factoring transport SFTP secret",
		Table:          factoringcol.TransportTable,
		ID:             cols.ID,
		OrganizationID: cols.OrganizationID,
		BusinessUnitID: cols.BusinessUnitID,
		Value:          cols.SftpSecret,
		Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
			return []kernelports.SecretScope{*transportSecretScope(
				row.Tenant,
				&factoring.Transport{ID: pulid.ID(row.ID)},
			)}
		},
	}}
}
