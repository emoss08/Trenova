package billingqueuehandler

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/formulavisibility"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// registerReviewRoutes adds what a biller does from one item: its place in the
// queue, its activity, settling its checks, releasing a hold and posting, and
// approving several items as one job.
func (h *Handler) registerReviewRoutes(api *gin.RouterGroup) {
	read := h.pm.RequirePermission(permission.ResourceBillingQueue.String(), permission.OpRead)
	update := h.pm.RequirePermission(permission.ResourceBillingQueue.String(), permission.OpUpdate)

	api.GET("/summaries/", read, h.summaries)
	api.POST("/bulk/", update, h.startBulk)
	api.GET("/bulk/:runID/", read, h.getBulk)
	api.POST("/bulk/:runID/cancel/", update, h.cancelBulk)
	api.GET("/:itemID/neighbors/", read, h.neighbors)
	api.GET("/:itemID/activity/", read, h.activity)
	api.POST("/:itemID/issues/:issueID/resolve/", update, h.resolveIssue)
	api.POST("/:itemID/issues/:issueID/undo/", update, h.undoIssue)
	api.POST("/:itemID/release/", update, h.release)
	api.POST("/:itemID/post/", update, h.post)
	api.POST(
		"/pod-signatures/redetect/",
		update,
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpUpdate),
		h.redetectPODSignatures,
	)
}

// @Summary Re-read the signatures on queued PODs
// @Description Runs document intelligence again on the current proof-of-delivery documents of every shipment still in the billing queue whose signature nobody has read yet, so each one is marked signed or unsigned. Documents a person already marked, documents already read, and documents uploaded without machine reading are left alone and counted. Requires billing queue update and document update.
// @ID redetectBillingQueuePODSignatures
// @Tags Billing Queue
// @Produce json
// @Success 200 {object} billingqueueports.RedetectPODSignaturesResult
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/pod-signatures/redetect/ [post]
func (h *Handler) redetectPODSignatures(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.review.RedetectPODSignatures(
		c.Request.Context(),
		&billingqueueports.RedetectPODSignaturesRequest{
			TenantInfo: pagination.FromAuth(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) itemID(c *gin.Context) (pulid.ID, bool) {
	id, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, false
	}

	return id, true
}

// @Summary Get live state for billing queue rows
// @ID getBillingQueueSummaries
// @Tags Billing Queue
// @Produce json
// @Param ids query string true "Comma-separated item IDs (at most 200)"
// @Security BearerAuth
// @Router /billing-queue/summaries/ [get]
func (h *Handler) summaries(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	raw := strings.Split(c.Query("ids"), ",")
	ids := make([]pulid.ID, 0, len(raw))
	for _, part := range raw {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		id, err := pulid.MustParse(part)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		ids = append(ids, id)
	}

	out, err := h.review.Summaries(
		c.Request.Context(),
		&billingqueueports.ListBillingQueueSummariesRequest{
			TenantInfo: pagination.FromAuth(authCtx),
			ItemIDs:    ids,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, gin.H{"results": out})
}

// @Summary Get an item's neighbours in the queue
// @ID getBillingQueueNeighbors
// @Tags Billing Queue
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Param query query string false "Search query, as the list was read"
// @Security BearerAuth
// @Router /billing-queue/{itemID}/neighbors/ [get]
func (h *Handler) neighbors(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, ok := h.itemID(c)
	if !ok {
		return
	}

	out, err := h.review.Neighbors(
		c.Request.Context(),
		&billingqueueports.GetBillingQueueNeighborsRequest{
			ItemID:        itemID,
			Filter:        pagination.NewQueryOptions(c, authCtx),
			IncludePosted: helpers.QueryBool(c, "includePosted"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, out)
}

// @Summary List an item's activity, newest first
// @ID listBillingQueueActivity
// @Tags Billing Queue
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param beforeAt query int false "Continue before this entry's time"
// @Param beforeId query string false "Continue before this entry"
// @Security BearerAuth
// @Router /billing-queue/{itemID}/activity/ [get]
func (h *Handler) activity(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, ok := h.itemID(c)
	if !ok {
		return
	}
	req := &billingqueueports.ListBillingQueueActivityRequest{
		ItemID:     itemID,
		TenantInfo: pagination.FromAuth(authCtx),
		Limit:      helpers.QueryInt(c, "limit", 30),
		BeforeAt:   helpers.QueryInt64(c, "beforeAt"),
	}
	if raw := c.Query("beforeId"); raw != "" {
		id, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		req.BeforeID = id
	}

	page, err := h.review.ListActivity(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, page)
}

type resolveIssueRequest struct {
	OptionKey string `json:"optionKey" binding:"required"`
}

// @Summary Settle one of an item's checks
// @ID resolveBillingQueueIssue
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /billing-queue/{itemID}/issues/{issueID}/resolve/ [post]
func (h *Handler) resolveIssue(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, ok := h.itemID(c)
	if !ok {
		return
	}
	issueID, err := pulid.MustParse(c.Param("issueID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body resolveIssueRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	item, err := h.review.ResolveIssue(
		c.Request.Context(),
		&billingqueueports.ResolveBillingQueueIssueRequest{
			ItemID:     itemID,
			IssueID:    issueID,
			OptionKey:  body.OptionKey,
			TenantInfo: pagination.FromAuth(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondItem(c, http.StatusOK, item)
}

// @Summary Take back how a check was settled
// @ID undoBillingQueueIssue
// @Tags Billing Queue
// @Produce json
// @Security BearerAuth
// @Router /billing-queue/{itemID}/issues/{issueID}/undo/ [post]
func (h *Handler) undoIssue(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, ok := h.itemID(c)
	if !ok {
		return
	}
	issueID, err := pulid.MustParse(c.Param("issueID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	item, err := h.review.UndoIssue(
		c.Request.Context(),
		&billingqueueports.UndoBillingQueueIssueRequest{
			ItemID:     itemID,
			IssueID:    issueID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondItem(c, http.StatusOK, item)
}

// @Summary Release an item's hold
// @ID releaseBillingQueueItem
// @Tags Billing Queue
// @Produce json
// @Security BearerAuth
// @Router /billing-queue/{itemID}/release/ [post]
func (h *Handler) release(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, ok := h.itemID(c)
	if !ok {
		return
	}

	item, err := h.review.Release(c.Request.Context(), &billingqueueports.BillingQueueItemRequest{
		ItemID:     itemID,
		TenantInfo: pagination.FromAuth(authCtx),
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondItem(c, http.StatusOK, item)
}

// @Summary Post an approved item's invoice
// @ID postBillingQueueItem
// @Tags Billing Queue
// @Produce json
// @Security BearerAuth
// @Router /billing-queue/{itemID}/post/ [post]
func (h *Handler) post(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, ok := h.itemID(c)
	if !ok {
		return
	}

	result, err := h.review.Post(c.Request.Context(), &billingqueueports.BillingQueueItemRequest{
		ItemID:     itemID,
		TenantInfo: pagination.FromAuth(authCtx),
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	formulavisibility.Respond(
		c,
		h.formulas,
		http.StatusOK,
		result,
		(*billingqueueports.PostBillingQueueItemResult).WithoutFormulaContent,
	)
}

type bulkRequest struct {
	ItemIDs        []pulid.ID                   `json:"itemIds"        binding:"required"`
	IdempotencyKey string                       `json:"idempotencyKey" binding:"required"`
	Action         billingapprovalrun.RunAction `json:"action"`
	Params         billingapprovalrun.RunParams `json:"params"`
}

// @Summary Apply one action to several items as one job
// @Description Approve, post, hold, assign, cancel, ask drivers for paperwork, or settle one check, across up to 500 items. Every action but posting waits out a short undo window first. Posting also needs invoice post permission, assigning needs billing queue assign.
// @ID startBillingQueueBulk
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Success 202 {object} billingapprovalrun.ApprovalRun
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/bulk/ [post]
func (h *Handler) startBulk(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	var body bulkRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if len(body.ItemIDs) > billingapprovalrun.MaxApprovalRunItems {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"itemIds", errortypes.ErrInvalid, "Act on at most 500 items at a time",
		))
		return
	}
	if err := h.requireBulkPermission(c, body.Action); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	run, err := h.approval.Start(
		c.Request.Context(),
		&billingqueueports.StartBillingQueueApprovalRequest{
			TenantInfo:     pagination.FromAuthAsUser(authCtx),
			ItemIDs:        body.ItemIDs,
			IdempotencyKey: body.IdempotencyKey,
			Action:         body.Action,
			Params:         body.Params,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusAccepted, run)
}

func (h *Handler) requireBulkPermission(c *gin.Context, action billingapprovalrun.RunAction) error {
	resource, op := "", permission.Operation("")
	switch action {
	case billingapprovalrun.RunActionPost:
		resource, op = permission.ResourceInvoice.String(), permission.OpPost
	case billingapprovalrun.RunActionAssign:
		resource, op = permission.ResourceBillingQueue.String(), permission.OpAssign
	case billingapprovalrun.RunActionApprove,
		billingapprovalrun.RunActionHold,
		billingapprovalrun.RunActionCancel,
		billingapprovalrun.RunActionRequestPaperwork,
		billingapprovalrun.RunActionResolveIssue:
		return nil
	default:
		return nil
	}

	allowed, err := h.pm.Allowed(c, resource, op)
	if err != nil {
		return err
	}
	if !allowed {
		return errortypes.NewAuthorizationError("You don't have permission for this bulk action")
	}

	return nil
}

func (h *Handler) runRequest(
	c *gin.Context,
) (*billingqueueports.BillingQueueApprovalRunRequest, bool) {
	authCtx := authctx.GetAuthContext(c)
	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return nil, false
	}

	// Undo is the approver's alone, so the request says who is asking.
	return &billingqueueports.BillingQueueApprovalRunRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		RunID:      runID,
	}, true
}

// @Summary Get a bulk action and its per-item results
// @ID getBillingQueueBulk
// @Tags Billing Queue
// @Produce json
// @Success 200 {object} billingapprovalrun.ApprovalRun
// @Security BearerAuth
// @Router /billing-queue/bulk/{runID}/ [get]
func (h *Handler) getBulk(c *gin.Context) {
	req, ok := h.runRequest(c)
	if !ok {
		return
	}
	run, err := h.approval.Get(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, run)
}

// @Summary Undo a bulk action inside its window
// @ID cancelBillingQueueBulk
// @Tags Billing Queue
// @Produce json
// @Success 200 {object} billingapprovalrun.ApprovalRun
// @Security BearerAuth
// @Router /billing-queue/bulk/{runID}/cancel/ [post]
func (h *Handler) cancelBulk(c *gin.Context) {
	req, ok := h.runRequest(c)
	if !ok {
		return
	}
	run, err := h.approval.Undo(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, run)
}
