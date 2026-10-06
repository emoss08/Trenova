package supportaccess

import (
	"context"

	domain "github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const ginActiveKey = "supportaccess.active"

type activeKey struct{}

type Active struct {
	SessionID       pulid.ID
	OrganizationID  pulid.ID
	BusinessUnitID  pulid.ID
	GrantID         pulid.ID
	StaffUserID     pulid.ID
	StaffName       string
	PrincipalUserID pulid.ID
	BaseSessionID   pulid.ID
	GrantMode       domain.AccessMode
	ExpiresAt       int64
	ElevatedUntil   int64
	Now             int64
}

func (a *Active) WriteActive() bool {
	return a != nil && a.GrantMode.AllowsWrite() && a.ElevatedUntil > a.Now
}

func (a *Active) EffectiveMode() domain.AccessMode {
	if a.WriteActive() {
		return domain.AccessModeReadWrite
	}
	return domain.AccessModeReadOnly
}

func WithActive(ctx context.Context, active *Active) context.Context {
	return context.WithValue(ctx, activeKey{}, active)
}

func ActiveFrom(ctx context.Context) (*Active, bool) {
	active, ok := ctx.Value(activeKey{}).(*Active)
	return active, ok && active != nil
}

func BindActive(c *gin.Context, active *Active) {
	c.Set(ginActiveKey, active)
	c.Request = c.Request.WithContext(WithActive(c.Request.Context(), active))
}

func ActiveFromGin(c *gin.Context) (*Active, bool) {
	value, exists := c.Get(ginActiveKey)
	if !exists {
		return nil, false
	}
	active, ok := value.(*Active)
	return active, ok && active != nil
}
