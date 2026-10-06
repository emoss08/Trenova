package routegroup_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/routegroup"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

type publicRoutes struct{}

func (publicRoutes) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/open", func(c *gin.Context) { c.Status(http.StatusOK) })
}

type protectedRoutes struct{}

func (protectedRoutes) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.GET("/closed", func(c *gin.Context) { c.Status(http.StatusOK) })
}

type headerMiddleware struct{}

func (headerMiddleware) ProtectedMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Edition", "cloud")
		c.Next()
	}
}

func TestAsHelpersFillTheEditionGroups(t *testing.T) {
	t.Parallel()

	var got routegroup.Registrars
	app := fx.New(
		fx.NopLogger,
		fx.Provide(
			routegroup.AsPublicRoutes(func() publicRoutes { return publicRoutes{} }),
			routegroup.AsProtectedRoutes(func() protectedRoutes { return protectedRoutes{} }),
			routegroup.AsProtectedMiddleware(func() headerMiddleware { return headerMiddleware{} }),
		),
		fx.Populate(&got),
	)
	require.NoError(t, app.Err())

	assert.Len(t, got.Public, 1)
	assert.Len(t, got.Protected, 1)
	assert.Len(t, got.Middleware, 1)
}

func TestWithoutAnEditionTheGroupsAreEmpty(t *testing.T) {
	t.Parallel()

	var got routegroup.Registrars
	app := fx.New(fx.NopLogger, fx.Populate(&got))
	require.NoError(t, app.Err())

	assert.Empty(t, got.Public)
	assert.Empty(t, got.Protected)
	assert.Empty(t, got.Middleware)

	engine := gin.New()
	got.RegisterPublic(&engine.RouterGroup)
	got.UseProtectedMiddleware(&engine.RouterGroup)
	got.RegisterProtected(&engine.RouterGroup)
	assert.Empty(t, engine.Routes())
}

func TestRegistrarsRegisterRoutesAndMiddleware(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	registrars := routegroup.Registrars{
		Public: []routegroup.PublicRoutes{
			publicRoutes{},
			nil,
			routegroup.PublicRoutesFunc(func(rg *gin.RouterGroup) {
				rg.GET("/func", func(c *gin.Context) { c.Status(http.StatusAccepted) })
			}),
		},
		Protected: []routegroup.ProtectedRoutes{
			protectedRoutes{},
			nil,
			routegroup.ProtectedRoutesFunc(func(rg *gin.RouterGroup) {
				rg.GET("/closed-func", func(c *gin.Context) { c.Status(http.StatusAccepted) })
			}),
		},
		Middleware: []routegroup.ProtectedMiddleware{
			headerMiddleware{},
			nil,
			routegroup.ProtectedMiddlewareFunc(func() gin.HandlerFunc { return nil }),
		},
	}

	engine := gin.New()
	registrars.RegisterPublic(engine.Group(""))
	protected := engine.Group("/app")
	registrars.UseProtectedMiddleware(protected)
	registrars.RegisterProtected(protected)

	serve := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	open := serve("/open")
	assert.Equal(t, http.StatusOK, open.Code)
	assert.Empty(t, open.Header().Get("X-Edition"), "public routes skip protected middleware")
	assert.Equal(t, http.StatusAccepted, serve("/func").Code)

	closed := serve("/app/closed")
	assert.Equal(t, http.StatusOK, closed.Code)
	assert.Equal(t, "cloud", closed.Header().Get("X-Edition"))
	assert.Equal(t, http.StatusAccepted, serve("/app/closed-func").Code)
}
