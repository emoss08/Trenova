package billingqueuehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerAssistRoutes(api *gin.RouterGroup) {
	read := h.pm.RequirePermission(permission.ResourceBillingQueue.String(), permission.OpRead)

	api.GET("/assist/", read, h.assistAvailability)
	api.GET("/brief/", read, h.brief)
	api.GET("/nova-generation/", read, h.novaGeneration)
	api.POST("/search-intent/", read, h.searchIntent)
}

// @Summary What Nova can do on the billing queue for this organization
// @Description Whether Nova reviews items before approval, writes the queue's brief, and reads a search typed as a sentence. Each depends on the billing controls and on an AI provider serving the task.
// @ID getBillingQueueAssist
// @Tags Billing Queue
// @Produce json
// @Success 200 {object} billingqueueports.AssistAvailability
// @Security BearerAuth
// @Router /billing-queue/assist/ [get]
func (h *Handler) assistAvailability(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	availability, err := h.assist.Availability(c.Request.Context(), pagination.FromAuth(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, availability)
}

// @Summary The brief Nova wrote for the billing queue
// @ID getBillingQueueBrief
// @Tags Billing Queue
// @Produce json
// @Success 200 {object} billingqueueports.QueueBriefView
// @Security BearerAuth
// @Router /billing-queue/brief/ [get]
func (h *Handler) brief(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	brief, err := h.assist.GetBrief(c.Request.Context(), pagination.FromAuth(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, brief)
}

// @Summary What Nova is writing for the billing queue in bulk
// @Description The reviews and brief Nova is generating for the whole queue, or null when nothing is. Reading it starts a generation when the queue has items Nova has not reviewed or no brief; one runs per organization at a time, so every reader sees the same one.
// @ID getBillingQueueNovaGeneration
// @Tags Billing Queue
// @Produce json
// @Success 200 {object} billingqueueports.NovaGeneration
// @Security BearerAuth
// @Router /billing-queue/nova-generation/ [get]
func (h *Handler) novaGeneration(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	generation, err := h.assist.NovaGeneration(c.Request.Context(), pagination.FromAuth(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, generation)
}

type searchIntentRequest struct {
	Text string `json:"text" binding:"required"`
	// Today is the person's date, YYYY-MM-DD, which "last week" is counted from.
	Today string `json:"today"`
}

// @Summary Turn a sentence into billing queue filters
// @Description Reads a search such as "Acme over $5k delivered last week" and returns the lanes, payer, shipment customer, amounts, age and delivery, pickup and queued dates it names. Nothing is changed.
// @ID composeBillingQueueSearch
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Success 200 {object} billingqueueports.SearchIntent
// @Failure 400 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/search-intent/ [post]
func (h *Handler) searchIntent(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	var body searchIntentRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	intent, err := h.assist.ComposeSearch(
		c.Request.Context(),
		&billingqueueports.SearchIntentRequest{
			TenantInfo: pagination.FromAuth(authCtx),
			UserID:     authCtx.UserID,
			Text:       body.Text,
			Today:      body.Today,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, intent)
}
