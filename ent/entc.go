//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"reflect"

	"ariga.io/atlas/sql/migrate"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/emoss08/trenova/ent/migrate/migratedata"
)

func main() {
	err := entc.Generate("./schema", &gen.Config{
		Hooks: []gen.Hook{
			EnsureStructTag("json"),
		},
	})
	if err != nil {
		log.Fatalf("running ent codegen: %v", err)
	}

	// Migration Directory
	migrationDir, migrationErr := migrate.NewLocalDir("./migrate/migrations")

	if migrationErr != nil {
		log.Fatalf("creating migration directory: %v", migrationErr)
	}

	// Call the seeders
	if err := migratedata.SeedBusinessUnit(migrationDir); err != nil {
		log.Fatalf("running seed business unit: %v", err)
	}

	if err := migratedata.SeedOrganization(migrationDir); err != nil {
		log.Fatalf("running seed organization: %v", err)
	}
}

// EnsureStructTag ensures all fields in the graph have a specific tag name.
func EnsureStructTag(name string) gen.Hook {
	return func(next gen.Generator) gen.Generator {
		return gen.GenerateFunc(func(g *gen.Graph) error {
			for _, node := range g.Nodes {
				for _, field := range node.Fields {
					tag := reflect.StructTag(field.StructTag)
					if _, ok := tag.Lookup(name); !ok {
						return fmt.Errorf("struct tag %q is missing for field %s.%s", name, node.Name, field.Name)
					}
				}
			}
			return next.Generate(g)
		})
	}
}
