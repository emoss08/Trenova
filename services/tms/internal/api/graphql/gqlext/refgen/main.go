// Command refgen writes the client's reference fragments and reference view
// map from the permission registry, then the page access manifest from the
// web app's screens and the operations they run.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/api/pagemanifest"
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
	write(gqlext.ReferenceFragmentsPath, []byte(fragments))
	write(gqlext.ReferenceViewsPath, []byte(views))

	inputs, err := pagemanifest.FromRepository(repoRoot, registry)
	if err != nil {
		log.Fatal(err)
	}
	manifest, err := pagemanifest.Build(inputs)
	if err != nil {
		log.Fatal(err)
	}
	rendered, err := pagemanifest.Render(manifest)
	if err != nil {
		log.Fatal(err)
	}
	write(pagemanifest.ManifestPath, rendered)

	words, err := pagemanifest.RenderWords(schema, registry)
	if err != nil {
		log.Fatal(err)
	}
	write(pagemanifest.WordsPath, []byte(words))
}

func write(path string, content []byte) {
	target := filepath.Join(repoRoot, path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o600); err != nil {
		log.Fatal(err)
	}
}
