package middleware

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
)

// PasswordChangeMiddleware refuses every request from a session whose user still owes
// a password change.
//
// The client has a step that puts the change in front of the user, but a step is a
// suggestion: without this, anybody holding the session cookie can skip the screen and
// call the API directly. The flag rides on the session, set at login, so enforcing it
// costs no extra lookup.
type PasswordChangeMiddleware struct {
	errorHandler *helpers.ErrorHandler
}

func NewPasswordChangeMiddleware(errorHandler *helpers.ErrorHandler) *PasswordChangeMiddleware {
	return &PasswordChangeMiddleware{errorHandler: errorHandler}
}

// allowedPaths are the only things a session in this state may reach: the endpoint
// that resolves it, the one that tells the client who it is (so the app can render the
// change screen at all), and the way out.
var allowedPaths = []string{
	"/api/v1/users/me/change-password/",
	"/api/v1/users/me/",
	"/api/v1/auth/logout",
	"/api/v1/auth/csrf",
}

func (m *PasswordChangeMiddleware) RequireCurrentPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !mustChangePassword(c) || isPasswordChangeExempt(c.Request.URL.Path) {
			c.Next()
			return
		}

		m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
			"Your password must be changed before you can continue.",
		))
	}
}

func mustChangePassword(c *gin.Context) bool {
	value, ok := c.Get(string(authctx.MustChangePasswordKey))
	if !ok {
		return false
	}
	required, ok := value.(bool)
	return ok && required
}

// isPasswordChangeExempt compares against the path with any trailing slash normalised,
// because the router registers some of these with one and some without.
func isPasswordChangeExempt(path string) bool {
	normalised := strings.TrimSuffix(path, "/")
	for _, allowed := range allowedPaths {
		if normalised == strings.TrimSuffix(allowed, "/") {
			return true
		}
	}
	return false
}
