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
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const (
	maxDraftNameLength        = 100
	maxDraftDescriptionLength = 4000
	maxSampleValues           = formulatemplate.MaxVariableDefinitions + 40
	maxExistingScenarios      = 50
)

type formulaDraftContentRequest struct {
	TemplateID           string                              `json:"templateId"`
	Name                 string                              `json:"name"`
	Description          string                              `json:"description"`
	Type                 string                              `json:"type"`
	SchemaID             string                              `json:"schemaId"`
	Expression           string                              `json:"expression"`
	VariableDefinitions  []*formulatypes.VariableDefinition  `json:"variableDefinitions"`
	BreakdownDefinitions []*formulatypes.BreakdownDefinition `json:"breakdownDefinitions"`
	MinCharge            *string                             `json:"minCharge"`
	MaxCharge            *string                             `json:"maxCharge"`
	RoundingMode         string                              `json:"roundingMode"`
	RoundingPrecision    *int32                              `json:"roundingPrecision"`
}

type explainFormulaRequest struct {
	formulaDraftContentRequest
	SampleValues map[string]any `json:"sampleValues"`
	ShipmentID   string         `json:"shipmentId"`
}

type suggestScenariosRequest struct {
	formulaDraftContentRequest
	Existing []agentports.ExistingFormulaScenario `json:"existing"`
}

type explainScenarioRequest struct {
	Candidate *formulatemplateservice.TestCaseCandidate `json:"candidate"`
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

func (r *formulaDraftContentRequest) toDraft(
	requireExpression bool,
) (agentports.FormulaStudioDraft, error) {
	multiErr := errortypes.NewMultiError()
	draft := agentports.FormulaStudioDraft{
		Name:        strings.TrimSpace(r.Name),
		Description: strings.TrimSpace(r.Description),
		Type:        formulatemplate.TemplateType(strings.TrimSpace(r.Type)),
		SchemaID:    strings.TrimSpace(r.SchemaID),
		Content: formulatemplate.ScenarioContent{
			Expression:           r.Expression,
			VariableDefinitions:  r.VariableDefinitions,
			BreakdownDefinitions: r.BreakdownDefinitions,
		},
	}
	if draft.SchemaID == "" {
		draft.SchemaID = "shipment"
	}

	if raw := strings.TrimSpace(r.TemplateID); raw != "" {
		templateID, err := pulid.Parse(raw)
		if err != nil {
			multiErr.Add("templateId", errortypes.ErrInvalid, "Invalid formula template ID")
		}
		draft.TemplateID = templateID
	}
	if requireExpression && strings.TrimSpace(r.Expression) == "" {
		multiErr.Add("expression", errortypes.ErrRequired, "An expression is required")
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
	if draft.Type != "" && draft.Type != formulatemplate.TemplateTypeFreightCharge &&
		draft.Type != formulatemplate.TemplateTypeAccessorialCharge {
		multiErr.Add("type", errortypes.ErrInvalid, "Must be FreightCharge or AccessorialCharge")
	}

	minCharge, err := parseGuardrailCharge("minCharge", r.MinCharge)
	if err != nil {
		return draft, err
	}
	maxCharge, err := parseGuardrailCharge("maxCharge", r.MaxCharge)
	if err != nil {
		return draft, err
	}
	policy, err := parseRoundingPolicy(r.RoundingMode, r.RoundingPrecision)
	if err != nil {
		return draft, err
	}
	draft.Content.MinCharge = minCharge
	draft.Content.MaxCharge = maxCharge
	draft.Content.RoundingMode = policy.RoundingMode
	draft.Content.RoundingPrecision = policy.RoundingPrecision

	formulatemplateservice.ValidateScenarioContent(multiErr, "", &draft.Content)
	if multiErr.HasErrors() {
		return draft, multiErr
	}

	return draft, nil
}

// @Summary Explain a formula statement by statement
// @Description Prices the editor's draft with the formula engine, tracing every statement, and asks the formula assistant to say in a sentence what each one works out. Every amount comes from the engine. Requires formula template read, assistant create, and access to the formula assistant.
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

	draft, err := req.toDraft(true)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if len(req.SampleValues) > maxSampleValues {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"sampleValues",
			errortypes.ErrInvalid,
			"Too many sample values",
		))
		return
	}

	shipmentID := pulid.Nil
	if raw := strings.TrimSpace(req.ShipmentID); raw != "" {
		shipmentID, err = pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		if !h.allowShipmentRead(c, authCtx) {
			return
		}
	}

	resp, err := h.studio.ExplainFormula(c.Request.Context(), &agentports.ExplainFormulaRequest{
		TenantInfo:   pagination.FromAuthAsUser(authCtx),
		Draft:        draft,
		SampleValues: req.SampleValues,
		ShipmentID:   shipmentID,
	}, h.studioActor(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary Describe a formula's undescribed inputs
// @Description Asks the formula assistant for a description of every input in the draft whose description is blank, at most 120 characters each. Requires formula template read, assistant create, and access to the formula assistant.
// @ID describeFormulaInputs
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body formulaDraftContentRequest true "The editor's draft"
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

	var req formulaDraftContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !h.studioAvailable(c) {
		return
	}

	draft, err := req.toDraft(false)
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

// @Summary Suggest edge-case scenarios for a formula
// @Description Asks the formula assistant for three to five scenarios that pin the draft's behaviour at its edges, each priced by the formula engine as the billed amount with the charge policy applied. Requires formula template read, assistant create, and access to the formula assistant.
// @ID suggestFormulaScenarios
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body suggestScenariosRequest true "The editor's draft and the scenarios it already has"
// @Success 200 {object} agentports.SuggestFormulaScenariosResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ai/suggest-scenarios [post]
func (h *Handler) suggestScenarios(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req suggestScenariosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !h.studioAvailable(c) {
		return
	}

	draft, err := req.toDraft(true)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if len(req.Existing) > maxExistingScenarios {
		req.Existing = req.Existing[:maxExistingScenarios]
	}

	resp, err := h.studio.SuggestScenarios(
		c.Request.Context(),
		&agentports.SuggestFormulaScenariosRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Draft:      draft,
			Existing:   req.Existing,
		},
		h.studioActor(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary Explain why a test scenario no longer bills what it expects
// @Description Evaluates the scenario's inputs on the content its expectation was recorded on and on the editor's content, returns the rows that differ, and asks the formula assistant for the cause and a verdict. Without a model the cause is written from the trace and generated is false. Requires formula template read, assistant create, and access to the formula assistant.
// @ID explainFormulaTemplateTestCase
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param testCaseID path string true "Test scenario ID"
// @Param request body explainScenarioRequest true "The editor's content"
// @Success 200 {object} agentports.ExplainFormulaScenarioResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases/{testCaseID}/explain [post]
func (h *Handler) explainScenario(c *gin.Context) {
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

	var req explainScenarioRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if req.Candidate == nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"candidate",
			errortypes.ErrRequired,
			"The editor's content is required to explain a scenario",
		))
		return
	}
	if !h.studioAvailable(c) {
		return
	}

	resp, err := h.studio.ExplainScenario(
		c.Request.Context(),
		&agentports.ExplainFormulaScenarioRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			TemplateID: templateID,
			TestCaseID: testCaseID,
			Candidate:  *req.Candidate,
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
