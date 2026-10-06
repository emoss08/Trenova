package routegroup

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const (
	PublicRoutesGroup        = "edition_public_routes"
	ProtectedRoutesGroup     = "edition_protected_routes"
	ProtectedMiddlewareGroup = "edition_protected_middleware"
)

type PublicRoutes interface {
	RegisterPublicRoutes(rg *gin.RouterGroup)
}

type ProtectedRoutes interface {
	RegisterProtectedRoutes(rg *gin.RouterGroup)
}

type ProtectedMiddleware interface {
	ProtectedMiddleware() gin.HandlerFunc
}

type PublicRoutesFunc func(rg *gin.RouterGroup)

func (f PublicRoutesFunc) RegisterPublicRoutes(rg *gin.RouterGroup) {
	f(rg)
}

type ProtectedRoutesFunc func(rg *gin.RouterGroup)

func (f ProtectedRoutesFunc) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	f(rg)
}

type ProtectedMiddlewareFunc func() gin.HandlerFunc

func (f ProtectedMiddlewareFunc) ProtectedMiddleware() gin.HandlerFunc {
	return f()
}

type Registrars struct {
	fx.In

	Public     []PublicRoutes        `group:"edition_public_routes"`
	Protected  []ProtectedRoutes     `group:"edition_protected_routes"`
	Middleware []ProtectedMiddleware `group:"edition_protected_middleware"`
}

func AsPublicRoutes(constructor any) any {
	return fx.Annotate(
		constructor,
		fx.As(new(PublicRoutes)),
		fx.ResultTags(groupTag(PublicRoutesGroup)),
	)
}

func AsProtectedRoutes(constructor any) any {
	return fx.Annotate(
		constructor,
		fx.As(new(ProtectedRoutes)),
		fx.ResultTags(groupTag(ProtectedRoutesGroup)),
	)
}

func AsProtectedMiddleware(constructor any) any {
	return fx.Annotate(
		constructor,
		fx.As(new(ProtectedMiddleware)),
		fx.ResultTags(groupTag(ProtectedMiddlewareGroup)),
	)
}

func groupTag(name string) string {
	return `group:"` + name + `"`
}

func (r Registrars) RegisterPublic(rg *gin.RouterGroup) {
	for _, registrar := range r.Public {
		if registrar != nil {
			registrar.RegisterPublicRoutes(rg)
		}
	}
}

func (r Registrars) RegisterProtected(rg *gin.RouterGroup) {
	for _, registrar := range r.Protected {
		if registrar != nil {
			registrar.RegisterProtectedRoutes(rg)
		}
	}
}

func (r Registrars) UseProtectedMiddleware(rg *gin.RouterGroup) {
	for _, middleware := range r.Middleware {
		if middleware == nil {
			continue
		}
		if handler := middleware.ProtectedMiddleware(); handler != nil {
			rg.Use(handler)
		}
	}
}
