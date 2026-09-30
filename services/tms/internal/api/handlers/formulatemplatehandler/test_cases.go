package formulatemplatehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// @Summary List a formula template's saved test scenarios
// @ID listFormulaTemplateTestCases
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {array} formulatemplate.TestCase
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases [get]
func (h *Handler) listTestCases(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	cases, err := h.service.ListTestCases(c.Request.Context(), repositories.ListTestCasesRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		TemplateID: templateID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, cases)
}

// @Summary Create a test scenario for a formula template
// @ID createFormulaTemplateTestCase
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body formulatemplateservice.TestCaseInput true "Test scenario"
// @Success 200 {object} formulatemplate.TestCase
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases [post]
func (h *Handler) createTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var input formulatemplateservice.TestCaseInput
	if err = c.ShouldBindJSON(&input); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.CreateTestCase(
		c.Request.Context(),
		&formulatemplateservice.CreateTestCaseRequest{
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
			TemplateID:    templateID,
			TestCaseInput: input,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, created)
}

// @Summary Update a formula template test scenario
// @ID updateFormulaTemplateTestCase
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param testCaseID path string true "Test scenario ID"
// @Param request body formulatemplateservice.UpdateTestCaseRequest true "Test scenario"
// @Success 200 {object} formulatemplate.TestCase
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases/{testCaseID} [put]
func (h *Handler) updateTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req formulatemplateservice.UpdateTestCaseRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	req.TemplateID = templateID
	req.TestCaseID = testCaseID

	updated, err := h.service.UpdateTestCase(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Delete a formula template test scenario
// @ID deleteFormulaTemplateTestCase
// @Tags Formula Templates
// @Param templateID path string true "Formula template ID"
// @Param testCaseID path string true "Test scenario ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases/{testCaseID} [delete]
func (h *Handler) deleteTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.DeleteTestCase(c.Request.Context(), repositories.GetTestCaseByIDRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		TemplateID: templateID,
		TestCaseID: testCaseID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Run a formula template's test scenarios
// @ID runFormulaTemplateTestCases
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body formulatemplateservice.RunTestCasesRequest true "Optional candidate content"
// @Success 200 {object} formulatemplateservice.RunTestCasesResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases/run [post]
func (h *Handler) runTestCases(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req formulatemplateservice.RunTestCasesRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	req.TemplateID = templateID

	result, err := h.service.RunTestCases(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
