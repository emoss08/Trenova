// Package documenttemplatehandler serves the one part of the template system
// that cannot travel over GraphQL: the rendered PDF bytes.
//
// Everything else about templates is a GraphQL query or mutation. Base64 in a
// JSON envelope inflates by roughly a third and cannot stream, so a print that
// an administrator is waiting on gets its own route.
package documenttemplatehandler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documenttemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *documenttemplateservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *documenttemplateservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/document-templates")
	api.GET(
		"/versions/:versionID/preview.pdf",
		h.pm.RequirePermission(
			permission.ResourceDocumentTemplate.String(), permission.OpRead,
		),
		h.previewPDF,
	)
}

// previewPDF prints a stored version against the kind's sample data.
//
// It renders sample data, never a real customer's, because a preview is an
// authoring tool: an administrator checking a layout has no business pulling an
// arbitrary invoice through it, and the sample exercises every declared variable
// anyway.
func (h *Handler) previewPDF(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	version, err := h.service.GetVersion(
		c.Request.Context(),
		&repositories.GetDocumentTemplateVersionByIDRequest{
			VersionID:  versionID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	kind := h.service.KindOf(c.Request.Context(), version, tenantInfo)
	if kind == "" {
		h.eh.HandleError(c, errortypes.NewNotFoundError(
			"This version's template no longer exists",
		))
		return
	}

	result, err := h.service.PreviewVersion(
		c.Request.Context(),
		&services.PreviewVersionRequest{
			TenantInfo: tenantInfo,
			Kind:       kind,
			VersionID:  &versionID,
			IncludePDF: true,
		},
	)
	if err != nil {
		if errors.Is(err, services.ErrPDFRendererUnavailable) {
			// A missing renderer is a deployment condition, not a server fault,
			// and the editor shows it as such rather than as a crash.
			h.eh.HandleError(c, errortypes.NewBusinessError(
				"The PDF renderer is not available, so this template cannot be printed. "+
					"The HTML preview still works.",
			))
			return
		}
		h.eh.HandleError(c, err)
		return
	}

	if len(result.PDF) == 0 {
		h.eh.HandleError(c, errortypes.NewBusinessError(
			"This template produced no printable output. "+
				result.Diagnostics.Summary(),
		))
		return
	}

	writePreviewPDF(c, string(kind), version.VersionNumber, result.PDF)
}

// writePreviewPDF sends the printed bytes.
//
// Split out from the handler so the response contract — the content type, the
// sniffing guard, and the header-safe filename — is testable without standing up
// the renderer.
func writePreviewPDF(c *gin.Context, kind string, versionNumber int64, pdf []byte) {
	// inline, not attachment: the editor shows this in an embedded viewer.
	// The filename is built from a sanitized kind and a number, never from
	// organization-authored text, so it cannot carry a quote or a newline into
	// the header.
	c.Header("Content-Disposition", fmt.Sprintf(
		`inline; filename="%s-v%d.pdf"`,
		sanitizeFilenamePart(kind), versionNumber,
	))
	// A preview is only valid for the exact bytes that produced it, and those
	// change as the author edits. Caching it would show a stale page after a
	// save.
	c.Header("Cache-Control", "no-store")
	// This response is a PDF and nothing else. Letting a browser sniff it would
	// hand organization-authored markup an app-origin document.
	c.Header("X-Content-Type-Options", "nosniff")

	c.Data(http.StatusOK, "application/pdf", pdf)
}

// sanitizeFilenamePart keeps a kind safe to interpolate into a header.
//
// Kinds are registry constants today, so nothing hostile can reach here — the
// filter exists so that stays true if a kind ever becomes data.
func sanitizeFilenamePart(value string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, value)
}
