package storedobject

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcontentcol"
)

func EnvelopeObjects() []kernelports.EnvelopeObject {
	doc := documentcol.DocumentColumns
	content := documentcontentcol.ContentColumns
	return []kernelports.EnvelopeObject{
		{
			Column: kernelports.EnvelopeColumn{
				Name:           "document files",
				Table:          documentcol.DocumentTable,
				ID:             doc.ID,
				OrganizationID: doc.OrganizationID,
				BusinessUnitID: doc.BusinessUnitID,
				Value:          doc.StoragePath,
				Extra:          []buncolgen.Column{doc.StorageVersionID},
				Scopes:         objectScopes,
			},
			Version: &doc.StorageVersionID,
		},
		{
			Column: kernelports.EnvelopeColumn{
				Name:           "document previews",
				Table:          documentcol.DocumentTable,
				ID:             doc.ID,
				OrganizationID: doc.OrganizationID,
				BusinessUnitID: doc.BusinessUnitID,
				Value:          doc.PreviewStoragePath,
				Scopes:         objectScopes,
			},
		},
		{
			Column: kernelports.EnvelopeColumn{
				Name:           "document conversions",
				Table:          documentcontentcol.ContentTable,
				ID:             content.ID,
				OrganizationID: content.OrganizationID,
				BusinessUnitID: content.BusinessUnitID,
				Value:          content.RawObjectKey,
				Scopes:         objectScopes,
			},
		},
	}
}

func objectScopes(row *kernelports.EnvelopeRow, key string) []kernelports.SecretScope {
	return []kernelports.SecretScope{AAD(row.Tenant, key).SecretScope()}
}
