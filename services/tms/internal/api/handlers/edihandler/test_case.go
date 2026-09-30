package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerTestCaseRoutes(testCases *gin.RouterGroup) {
	testCases.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTestCases,
	)
	testCases.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createTestCase,
	)
	testCases.GET(
		"/:testCaseID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTestCase,
	)
	testCases.PUT(
		"/:testCaseID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateTestCase,
	)
	testCases.DELETE(
		"/:testCaseID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpDelete),
		h.deleteTestCase,
	)
	testCases.POST(
		"/:testCaseID/preview/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.previewTestCase,
	)
}

func (h *Handler) listTestCases(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	profileID, _ := pulid.MustParse(helpers.QueryString(c, "partnerDocumentProfileId", ""))
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDITestCase], error) {
		return h.service.ListTestCases(
			c.Request.Context(),
			&repositories.ListEDITestCasesRequest{
				Filter:                   req,
				PartnerDocumentProfileID: profileID,
			},
		)
	})
}

type testCaseRequest struct {
	PartnerDocumentProfileID pulid.ID            `json:"partnerDocumentProfileId"`
	Name                     string              `json:"name"`
	Description              string              `json:"description"`
	Payload                  edi.DocumentPayload `json:"payload"`
	ExpectedWarnings         int                 `json:"expectedWarnings"`
	ExpectedErrors           int                 `json:"expectedErrors"`
	ExpectedWarningCodes     []string            `json:"expectedWarningCodes"`
	ExpectedErrorCodes       []string            `json:"expectedErrorCodes"`
	Version                  int64               `json:"version"`
}

func (r *testCaseRequest) toServiceRequest(
	testCaseID pulid.ID,
	tenantInfo pagination.TenantInfo,
) *ediservice.SaveEDITestCaseRequest {
	return &ediservice.SaveEDITestCaseRequest{
		TenantInfo:               tenantInfo,
		ID:                       testCaseID,
		PartnerDocumentProfileID: r.PartnerDocumentProfileID,
		Name:                     r.Name,
		Description:              r.Description,
		Payload:                  r.Payload,
		ExpectedWarnings:         r.ExpectedWarnings,
		ExpectedErrors:           r.ExpectedErrors,
		ExpectedWarningCodes:     r.ExpectedWarningCodes,
		ExpectedErrorCodes:       r.ExpectedErrorCodes,
		Version:                  r.Version,
	}
}

func (h *Handler) createTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(testCaseRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	created, err := h.service.CreateTestCase(
		c.Request.Context(),
		req.toServiceRequest(pulid.Nil, pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		}),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) getTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	testCase, err := h.service.GetTestCase(
		c.Request.Context(),
		repositories.GetEDITestCaseByIDRequest{
			ID:         testCaseID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, testCase)
}

func (h *Handler) updateTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(testCaseRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	updated, err := h.service.UpdateTestCase(
		c.Request.Context(),
		req.toServiceRequest(testCaseID, pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		}),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = h.service.DeleteTestCase(
		c.Request.Context(),
		repositories.DeleteEDITestCaseRequest{
			ID:         testCaseID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) previewTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	preview, err := h.service.PreviewTestCase(
		c.Request.Context(),
		testCaseID,
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}
