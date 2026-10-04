package billingqueuehandler

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
	api.POST("/bulk-approve/", update, h.startBulkApprove)
	api.GET("/bulk-approve/:runID/", read, h.getBulkApprove)
	api.POST("/bulk-approve/:runID/cancel/", update, h.cancelBulkApprove)
	api.GET("/:itemID/neighbors/", read, h.neighbors)
	api.GET("/:itemID/activity/", read, h.activity)
	api.POST("/:itemID/issues/:issueID/resolve/", update, h.resolveIssue)
	api.POST("/:itemID/issues/:issueID/undo/", update, h.undoIssue)
	api.POST("/:itemID/release/", update, h.release)
	api.POST("/:itemID/post/", update, h.post)
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

	out, err := h.review.Summaries(c.Request.Context(), &repositories.ListBillingQueueSummariesRequest{
		TenantInfo: pagination.FromAuth(authCtx),
		ItemIDs:    ids,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": out})
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

	out, err := h.review.Neighbors(c.Request.Context(), &repositories.GetBillingQueueNeighborsRequest{
		ItemID:        itemID,
		Filter:        pagination.NewQueryOptions(c, authCtx),
		IncludePosted: helpers.QueryBool(c, "includePosted"),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
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
	req := &services.ListBillingQueueActivityRequest{
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

	c.JSON(http.StatusOK, page)
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

	item, err := h.review.ResolveIssue(c.Request.Context(), &services.ResolveBillingQueueIssueRequest{
		ItemID:     itemID,
		IssueID:    issueID,
		OptionKey:  body.OptionKey,
		TenantInfo: pagination.FromAuth(authCtx),
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
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

	item, err := h.review.UndoIssue(c.Request.Context(), &services.UndoBillingQueueIssueRequest{
		ItemID:     itemID,
		IssueID:    issueID,
		TenantInfo: pagination.FromAuth(authCtx),
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
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

	item, err := h.review.Release(c.Request.Context(), &services.BillingQueueItemRequest{
		ItemID:     itemID,
		TenantInfo: pagination.FromAuth(authCtx),
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
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

	result, err := h.review.Post(c.Request.Context(), &services.BillingQueueItemRequest{
		ItemID:     itemID,
		TenantInfo: pagination.FromAuth(authCtx),
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type bulkApproveRequest struct {
	ItemIDs        []pulid.ID `json:"itemIds"        binding:"required"`
	IdempotencyKey string     `json:"idempotencyKey" binding:"required"`
	// AssignApprover makes the person approving the biller of the picked
	// items that have none.
	AssignApprover bool `json:"assignApprover"`
}

// @Summary Approve several items as one job, after an undo window
// @ID startBillingQueueBulkApprove
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /billing-queue/bulk-approve/ [post]
func (h *Handler) startBulkApprove(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	var body bulkApproveRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if len(body.ItemIDs) > billingqueue.MaxApprovalRunItems {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"itemIds", errortypes.ErrInvalid, "Approve at most 500 items at a time",
		))
		return
	}

	// The run is the person's: it is approved in their name, and an item with
	// no biller can only be given to them.
	run, err := h.approval.Start(c.Request.Context(), &services.StartBillingQueueApprovalRequest{
		TenantInfo:     pagination.FromAuthAsUser(authCtx),
		ItemIDs:        body.ItemIDs,
		IdempotencyKey: body.IdempotencyKey,
		AssignApprover: body.AssignApprover,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, run)
}

func (h *Handler) runRequest(c *gin.Context) (*services.BillingQueueApprovalRunRequest, bool) {
	authCtx := authctx.GetAuthContext(c)
	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return nil, false
	}

	// Undo is the approver's alone, so the request says who is asking.
	return &services.BillingQueueApprovalRunRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		RunID:      runID,
	}, true
}

// @Summary Get a bulk approval and its per-item results
// @ID getBillingQueueBulkApprove
// @Tags Billing Queue
// @Produce json
// @Security BearerAuth
// @Router /billing-queue/bulk-approve/{runID}/ [get]
func (h *Handler) getBulkApprove(c *gin.Context) {
	req, ok := h.runRequest(c)
	if !ok {
		return
	}
	run, err := h.approval.Get(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}

// @Summary Undo a bulk approval inside its window
// @ID cancelBillingQueueBulkApprove
// @Tags Billing Queue
// @Produce json
// @Security BearerAuth
// @Router /billing-queue/bulk-approve/{runID}/cancel/ [post]
func (h *Handler) cancelBulkApprove(c *gin.Context) {
	req, ok := h.runRequest(c)
	if !ok {
		return
	}
	run, err := h.approval.Undo(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}
