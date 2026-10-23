package formulatemplatehandler

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const (
	maxDraftNameLength        = 100
	maxDraftDescriptionLength = 4000
)

type formulaDraftRequest struct {
	TemplateID  string `json:"templateId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

type explainFormulaRequest struct {
	formulaDraftRequest
	Inputs     tarifftypes.Inputs `json:"inputs"`
	ShipmentID string             `json:"shipmentId"`
}

type explainTestRequest struct {
	formulaDraftRequest
	TestName string `json:"testName"`
}

func (h *Handler) studioActor(authCtx *authctx.AuthContext) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
}

func (h *Handler) studioAvailable(c *gin.Context) bool {
	if h.studio != nil {
		return true
	}

	h.eh.HandleError(c, errortypes.NewBusinessError(
		"The formula assistant is not available on this server",
	))
	return false
}

func (r *formulaDraftRequest) toDraft() (agentports.FormulaStudioDraft, error) {
	multiErr := errortypes.NewMultiError()
	draft := agentports.FormulaStudioDraft{
		Name:        strings.TrimSpace(r.Name),
		Description: strings.TrimSpace(r.Description),
		Source:      r.Source,
	}

	if raw := strings.TrimSpace(r.TemplateID); raw != "" {
		templateID, err := pulid.Parse(raw)
		if err != nil {
			multiErr.Add("templateId", errortypes.ErrInvalid, "Invalid formula template ID")
		}
		draft.TemplateID = templateID
	}
	switch {
	case strings.TrimSpace(r.Source) == "":
		multiErr.Add("source", errortypes.ErrRequired, "Source is required")
	case len(r.Source) > tarifftypes.MaxSourceLength:
		multiErr.Add("source", errortypes.ErrInvalid, "Source cannot exceed 64 KiB")
	}
	if utf8.RuneCountInString(draft.Name) > maxDraftNameLength {
		multiErr.Add("name", errortypes.ErrInvalid, "Name cannot be longer than 100 characters")
	}
	if utf8.RuneCountInString(draft.Description) > maxDraftDescriptionLength {
		multiErr.Add(
			"description",
			errortypes.ErrInvalid,
			"Description cannot be longer than 4,000 characters",
		)
	}
	if multiErr.HasErrors() {
		return draft, multiErr
	}
	return draft, nil
}

// @Summary Explain a formula statement by statement
// @Description Previews the editor's source with the rate trace on, and asks the formula assistant to say in a sentence what each traced step works out. Every amount comes from the engine. Requires formula template read, assistant create, and access to the formula assistant.
// @ID explainFormulaDraft
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body explainFormulaRequest true "The editor's draft and the preview load"
// @Success 200 {object} agentports.ExplainFormulaResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ai/explain [post]
func (h *Handler) explainFormula(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req explainFormulaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !h.studioAvailable(c) {
		return
	}

	draft, err := req.toDraft()
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if len(req.Inputs) > formulatemplateservice.MaxPreviewInputs {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"inputs",
			errortypes.ErrInvalid,
			"Too many input values",
		))
		return
	}

	shipment, ok := h.optionalShipment(c, authCtx, req.ShipmentID)
	if !ok {
		return
	}
	shipmentID := pulid.Nil
	if shipment != nil {
		shipmentID = *shipment
	}

	resp, err := h.studio.ExplainFormula(c.Request.Context(), &agentports.ExplainFormulaRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		Draft:      draft,
		Inputs:     req.Inputs,
		ShipmentID: shipmentID,
	}, h.studioActor(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary Describe a formula's undescribed inputs
// @Description Asks the formula assistant for a doc comment for every input the draft declares without one, at most 120 characters each. Requires formula template read, assistant create, and access to the formula assistant.
// @ID describeFormulaInputs
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body formulaDraftRequest true "The editor's draft"
// @Success 200 {object} agentports.DescribeFormulaInputsResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ai/describe-inputs [post]
func (h *Handler) describeInputs(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req formulaDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !h.studioAvailable(c) {
		return
	}

	draft, err := req.toDraft()
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.studio.DescribeInputs(
		c.Request.Context(),
		&agentports.DescribeFormulaInputsRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Draft:      draft,
		},
		h.studioActor(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary Suggest tests for a formula
// @Description Asks the formula assistant for test blocks that pin the draft's behaviour at its edges, then runs each against the draft and returns it with its result. Requires formula template read, assistant create, and access to the formula assistant.
// @ID suggestFormulaTests
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body formulaDraftRequest true "The editor's draft"
// @Success 200 {object} agentports.SuggestFormulaTestsResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ai/suggest-tests [post]
func (h *Handler) suggestTests(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req formulaDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !h.studioAvailable(c) {
		return
	}

	draft, err := req.toDraft()
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.studio.SuggestTests(
		c.Request.Context(),
		&agentports.SuggestFormulaTestsRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Draft:      draft,
		},
		h.studioActor(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary Explain why a formula's test fails
// @Description Runs the draft's tests and asks the formula assistant why the named test fails, with the values the failing expectation saw, and whether the formula or the test is wrong. Without a model the cause is written from the failure and generated is false. Requires formula template read, assistant create, and access to the formula assistant.
// @ID explainFormulaTest
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body explainTestRequest true "The editor's draft and the test's name"
// @Success 200 {object} agentports.ExplainFormulaTestResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ai/explain-test [post]
func (h *Handler) explainTest(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req explainTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !h.studioAvailable(c) {
		return
	}

	draft, err := req.toDraft()
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	name := strings.TrimSpace(req.TestName)
	if name == "" || utf8.RuneCountInString(name) > formulatemplateservice.MaxTestNameLength {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"testName",
			errortypes.ErrInvalid,
			"Name the test to explain, in at most 100 characters",
		))
		return
	}

	resp, err := h.studio.ExplainTest(
		c.Request.Context(),
		&agentports.ExplainFormulaTestRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Draft:      draft,
			TestName:   name,
		},
		h.studioActor(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary Draft the note for a formula template's reviewer
// @Description Drafts a note for the reviewer from the difference between the saved content and the last approved version, and from a backtest of the saved content against recent shipments (null on a first version, or when the caller may not read shipments). Requires formula template read, assistant create, and access to the formula assistant.
// @ID draftFormulaTemplateReviewNote
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {object} agentports.DraftFormulaReviewNoteResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/review-note [post]
func (h *Handler) reviewNote(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !h.studioAvailable(c) {
		return
	}

	resp, err := h.studio.DraftReviewNote(
		c.Request.Context(),
		&agentports.DraftFormulaReviewNoteRequest{
			TenantInfo:      pagination.FromAuthAsUser(authCtx),
			TemplateID:      templateID,
			IncludeBacktest: h.mayReadShipments(c, authCtx),
		},
		h.studioActor(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary List who may review a formula template
// @Description Lists the active people in the organization, other than the caller, who hold formula template approve.
// @ID listFormulaTemplateReviewers
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {array} formulatemplateservice.Reviewer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/reviewers [get]
func (h *Handler) listReviewers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	reviewers, err := h.service.ListReviewers(
		c.Request.Context(),
		&formulatemplateservice.ListReviewersRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			TemplateID: templateID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, reviewers)
}

func (h *Handler) mayReadShipments(c *gin.Context, authCtx *authctx.AuthContext) bool {
	result, err := h.permEngine.Check(
		c.Request.Context(),
		middleware.BuildPermissionCheckRequest(
			authCtx,
			permission.ResourceShipment.String(),
			permission.OpRead,
		),
	)

	return err == nil && result != nil && result.Allowed
}
