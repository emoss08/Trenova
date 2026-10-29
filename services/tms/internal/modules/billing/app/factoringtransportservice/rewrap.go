package factoringtransportservice

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	return []kernelports.EnvelopeColumn{{
		Name:  "factoring transport SFTP secret",
		Value: factoringsubmissiondb.Transports.SftpSecret,
		Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
			return []kernelports.SecretScope{*transportSecretScope(
				row.Tenant,
				&factoringsubmission.Transport{ID: pulid.ID(row.ID)},
			)}
		},
	}}
}
