package keelcatalog

import (
	"fmt"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/keeltables"
)

func ForModel(model string) (*schema.Table, bool) {
	return keeltables.Catalog().Model(model)
}

func MustNamed(name string) *schema.Table {
	table, ok := keeltables.Catalog().Table(name)
	if !ok {
		panic(fmt.Sprintf("keelcatalog: no generated table is named %q", name))
	}

	return table
}
