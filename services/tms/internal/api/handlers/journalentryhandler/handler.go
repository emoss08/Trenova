package journalentryhandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/journalentryservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Service              *journalentryservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}
type Handler struct {
	service *journalentryservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/journal-entries")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceJournalEntry.String(), permission.OpRead),
		h.listEntries,
	)
	api.GET(
		"/:journalEntryID/",
		h.pm.RequirePermission(permission.ResourceJournalEntry.String(), permission.OpRead),
		h.getEntry,
	)
	api.GET(
		"/source/:sourceObjectType/:sourceObjectID/",
		h.pm.RequirePermission(permission.ResourceJournalEntry.String(), permission.OpRead),
		h.getSource,
	)
}

func (h *Handler) listEntries(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	query := pagination.NewQueryOptions(c, auth)

	var fiscalYearID pulid.ID
	if raw := c.Query("fiscalYearId"); raw != "" {
		parsed, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		fiscalYearID = parsed
	}

	var fiscalPeriodID pulid.ID
	if raw := c.Query("fiscalPeriodId"); raw != "" {
		parsed, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		fiscalPeriodID = parsed
	}

	accountingDateStart := int64(0)
	if raw := c.Query("accountingDateStart"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		accountingDateStart = parsed
	}

	accountingDateEnd := int64(0)
	if raw := c.Query("accountingDateEnd"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		accountingDateEnd = parsed
	}

	pagination.List(c, query, h.eh, func() (*pagination.ListResult[*journalentry.Entry], error) {
		return h.service.ListEntries(
			c.Request.Context(),
			&repositories.ListJournalEntriesRequest{
				Filter:              query,
				FiscalYearID:        fiscalYearID,
				FiscalPeriodID:      fiscalPeriodID,
				ReferenceType:       c.Query("referenceType"),
				Status:              c.Query("status"),
				AccountingDateStart: accountingDateStart,
				AccountingDateEnd:   accountingDateEnd,
			},
		)
	})
}

func (h *Handler) getEntry(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("journalEntryID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.GetEntry(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		},
		id,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}

func (h *Handler) getSource(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	entity, err := h.service.GetSourceByObject(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		},
		c.Param("sourceObjectType"),
		c.Param("sourceObjectID"),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}
