package captureservice

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/capturecol"
)

func EnvelopeObjects() []kernelports.EnvelopeObject {
	page := capturecol.CapturePageColumns
	pageObject := func(name string, value buncolgen.Column) kernelports.EnvelopeObject {
		return kernelports.EnvelopeObject{Column: kernelports.EnvelopeColumn{
			Name:           name,
			Table:          capturecol.CapturePageTable,
			ID:             page.ID,
			OrganizationID: page.OrganizationID,
			BusinessUnitID: page.BusinessUnitID,
			Value:          value,
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
