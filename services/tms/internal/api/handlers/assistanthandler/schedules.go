package assistanthandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/conversationscheduleservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// scheduleService is what the schedule routes ask of the schedule service.
type scheduleService interface {
	Create(
		ctx context.Context,
		req conversationscheduleservice.CreateRequest,
	) (*conversationscheduleservice.CreateResult, error)
	List(
		ctx context.Context,
		req conversationscheduleservice.ListRequest,
	) (*pagination.ListResult[*conversationschedule.Schedule], error)
	SetEnabled(
		ctx context.Context,
		req conversationscheduleservice.ScheduleRequest,
		enabled bool,
	) (*conversationschedule.Schedule, error)
	Delete(ctx context.Context, req conversationscheduleservice.ScheduleRequest) error
	RunNow(
		ctx context.Context,
		req conversationscheduleservice.ScheduleRequest,
	) (*conversation.AssistantTurn, error)
}

// registerScheduleRoutes adds the routes for requests scheduled in a
// conversation.
//
// Scheduling a request is asking the agent something, later and again, so it
// is gated like sending a message, and so is running one now. Pausing,
// resuming and deleting one is arranging the person's own conversation, like
// naming it, and every schedule is read under the caller's own user id: one
// that is not theirs is not found.
func (h *Handler) registerScheduleRoutes(api *gin.RouterGroup, resource string) {
	api.GET(
		"/schedules/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listSchedules,
	)
	api.GET(
		"/threads/:threadID/schedules/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listThreadSchedules,
	)
	api.POST(
		"/threads/:threadID/schedules/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.createSchedule,
	)
	api.PATCH(
		"/schedules/:scheduleID/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.updateSchedule,
	)
	api.DELETE(
		"/schedules/:scheduleID/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.deleteSchedule,
	)
	api.POST(
		"/schedules/:scheduleID/run/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.runSchedule,
	)
}

func (h *Handler) schedulesReady(c *gin.Context) bool {
	if h.schedules != nil {
		return true
	}
	h.eh.HandleError(c, errortypes.NewBusinessError("Scheduled requests are not available"))

	return false
}

type listSchedulesQuery struct {
	Limit  int `form:"limit"`
	Offset int `form:"offset"`
}

func (h *Handler) listSchedules(c *gin.Context) {
	h.respondWithSchedules(c, pulid.Nil)
}

func (h *Handler) listThreadSchedules(c *gin.Context) {
	threadID, err := pulid.Parse(c.Param("threadID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondWithSchedules(c, threadID)
}

func (h *Handler) respondWithSchedules(c *gin.Context, threadID pulid.ID) {
	if !h.schedulesReady(c) {
		return
	}

	var query listSchedulesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.schedules.List(c.Request.Context(), conversationscheduleservice.ListRequest{
		ThreadID: threadID,
		Actor:    requestActorFromAuthContext(authctx.GetAuthContext(c)),
		Limit:    query.Limit,
		Offset:   query.Offset,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type createScheduleRequest struct {
	// Content is the message as the person sent it: when, then what to ask.
	Content string `json:"content"`
}

func (h *Handler) createSchedule(c *gin.Context) {
	if !h.schedulesReady(c) {
		return
	}

	threadID, err := pulid.Parse(c.Param("threadID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body createScheduleRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.schedules.Create(c.Request.Context(), conversationscheduleservice.CreateRequest{
		ThreadID: threadID,
		Text:     body.Content,
		Actor:    requestActorFromAuthContext(authctx.GetAuthContext(c)),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// scheduleRequest names the schedule in the path, as the caller's own.
func scheduleRequest(c *gin.Context) (conversationscheduleservice.ScheduleRequest, error) {
	scheduleID, err := pulid.Parse(c.Param("scheduleID"))
	if err != nil {
		return conversationscheduleservice.ScheduleRequest{}, err
	}

	return conversationscheduleservice.ScheduleRequest{
		ScheduleID: scheduleID,
		Actor:      requestActorFromAuthContext(authctx.GetAuthContext(c)),
	}, nil
}

type updateScheduleRequest struct {
	Enabled *bool `json:"enabled"`
}

func (h *Handler) updateSchedule(c *gin.Context) {
	if !h.schedulesReady(c) {
		return
	}

	req, err := scheduleRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body updateScheduleRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if body.Enabled == nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"enabled", errortypes.ErrRequired, "Say whether the schedule should run",
		))
		return
	}

	schedule, err := h.schedules.SetEnabled(c.Request.Context(), req, *body.Enabled)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, schedule)
}

func (h *Handler) deleteSchedule(c *gin.Context) {
	if !h.schedulesReady(c) {
		return
	}

	req, err := scheduleRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.schedules.Delete(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// runSchedule asks a schedule's request now and returns the turn to watch,
// the same answer as asking a question.
func (h *Handler) runSchedule(c *gin.Context) {
	if !h.schedulesReady(c) {
		return
	}

	req, err := scheduleRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	turn, err := h.schedules.RunNow(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, turnStarted(turn))
}
