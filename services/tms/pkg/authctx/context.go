//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package authctx

import (
	"context"
	"reflect"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type Key string

const (
	UserIDKey                 = Key("userId")
	BuIDKey                   = Key("businessUnitId")
	OrgIDKey                  = Key("organizationId")
	TypeKey                   = Key("principalType")
	ActorKey                  = Key("principalId")
	APIKeyID                  = Key("apiKeyId")
	SessionID                 = Key("sessionId")
	ActiveRoleIDsKey          = Key("activeRoleIds")
	RequiresRoleActivationKey = Key("requiresRoleActivation")
	AuthProviderKey           = Key("authProvider")
	ExternalIdentityIDKey     = Key("externalIdentityId")
	ExternalSubjectKey        = Key("externalSubject")
	AuthenticatorAALKey       = Key("authenticatorAal")
	FederationFALKey          = Key("federationFal")
	MFAAuthenticatedAtKey     = Key("mfaAuthenticatedAt")
	LastReauthenticatedAtKey  = Key("lastReauthenticatedAt")
	RiskDecisionKey           = Key("riskDecision")
	RiskDecisionIDKey         = Key("riskDecisionId")
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

type SessionAuthContextParams struct {
	SessionID              pulid.ID
	UserID                 pulid.ID
	BusinessUnitID         pulid.ID
	OrganizationID         pulid.ID
	ActiveRoleIDs          []pulid.ID
	RequiresRoleActivation bool
	AuthProvider           string
	ExternalIdentityID     string
	ExternalSubject        string
	AuthenticatorAAL       int
	FederationFAL          int
	MFAAuthenticatedAt     int64
	LastReauthenticatedAt  int64
	RiskDecision           string
	RiskDecisionID         pulid.ID
}

func SetSessionAuthContext(c *gin.Context, p SessionAuthContextParams) {
	SetAuthContext(c, p.UserID, p.BusinessUnitID, p.OrganizationID)
	c.Set(string(SessionID), p.SessionID)
	c.Set(string(ActiveRoleIDsKey), p.ActiveRoleIDs)
	c.Set(string(RequiresRoleActivationKey), p.RequiresRoleActivation)
	c.Set(string(AuthProviderKey), p.AuthProvider)
	c.Set(string(ExternalIdentityIDKey), p.ExternalIdentityID)
	c.Set(string(ExternalSubjectKey), p.ExternalSubject)
	c.Set(string(AuthenticatorAALKey), p.AuthenticatorAAL)
	c.Set(string(FederationFALKey), p.FederationFAL)
	c.Set(string(MFAAuthenticatedAtKey), p.MFAAuthenticatedAt)
	c.Set(string(LastReauthenticatedAtKey), p.LastReauthenticatedAt)
	c.Set(string(RiskDecisionKey), p.RiskDecision)
	c.Set(string(RiskDecisionIDKey), p.RiskDecisionID)
	c.Request = c.Request.WithContext(WithSessionRoleActivation(
		c.Request.Context(),
		p.ActiveRoleIDs,
		p.RequiresRoleActivation,
	))
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
	PrincipalType          string
	PrincipalID            pulid.ID
	UserID                 pulid.ID
	BusinessUnitID         pulid.ID
	OrganizationID         pulid.ID
	APIKeyID               pulid.ID
	SessionID              pulid.ID
	ActiveRoleIDs          []pulid.ID
	RequiresRoleActivation bool
	AuthProvider           string
	ExternalIdentityID     string
	ExternalSubject        string
	AuthenticatorAAL       int
	FederationFAL          int
	MFAAuthenticatedAt     int64
	LastReauthenticatedAt  int64
	RiskDecision           string
	RiskDecisionID         pulid.ID
}

func (ac *AuthContext) IsAPIKey() bool {
	if ac == nil {
		return false
	}
	return ac.PrincipalType == PrincipalTypeAPIKey
}

func GetAuthContext(c *gin.Context) *AuthContext {
	ac := authContextFromGin(c)
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

func authContextFromGin(c *gin.Context) *AuthContext {
	return &AuthContext{
		PrincipalType: helpers.ContextValueOr[string](
			c,
			string(TypeKey),
			"",
		),
		PrincipalID: helpers.ContextValueOr[pulid.ID](
			c,
			string(ActorKey),
			pulid.Nil,
		),
		UserID: helpers.ContextValueOr[pulid.ID](
			c,
			string(UserIDKey),
			pulid.Nil,
		),
		BusinessUnitID: helpers.ContextValueOr[pulid.ID](
			c,
			string(BuIDKey),
			pulid.Nil,
		),
		OrganizationID: helpers.ContextValueOr[pulid.ID](
			c,
			string(OrgIDKey),
			pulid.Nil,
		),
		APIKeyID: helpers.ContextValueOr[pulid.ID](
			c,
			string(APIKeyID),
			pulid.Nil,
		),
		SessionID: helpers.ContextValueOr[pulid.ID](
			c,
			string(SessionID),
			pulid.Nil,
		),
		ActiveRoleIDs: helpers.ContextValueOr[[]pulid.ID](
			c,
			string(ActiveRoleIDsKey),
			nil,
		),
		RequiresRoleActivation: helpers.ContextValueOr[bool](
			c,
			string(RequiresRoleActivationKey),
			false,
		),
		AuthProvider: helpers.ContextValueOr[string](
			c,
			string(AuthProviderKey),
			"",
		),
		ExternalIdentityID: helpers.ContextValueOr[string](
			c,
			string(ExternalIdentityIDKey),
			"",
		),
		ExternalSubject: helpers.ContextValueOr[string](
			c,
			string(ExternalSubjectKey),
			"",
		),
		AuthenticatorAAL: helpers.ContextValueOr[int](
			c,
			string(AuthenticatorAALKey),
			0,
		),
		FederationFAL: helpers.ContextValueOr[int](
			c,
			string(FederationFALKey),
			0,
		),
		MFAAuthenticatedAt: helpers.ContextValueOr[int64](
			c,
			string(MFAAuthenticatedAtKey),
			0,
		),
		LastReauthenticatedAt: helpers.ContextValueOr[int64](
			c,
			string(LastReauthenticatedAtKey),
			0,
		),
		RiskDecision: helpers.ContextValueOr[string](
			c,
			string(RiskDecisionKey),
			"",
		),
		RiskDecisionID: helpers.ContextValueOr[pulid.ID](
			c,
			string(RiskDecisionIDKey),
			pulid.Nil,
		),
	}
}

type sessionRoleActivationContextKey struct{}

type SessionRoleActivation struct {
	ActiveRoleIDs      []pulid.ID
	RequiresActivation bool
}

func WithSessionRoleActivation(
	ctx context.Context,
	activeRoleIDs []pulid.ID,
	requiresActivation bool,
) context.Context {
	ids := make([]pulid.ID, len(activeRoleIDs))
	copy(ids, activeRoleIDs)
	return context.WithValue(ctx, sessionRoleActivationContextKey{}, SessionRoleActivation{
		ActiveRoleIDs:      ids,
		RequiresActivation: requiresActivation,
	})
}

func GetSessionRoleActivation(ctx context.Context) (SessionRoleActivation, bool) {
	value, ok := ctx.Value(sessionRoleActivationContextKey{}).(SessionRoleActivation)
	return value, ok
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
