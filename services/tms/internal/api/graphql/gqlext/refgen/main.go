// Command refgen writes the client's reference fragments and reference view
// map from the permission registry.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
)

const (
	schemaDir = "internal/api/graphql/schema"
	repoRoot  = "../.."
)

func main() {
	schema, err := schemadiff.LoadDir(schemaDir)
	if err != nil {
		log.Fatal(err)
	}
	registry := fieldsensitivity.DefaultRegistry()

	fragments, err := gqlext.RenderReferenceFragments(schema, registry)
	if err != nil {
		log.Fatal(err)
	}
	views, err := gqlext.RenderReferenceViews(schema, registry)
	if err != nil {
		log.Fatal(err)
	}
	write(gqlext.ReferenceFragmentsPath, fragments)
	write(gqlext.ReferenceViewsPath, views)
}

func write(path, content string) {
	target := filepath.Join(repoRoot, path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		log.Fatal(err)
	}
}
