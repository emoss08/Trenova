package context

import (
	"context"
	"reflect"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
)

type Key string

const (
	UserIDKey         Key = "userId"
	BusinessUnitIDKey Key = "businessUnitId"
	OrganizationIDKey Key = "organizationId"
	SessionKey        Key = "session"
	APITokenKey       Key = "apiToken"
	AuthTypeKey       Key = "authType"
	AuthTypeSession       = "session"
	AuthTypeAPIToken      = "apiToken"
)

func SetUserID(c *gin.Context, userID pulid.ID) {
	c.Set(string(UserIDKey), userID)
}

func GetUserID(c *gin.Context) (pulid.ID, bool) {
	if val, exists := c.Get(string(UserIDKey)); exists {
		if userID, ok := val.(pulid.ID); ok {
			return userID, true
		}
	}
	var empty pulid.ID
	return empty, false
}

func SetBusinessUnitID(c *gin.Context, buID pulid.ID) {
	c.Set(string(BusinessUnitIDKey), buID)
}

func GetBusinessUnitID(c *gin.Context) (pulid.ID, bool) {
	if val, exists := c.Get(string(BusinessUnitIDKey)); exists {
		if buID, ok := val.(pulid.ID); ok {
			return buID, true
		}
	}
	var empty pulid.ID
	return empty, false
}

func SetOrganizationID(c *gin.Context, orgID pulid.ID) {
	c.Set(string(OrganizationIDKey), orgID)
}

func GetOrganizationID(c *gin.Context) (pulid.ID, bool) {
	if val, exists := c.Get(string(OrganizationIDKey)); exists {
		if orgID, ok := val.(pulid.ID); ok {
			return orgID, true
		}
	}
	var empty pulid.ID
	return empty, false
}

func SetSession(c *gin.Context, sess *session.Session) {
	c.Set(string(SessionKey), sess)
}

func GetSession(c *gin.Context) (*session.Session, bool) {
	if val, exists := c.Get(string(SessionKey)); exists {
		if sess, ok := val.(*session.Session); ok {
			return sess, true
		}
	}
	return nil, false
}

func SetAPIToken(c *gin.Context, token *tenant.APIToken) {
	c.Set(string(APITokenKey), token)
}

func GetAPIToken(c *gin.Context) (*tenant.APIToken, bool) {
	if val, exists := c.Get(string(APITokenKey)); exists {
		if token, ok := val.(*tenant.APIToken); ok {
			return token, true
		}
	}
	return nil, false
}

func SetAuthType(c *gin.Context, authType string) {
	c.Set(string(AuthTypeKey), authType)
}

func GetAuthType(c *gin.Context) (string, bool) {
	if val, exists := c.Get(string(AuthTypeKey)); exists {
		if authType, ok := val.(string); ok {
			return authType, true
		}
	}
	return "", false
}

func IsSessionAuth(c *gin.Context) bool {
	authType, exists := GetAuthType(c)
	return exists && authType == AuthTypeSession
}

func IsAPITokenAuth(c *gin.Context) bool {
	authType, exists := GetAuthType(c)
	return exists && authType == AuthTypeAPIToken
}

func SetAuthContext(c *gin.Context, userID, buID, orgID pulid.ID, authType string) {
	SetUserID(c, userID)
	SetBusinessUnitID(c, buID)
	SetOrganizationID(c, orgID)
	SetAuthType(c, authType)
}

type AuthContext struct {
	UserID         pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	AuthType       string
	Session        *session.Session
	APIToken       *tenant.APIToken
}

func GetAuthContext(c *gin.Context) *AuthContext {
	ac := &AuthContext{}

	if userID, exists := GetUserID(c); exists {
		ac.UserID = userID
	}

	if buID, exists := GetBusinessUnitID(c); exists {
		ac.BusinessUnitID = buID
	}

	if orgID, exists := GetOrganizationID(c); exists {
		ac.OrganizationID = orgID
	}

	if authType, exists := GetAuthType(c); exists {
		ac.AuthType = authType
	}

	if sess, exists := GetSession(c); exists {
		ac.Session = sess
	}

	if token, exists := GetAPIToken(c); exists {
		ac.APIToken = token
	}

	return ac
}

func WithAuth(ctx context.Context, authCtx *AuthContext) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, authCtx.UserID)
	ctx = context.WithValue(ctx, BusinessUnitIDKey, authCtx.BusinessUnitID)
	ctx = context.WithValue(ctx, OrganizationIDKey, authCtx.OrganizationID)
	ctx = context.WithValue(ctx, AuthTypeKey, authCtx.AuthType)

	if authCtx.Session != nil {
		ctx = context.WithValue(ctx, SessionKey, authCtx.Session)
	}

	if authCtx.APIToken != nil {
		ctx = context.WithValue(ctx, APITokenKey, authCtx.APIToken)
	}

	return ctx
}

func GetUserIDFromContext(ctx context.Context) (pulid.ID, bool) {
	if val := ctx.Value(UserIDKey); val != nil {
		if userID, ok := val.(pulid.ID); ok {
			return userID, true
		}
	}
	var empty pulid.ID
	return empty, false
}

func AddContextToRequest(authCtx *AuthContext, req any) {
	val := reflect.ValueOf(req)
	if val.Kind() != reflect.Pointer {
		return
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}

	fieldMappings := []struct {
		value      pulid.ID
		fieldNames []string
	}{
		{authCtx.OrganizationID, []string{"OrganizationID", "OrgID"}},
		{authCtx.BusinessUnitID, []string{"BusinessUnitID", "BuID"}},
		{authCtx.UserID, []string{"UserID"}},
	}

	for _, mapping := range fieldMappings {
		for _, fieldName := range mapping.fieldNames {
			if field := elem.FieldByName(fieldName); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(mapping.value))
				break // Move to next mapping once field is set
			}
		}
	}
}
