package authctx

import (
	"reflect"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type Key string

const (
	UserIDKey = Key("userId")
	BuIDKey   = Key("businessUnitId")
	OrgIDKey  = Key("organizationId")
	TypeKey   = Key("principalType")
	ActorKey  = Key("principalId")
	APIKeyID  = Key("apiKeyId")
)

const (
	PrincipalTypeUser   = "session_user"
	PrincipalTypeAPIKey = "api_key"
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
	c.Set(string(BuIDKey), buID)
}

func GetBusinessUnitID(c *gin.Context) (pulid.ID, bool) {
	if val, exists := c.Get(string(BuIDKey)); exists {
		if buID, ok := val.(pulid.ID); ok {
			return buID, true
		}
	}
	var empty pulid.ID
	return empty, false
}

func SetOrganizationID(c *gin.Context, orgID pulid.ID) {
	c.Set(string(OrgIDKey), orgID)
}

func GetOrganizationID(c *gin.Context) (pulid.ID, bool) {
	if val, exists := c.Get(string(OrgIDKey)); exists {
		if orgID, ok := val.(pulid.ID); ok {
			return orgID, true
		}
	}
	var empty pulid.ID
	return empty, false
}

func SetAuthContext(c *gin.Context, userID, buID, orgID pulid.ID) {
	c.Set(string(TypeKey), PrincipalTypeUser)
	c.Set(string(ActorKey), userID)
	SetUserID(c, userID)
	SetBusinessUnitID(c, buID)
	SetOrganizationID(c, orgID)
}

func SetAPIKeyContext(
	c *gin.Context,
	principalID, buID, orgID pulid.ID,
) {
	c.Set(string(TypeKey), PrincipalTypeAPIKey)
	c.Set(string(ActorKey), principalID)
	c.Set(string(APIKeyID), principalID)
	SetBusinessUnitID(c, buID)
	SetOrganizationID(c, orgID)
}

type AuthContext struct {
	PrincipalType  string
	PrincipalID    pulid.ID
	UserID         pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	APIKeyID       pulid.ID
}

func (ac *AuthContext) IsAPIKey() bool {
	if ac == nil {
		return false
	}
	return ac.PrincipalType == PrincipalTypeAPIKey
}

func GetAuthContext(c *gin.Context) *AuthContext {
	ac := &AuthContext{}

	if val, exists := c.Get(string(TypeKey)); exists {
		if principalType, ok := val.(string); ok {
			ac.PrincipalType = principalType
		}
	}

	if val, exists := c.Get(string(ActorKey)); exists {
		if principalID, ok := val.(pulid.ID); ok {
			ac.PrincipalID = principalID
		}
	}

	if userID, exists := GetUserID(c); exists {
		ac.UserID = userID
	}

	if buID, exists := GetBusinessUnitID(c); exists {
		ac.BusinessUnitID = buID
	}

	if orgID, exists := GetOrganizationID(c); exists {
		ac.OrganizationID = orgID
	}

	if val, exists := c.Get(string(APIKeyID)); exists {
		if apiKeyID, ok := val.(pulid.ID); ok {
			ac.APIKeyID = apiKeyID
		}
	}

	switch ac.PrincipalType {
	case "":
		switch {
		case ac.APIKeyID.IsNotNil():
			ac.PrincipalType = PrincipalTypeAPIKey
		case ac.UserID.IsNotNil():
			ac.PrincipalType = PrincipalTypeUser
		}
	}

	if ac.PrincipalID.IsNil() {
		switch ac.PrincipalType {
		case PrincipalTypeAPIKey:
			ac.PrincipalID = ac.APIKeyID
		case PrincipalTypeUser:
			ac.PrincipalID = ac.UserID
		}
	}

	if ac.PrincipalType == PrincipalTypeAPIKey && ac.APIKeyID.IsNil() {
		ac.APIKeyID = ac.PrincipalID
	}

	return ac
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
