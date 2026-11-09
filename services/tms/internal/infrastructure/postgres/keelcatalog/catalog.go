package keelcatalog

import (
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/keeltables"
)

func ForModel(model string) (*schema.Table, bool) {
	return keeltables.Catalog().Model(model)
}

func Named(name string) (*schema.Table, bool) {
	return keeltables.Catalog().Table(name)
}
