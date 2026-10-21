// Package responseguard holds the records a REST response carries inside
// others to what the caller may see of them. Every response goes through it:
// handlers write with helpers.JSON, which asks the guard Install put on the
// request.
//
// A handler returns whole records: a shipment with its customer, its stops'
// locations and its drivers; an invoice with its customer and factoring
// company. The record the route authorized is the caller's to see. A record of
// another kind reached through it shows its kind's reference view, and any
// other field of it only to a caller who reads that kind, as the GraphQL field
// guard decides it for the screens. Protected field groups are withheld from
// every record, wherever it sits. Both decisions are refview's and
// fieldguard's; this package only finds the records.
package responseguard

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/fieldguard"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// WithheldHeader names, without list indices, every record in the response
// that was answered with only part of its fields: each field of it that is
// null there and outside its kind's reference view was withheld, not empty.
const WithheldHeader = "Withheld-By-Access"

const (
	selfScopedKey = "responseguard.self_scoped"
	referencesKey = "responseguard.references"
)

const paginationPackage = "github.com/emoss08/trenova/pkg/pagination"

type GuardParams struct {
	fx.In

	PermissionEngine permissionports.PermissionEngine
	ReferenceViews   permissionports.ReferenceViews
}

// Guard holds what every request's guard shares: the reference view rule, the
// engine, and the type plans worked out once per response type.
type Guard struct {
	engine   permissionports.PermissionEngine
	views    permissionports.ReferenceViews
	registry *permission.Registry
	planner  *planner
}

func NewGuard(p GuardParams) *Guard {
	registry := fieldsensitivity.DefaultRegistry()
	return &Guard{
		engine:   p.PermissionEngine,
		views:    p.ReferenceViews,
		registry: registry,
		planner:  newPlanner(newKindTable(registry)),
	}
}

// Install puts the caller's guard on every request of a signed-in group. It
// goes after authentication; a response written without it is not guarded.
func (g *Guard) Install() gin.HandlerFunc {
	return func(c *gin.Context) {
		if authCtx := authctx.GetAuthContext(c); authCtx != nil {
			helpers.SetResponseGuard(c, &requestGuard{guard: g, c: c, auth: authCtx})
		}
		c.Next()
	}
}

// SelfScoped marks a route whose records are all the caller's own, such as
// the driver portal's: every record in its response is shown as a root
// record. Protected field groups still apply.
func SelfScoped() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(selfScopedKey, true)
		c.Next()
	}
}

// requestGuard is one request's answers, asked of the engine at most once per
// kind, field and group for the length of the request.
type requestGuard struct {
	guard *Guard
	c     *gin.Context
	auth  *authctx.AuthContext

	linkedOnce sync.Once
	linked     permissionports.LinkedFields

	groupsOnce sync.Once
	visibleSet permission.FieldGroupSet

	mu        sync.Mutex
	decisions map[fieldKey]bool
}

type fieldKey struct {
	kind  permission.Resource
	field string
}

// References marks a route that hands out records for another record to
// link to, such as a picker's select options: it authorizes no kind, so each
// record it returns shows its kind's reference view unless the caller reads
// the kind, as it would inside the record that links to it.
func References() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(referencesKey, true)
		c.Next()
	}
}

// Guarded is the value to write for a response: the value itself when it
// carries nothing the caller may not see, and otherwise a copy with every
// withheld field null. It names the withheld records in WithheldHeader.
func (r *requestGuard) Guarded(c *gin.Context, value any) any {
	if value == nil || c == nil || c.Request == nil {
		return value
	}
	root := reflect.ValueOf(value)
	if !r.guard.planner.mayGuard(root.Type()) {
		return value
	}

	var top *record
	if c.GetBool(referencesKey) {
		top = referenceCarrier
	}
	w := &walker{
		ctx:        c.Request.Context(),
		decider:    r,
		planner:    r.guard.planner,
		registry:   r.guard.registry,
		selfScoped: c.GetBool(selfScopedKey),
		path:       make([]string, 0, 8),
	}
	guarded, changed := w.guard(root, top, 0)
	if withheld := w.withheldRecords(); len(withheld) > 0 {
		c.Header(WithheldHeader, strings.Join(withheld, ", "))
	}
	if !changed {
		return value
	}

	return interfaceOf(guarded)
}

var (
	_ decider               = (*requestGuard)(nil)
	_ helpers.ResponseGuard = (*requestGuard)(nil)
)

func (r *requestGuard) fields() permissionports.LinkedFields {
	r.linkedOnce.Do(func() {
		engine := r.guard.engine
		auth := r.auth
		r.linked = r.guard.views.Person(
			actorutil.FromAuthContext(auth),
			func(ctx context.Context, kind permission.Resource) (bool, error) {
				if engine == nil {
					return false, nil
				}
				result, err := engine.Check(
					ctx,
					middleware.BuildPermissionCheckRequest(auth, kind.String(), permission.OpRead),
				)
				if err != nil || result == nil {
					return false, err
				}
				return result.Allowed, nil
			},
		)
	})

	return r.linked
}

func (r *requestGuard) reads(ctx context.Context, kind permission.Resource) bool {
	return r.fields().Reads(ctx, kind)
}

func (r *requestGuard) visible(ctx context.Context, kind permission.Resource, name string) bool {
	key := fieldKey{kind: kind, field: name}
	r.mu.Lock()
	decided, ok := r.decisions[key]
	r.mu.Unlock()
	if ok {
		return decided
	}

	decided = r.fields().Decide(ctx, kind, name) == permissionports.LinkedFieldVisible

	r.mu.Lock()
	if r.decisions == nil {
		r.decisions = make(map[fieldKey]bool, 32)
	}
	r.decisions[key] = decided
	r.mu.Unlock()

	return decided
}

func (r *requestGuard) groups() permission.FieldGroupSet {
	r.groupsOnce.Do(func() {
		r.visibleSet = fieldguard.Visible(r.c, r.guard.engine)
	})

	return r.visibleSet
}

// isEnvelope reports whether a struct only carries the records a route
// returned (a page of them), so each is a root record of its own.
func isEnvelope(t reflect.Type) bool {
	return t.PkgPath() == paginationPackage
}
