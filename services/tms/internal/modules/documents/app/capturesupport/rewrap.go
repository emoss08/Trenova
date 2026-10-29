package capturesupport

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func EnvelopeObjects() []kernelports.EnvelopeObject {
	page := capturedb.CapturePages
	pageObject := func(name string, value keel.Expr) kernelports.EnvelopeObject {
		return kernelports.EnvelopeObject{Column: kernelports.EnvelopeColumn{
			Name:  name,
			Value: value,
			Scopes: func(row *kernelports.EnvelopeRow, key string) []kernelports.SecretScope {
				return []kernelports.SecretScope{captureObjectAAD(row.Tenant, key).SecretScope()}
			},
		}}
	}
	return []kernelports.EnvelopeObject{
		pageObject("capture pages", page.StoragePath),
		pageObject("capture page thumbnails", page.ThumbnailPath),
	}
}
