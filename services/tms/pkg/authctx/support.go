package authctx

import (
	"context"

	"github.com/emoss08/trenova/pkg/auditctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// SupportKey holds the Trenova support binding of a support session on the gin context.
const SupportKey = Key("supportSession")

// Support binds a session to the Trenova support grant it was opened under. A session that
// carries it acts as the grant's support principal; the staff member it names is who is at the
// keyboard. It is part of the session's seal.
type Support struct {
	GrantID    pulid.ID `json:"grantId"`
	StaffEmail string   `json:"staffEmail"`
	StaffName  string   `json:"staffName"`
	Scope      string   `json:"scope"`
	EndsAt     int64    `json:"endsAt"`
}

type supportContextKey struct{}

// WithSupport puts the binding on ctx and marks every audit entry written under it as Trenova
// support.
func WithSupport(ctx context.Context, support *Support, principalUserID pulid.ID) context.Context {
	if support == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, supportContextKey{}, support)
	return auditctx.WithSupport(ctx, auditctx.Support{
		GrantID:         support.GrantID.String(),
		PrincipalUserID: principalUserID.String(),
		StaffEmail:      support.StaffEmail,
		StaffName:       support.StaffName,
	})
}

// SupportFrom returns the support binding of the request's session, if it is a support session.
func SupportFrom(ctx context.Context) (*Support, bool) {
	support, ok := ctx.Value(supportContextKey{}).(*Support)
	return support, ok && support != nil
}

func bindSupport(c *gin.Context, support *Support, principalUserID pulid.ID) {
	if support == nil {
		return
	}
	c.Set(string(SupportKey), support)
	if c.Request != nil {
		c.Request = c.Request.WithContext(WithSupport(c.Request.Context(), support, principalUserID))
	}
}

// GetSupport returns the support binding set on the gin context by SetSessionAuthContext.
func GetSupport(c *gin.Context) (*Support, bool) {
	value, exists := c.Get(string(SupportKey))
	if !exists {
		return nil, false
	}
	support, ok := value.(*Support)
	return support, ok && support != nil
}
