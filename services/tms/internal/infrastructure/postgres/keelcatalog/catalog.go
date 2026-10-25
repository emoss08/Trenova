package keelcatalog

import (
	"fmt"
	"sync"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/keeltables"
)

type catalog struct {
	byModel map[string]*schema.Table
	byName  map[string]*schema.Table
}

var tables = sync.OnceValue(func() catalog {
	all := keeltables.All()
	c := catalog{
		byModel: make(map[string]*schema.Table, len(all)),
		byName:  make(map[string]*schema.Table, len(all)),
	}
	ambiguous := make(map[string]bool)
	for _, table := range all {
		if _, seen := c.byModel[table.Model]; seen {
			ambiguous[table.Model] = true
		}
		c.byModel[table.Model] = table
		c.byName[table.Name] = table
	}
	for model := range ambiguous {
		delete(c.byModel, model)
	}

	return c
})

func ForModel(model string) (*schema.Table, bool) {
	table, ok := tables().byModel[model]
	return table, ok
}

func MustNamed(name string) *schema.Table {
	table, ok := tables().byName[name]
	if !ok {
		panic(fmt.Sprintf("keelcatalog: no generated table is named %q", name))
	}

	return table
}
