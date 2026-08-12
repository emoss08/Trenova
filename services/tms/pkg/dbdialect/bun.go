package dbdialect

import (
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
)

func FromBun(db bun.IDB) Kind {
	if db == nil {
		return DefaultKind
	}

	if db.Dialect().Name() == dialect.SQLite {
		return SQLite
	}

	return Postgres
}

func RequireFromBun(db bun.IDB, capability Capability) error {
	return Require(FromBun(db), capability)
}
