package fieldguard

import (
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/gin-gonic/gin"
)

func NewListGuard(visible func() protectedfield.GroupSet) listquery.FieldGuard {
	held := sync.OnceValue(visible)

	return func(table, field string) bool {
		group, protected := protectedfield.ColumnGroup(table, field)
		return protected && !held().Has(group)
	}
}

func InstallListGuard(engine permissionports.PermissionEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		guard := NewListGuard(func() protectedfield.GroupSet { return Visible(c, engine) })
		c.Request = c.Request.WithContext(listquery.WithFieldGuard(c.Request.Context(), guard))
		c.Next()
	}
}
