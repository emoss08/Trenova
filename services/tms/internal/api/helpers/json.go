package helpers

import "github.com/gin-gonic/gin"

const responseGuardKey = "helpers.response_guard"

// ResponseGuard decides what of a response value a request's caller may see
// (internal/api/responseguard): the value itself, or a copy with the fields
// withheld from them answered null.
type ResponseGuard interface {
	Guarded(c *gin.Context, value any) any
}

// SetResponseGuard installs a request's guard; JSON writes through it.
func SetResponseGuard(c *gin.Context, guard ResponseGuard) {
	c.Set(responseGuardKey, guard)
}

// JSON writes every REST response. A signed-in request's response goes
// through its guard, so a record carried inside the one the route returned
// shows only what the caller may see of it.
func JSON(c *gin.Context, status int, value any) {
	if guard, ok := ContextValue[ResponseGuard](c, responseGuardKey); ok && guard != nil {
		value = guard.Guarded(c, value)
	}
	c.JSON(status, value)
}
