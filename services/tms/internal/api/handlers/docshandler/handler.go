package docshandler

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/bytedance/sonic"
	generateddocs "github.com/emoss08/trenova/docs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"
)

type Params struct {
	fx.In

	Config *config.Config
}

type Handler struct {
	swaggerBytes  []byte
	openapi3Bytes []byte
}

func New(p Params) *Handler {
	swaggerSpec := applyRuntimeMetadata(
		generateddocs.ReadSpec(),
		p.Config.App.Version,
		"Trenova TMS API",
		"API documentation for Trenova TMS. Protected routes accept either a Bearer token in the Authorization header or an authenticated session cookie.",
		"/api/v1",
	)

	openapi3Spec := applyRuntimeMetadata(
		generateddocs.ReadOpenAPI3Spec(),
		p.Config.App.Version,
		"Trenova TMS API",
		"API documentation for Trenova TMS. Protected routes accept either a Bearer token in the Authorization header or an authenticated session cookie.",
		"/api/v1",
	)

	return &Handler{
		swaggerBytes:  swaggerSpec,
		openapi3Bytes: openapi3Spec,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/reference", h.reference)
	rg.GET("/openapi/swagger.json", h.swaggerSpec)
	rg.GET("/openapi/openapi-3.json", h.openAPI3Spec)
	rg.GET("/openapi/openapi-3.yaml", h.openAPI3YAMLSpec)
}

func (h *Handler) swaggerSpec(c *gin.Context) {
	c.Data(http.StatusOK, "application/json; charset=utf-8", h.swaggerBytes)
}

func (h *Handler) openAPI3Spec(c *gin.Context) {
	c.Data(http.StatusOK, "application/json; charset=utf-8", h.openapi3Bytes)
}

func (h *Handler) openAPI3YAMLSpec(c *gin.Context) {
	var doc map[string]any
	if err := sonic.Unmarshal(h.openapi3Bytes, &doc); err != nil {
		c.Data(
			http.StatusInternalServerError,
			"text/plain; charset=utf-8",
			[]byte("failed to render OpenAPI 3 YAML"),
		)
		return
	}

	content, err := yaml.Marshal(doc)
	if err != nil {
		c.Data(
			http.StatusInternalServerError,
			"text/plain; charset=utf-8",
			[]byte("failed to render OpenAPI 3 YAML"),
		)
		return
	}

	c.Data(http.StatusOK, "application/yaml; charset=utf-8", content)
}

func (h *Handler) reference(c *gin.Context) {
	specURL := template.HTMLEscapeString("/api/v1/openapi/openapi-3.json")
	jsSpecURL := template.JSEscapeString(specURL)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Trenova API Reference</title>
  <meta name="description" content="Professional API reference for the Trenova TMS backend.">
  <style>
    html, body, #app {
      margin: 0;
      min-height: 100%;
    }
  </style>
</head>
<body>
  <div id="app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  <script>
    Scalar.createApiReference('#app', {
      url: '`+jsSpecURL+`',
      theme: 'default',
      layout: 'modern',
      darkMode: true,
      defaultOpenFirstTag: true,
      defaultOpenAllTags: false,
      documentDownloadType: 'both',
      operationTitleSource: 'summary',
      searchHotKey: 'k',
      showSidebar: true,
      metaData: {
        title: 'Trenova TMS API Reference',
        description: 'Grouped API reference for the Trenova TMS service.',
      },
    })
  </script>
</body>
</html>`))
}

func applyRuntimeMetadata(
	spec []byte,
	version string,
	title string,
	description string,
	basePath string,
) []byte {
	var doc map[string]any
	if err := sonic.Unmarshal(spec, &doc); err != nil {
		return spec
	}

	info, _ := doc["info"].(map[string]any)
	if info == nil {
		info = make(map[string]any)
		doc["info"] = info
	}

	info["version"] = version
	info["title"] = title
	info["description"] = description

	if _, isOpenAPI3 := doc["openapi"]; isOpenAPI3 {
		doc["servers"] = []map[string]any{
			{"url": basePath},
		}
		delete(doc, "basePath")
	} else {
		doc["basePath"] = basePath
	}

	rendered, err := sonic.Marshal(doc)
	if err != nil {
		return spec
	}

	return bytes.TrimSpace(rendered)
}
