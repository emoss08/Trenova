package migrations

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/emoss08/keel/migrate"
)

const (
	SourceName   = "core"
	Dir          = "schema/migrations"
	ReconcileRLS = "SELECT trenova_rls.reconcile()"
)

//go:embed *.sql
var files embed.FS

func Source() migrate.Source {
	return migrate.Source{Name: SourceName, FS: files}
}

func Set(editions ...migrate.Source) migrate.Set {
	return migrate.Sources(append([]migrate.Source{Source()}, editions...)...)
}

func Options() migrate.Options {
	return migrate.Options{After: []string{ReconcileRLS}}
}

func Dirs() []migrate.Dir {
	return []migrate.Dir{{Name: SourceName, Path: Dir}}
}

func File(id string) (string, error) {
	body, err := fs.ReadFile(files, id+".sql")
	if err != nil {
		return "", fmt.Errorf("read migration %s: %w", id, err)
	}

	return string(body), nil
}
