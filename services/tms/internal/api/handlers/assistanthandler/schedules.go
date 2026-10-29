package assistanthandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/agent/app/conversationscheduleservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
	) (*pagination.ListResult[*conversationscheduleports.ListedConversationSchedule], error)
	SetEnabled(
		ctx context.Context,
		req conversationscheduleservice.ScheduleRequest,
		enabled bool,
	) (*conversationschedule.Schedule, error)
	Edit(
		ctx context.Context,
		req conversationscheduleservice.ScheduleRequest,
		edit conversationschedule.Edit,
	) (*conversationschedule.Schedule, error)
	Delete(ctx context.Context, req conversationscheduleservice.ScheduleRequest) error
	RunNow(
		ctx context.Context,
		req conversationscheduleservice.ScheduleRequest,
	) (*conversationturn.AssistantTurn, error)
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

// listSchedules is every schedule the person keeps, across their
// conversations, the ones that run soonest first and the paused ones after.
func (h *Handler) listSchedules(c *gin.Context) {
	h.respondWithSchedules(c, pulid.Nil, conversationscheduleports.ConversationScheduleOrderNextRun)
}

func (h *Handler) listThreadSchedules(c *gin.Context) {
	threadID, err := pulid.Parse(c.Param("threadID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondWithSchedules(c, threadID, conversationscheduleports.ConversationScheduleOrderNewest)
}

func (h *Handler) respondWithSchedules(
	c *gin.Context,
	threadID pulid.ID,
	order conversationscheduleports.ConversationScheduleOrder,
) {
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
		Order:    order,
		Limit:    query.Limit,
		Offset:   query.Offset,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
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

	result, err := h.schedules.Create(
		c.Request.Context(),
		conversationscheduleservice.CreateRequest{
			ThreadID: threadID,
			Text:     body.Content,
			Actor:    requestActorFromAuthContext(authctx.GetAuthContext(c)),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, result)
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

// updateScheduleRequest pauses or resumes a schedule (enabled), or edits it
// (prompt, repeat and time together), or both.
type updateScheduleRequest struct {
	Enabled *bool   `json:"enabled"`
	Prompt  *string `json:"prompt"`
	Repeat  *string `json:"repeat"`
	Time    *string `json:"time"`
}

// edit is the change the body asks for, nil when it asks for none. An edit
// names all three of what to ask, how often and when.
func (r updateScheduleRequest) edit() (*conversationschedule.Edit, error) {
	if r.Prompt == nil && r.Repeat == nil && r.Time == nil {
		return nil, nil
	}

	multiErr := errortypes.NewMultiError()
	if r.Prompt == nil {
		multiErr.Add("prompt", errortypes.ErrRequired, "Say what to ask on each run")
	}
	if r.Repeat == nil {
		multiErr.Add("repeat", errortypes.ErrRequired, "Choose how often it runs")
	}
	if r.Time == nil {
		multiErr.Add("time", errortypes.ErrRequired, "Enter a time of day, such as 08:00")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &conversationschedule.Edit{Prompt: *r.Prompt, Repeat: *r.Repeat, Time: *r.Time}, nil
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
	edit, err := body.edit()
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if edit == nil && body.Enabled == nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"enabled", errortypes.ErrRequired, "Say whether the schedule should run",
		))
		return
	}

	var schedule *conversationschedule.Schedule
	if edit != nil {
		if schedule, err = h.schedules.Edit(c.Request.Context(), req, *edit); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	if body.Enabled != nil {
		if schedule, err = h.schedules.SetEnabled(
			c.Request.Context(),
			req,
			*body.Enabled,
		); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}

	helpers.JSON(c, http.StatusOK, schedule)
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

	helpers.JSON(c, http.StatusAccepted, turnStarted(turn))
}
