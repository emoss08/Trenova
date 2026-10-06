package migrations

import (
	"embed"
	"io/fs"
)

//go:embed *.sql
var sqlMigrations embed.FS

func FS() fs.FS {
	return sqlMigrations
}
