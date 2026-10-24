package storedobject

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent/documentcontentdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func EnvelopeObjects() []kernelports.EnvelopeObject {
	doc := documentdb.Documents
	return []kernelports.EnvelopeObject{
		{
			Column: kernelports.EnvelopeColumn{
				Name:   "document files",
				Value:  doc.StoragePath,
				Extra:  []keel.Expr{doc.StorageVersionID},
				Scopes: objectScopes,
			},
			Version: doc.StorageVersionID,
		},
		{
			Column: kernelports.EnvelopeColumn{
				Name:   "document previews",
				Value:  doc.PreviewStoragePath,
				Scopes: objectScopes,
			},
		},
		{
			Column: kernelports.EnvelopeColumn{
				Name:   "document conversions",
				Value:  documentcontentdb.Contents.RawObjectKey,
				Scopes: objectScopes,
			},
		},
	}
}

func objectScopes(row *kernelports.EnvelopeRow, key string) []kernelports.SecretScope {
	return []kernelports.SecretScope{AAD(row.Tenant, key).SecretScope()}
}
