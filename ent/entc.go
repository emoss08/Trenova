//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"reflect"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

func main() {
	opts := []entc.Option{
		entc.FeatureNames("privacy", "schema/snapshot", "entql", "sql/modifier", "sql/execquery", "namedges"),
	}

	if err := entc.Generate("./ent/schema", &gen.Config{
		Hooks: []gen.Hook{
			EnsureStructTag("json"),
		},
	}, opts...); err != nil {
		log.Fatalf("running ent codegen: %v", err)
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
