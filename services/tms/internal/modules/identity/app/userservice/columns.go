package userservice

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
)

func columnName(c keel.Col) string {
	return tenantdb.UsersSchema().Columns[c.Index].Name
}
