package migrations

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/emoss08/keel/migrate"
)

const (
	SourceName            = "core"
	Dir                   = "schema/migrations"
	ReconcileRLS          = "SELECT trenova_rls.reconcile()"
	MarkLeakproof         = "SELECT trenova_rls.mark_leakproof_operators()"
	EnsureAuditPartitions = "SELECT audit_partitions.ensure_partitions(NULL, 3)"
)

//go:embed *.sql
var files embed.FS

func Source() migrate.Source {
	return migrate.Source{Name: SourceName, FS: files}
}

func Set() migrate.Set {
	return migrate.Sources(Source())
}

func Options() migrate.Options {
	return migrate.Options{
		After:  []string{ReconcileRLS},
		Ensure: []string{MarkLeakproof, EnsureAuditPartitions},
	}
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
