package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const (
	UserHeader         = "X-Trenova-User-Id"
	OrganizationHeader = "X-Trenova-Organization-Id"
	BusinessUnitHeader = "X-Trenova-Business-Unit-Id"
)

var sessionContextHeaders = []string{UserHeader, OrganizationHeader, BusinessUnitHeader}

func WithSessionContextHeaders(allowed []string) []string {
	merged := slices.Clone(allowed)
	for _, header := range sessionContextHeaders {
		if !slices.ContainsFunc(merged, func(existing string) bool {
			return strings.EqualFold(strings.TrimSpace(existing), header)
		}) {
			merged = append(merged, header)
		}
	}
	return merged
}

type SessionContextMiddleware struct {
	errorHandler *helpers.ErrorHandler
}

func NewSessionContextMiddleware(errorHandler *helpers.ErrorHandler) *SessionContextMiddleware {
	return &SessionContextMiddleware{errorHandler: errorHandler}
}

func (m *SessionContextMiddleware) RequireCurrent() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := authctx.GetAuthContext(c)
		if assertsOtherSession(c.Request.Header, authCtx) {
			m.errorHandler.HandleError(
				c,
				errortypes.NewSessionContextStaleError(
					authCtx.UserID,
					authCtx.OrganizationID,
					authCtx.BusinessUnitID,
				),
			)
			return
		}

		c.Next()
	}
}

func assertsOtherSession(header http.Header, authCtx *authctx.AuthContext) bool {
	return assertsOther(header, UserHeader, authCtx.UserID) ||
		assertsOther(header, OrganizationHeader, authCtx.OrganizationID) ||
		assertsOther(header, BusinessUnitHeader, authCtx.BusinessUnitID)
}

func assertsOther(header http.Header, name string, current pulid.ID) bool {
	asserted := strings.TrimSpace(header.Get(name))
	return asserted != "" && asserted != current.String()
}
