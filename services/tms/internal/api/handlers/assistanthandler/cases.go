package assistanthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// A case is the person's own conversation about a record, so binding,
// snoozing and waiting on a reply need no more than using the assistant;
// the service reads the thread under the caller's user id and checks they
// may read the record before anything about it is shown or bound.
func (h *Handler) registerCaseRoutes(api *gin.RouterGroup, resource string) {
	read := h.pm.RequirePermission(resource, permission.OpRead)

	api.GET("/threads/:threadID/case/", read, h.getCase)
	api.PUT("/threads/:threadID/case/", read, h.bindCase)
	api.DELETE("/threads/:threadID/case/", read, h.unbindCase)
	api.POST("/threads/:threadID/case/snooze/", read, h.snoozeCase)
	api.POST("/threads/:threadID/case/wake/", read, h.wakeCase)
	api.POST("/threads/:threadID/case/await-reply/", read, h.awaitCaseReply)
	api.POST("/threads/:threadID/case/ticks/", read, h.tickCaseItem)
}

type tickBody struct {
	ItemKey deskcase.ItemKey `json:"itemKey"`
	Ticked  bool             `json:"ticked"`
}

func (h *Handler) tickCaseItem(c *gin.Context) {
	req, err := caseRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body tickBody
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	view, err := h.cases.Tick(c.Request.Context(), &serviceports.TickCaseItemRequest{
		CaseThreadRequest: *req,
		ItemKey:           body.ItemKey,
		Ticked:            body.Ticked,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, view)
}

// caseBinding is what a write to a case answers with: what the
// conversation is about now and where its case stands.
type caseBinding struct {
	ThreadID     pulid.ID              `json:"threadId"`
	SubjectType  agent.SubjectType     `json:"subjectType"`
	SubjectID    pulid.ID              `json:"subjectId"`
	SnoozedUntil *int64                `json:"snoozedUntil,omitempty"`
	SnoozeAnchor deskcase.SnoozeAnchor `json:"snoozeAnchor,omitempty"`
	Case         *deskcase.Summary     `json:"case,omitempty"`
}

func bindingOf(thread *conversation.Thread) caseBinding {
	return caseBinding{
		ThreadID:     thread.ID,
		SubjectType:  thread.SubjectType,
		SubjectID:    thread.SubjectID,
		SnoozedUntil: thread.SnoozedUntil,
		SnoozeAnchor: thread.SnoozeAnchor,
		Case:         thread.Case,
	}
}

type bindCaseBody struct {
	SubjectType agent.SubjectType `json:"subjectType"`
	SubjectID   pulid.ID          `json:"subjectId"`
}

type snoozeCaseBody struct {
	Anchor deskcase.SnoozeAnchor `json:"anchor"`
	Until  int64                 `json:"until"`
}

type awaitReplyBody struct {
	Party            deskcase.WaitingOn `json:"party"`
	PartyID          pulid.ID           `json:"partyId"`
	GiveUpAfterHours int                `json:"giveUpAfterHours"`
}

func caseRequest(c *gin.Context) (*serviceports.CaseThreadRequest, error) {
	req, err := threadRequest(c)
	if err != nil {
		return nil, err
	}
	actor := requestActorFromAuthContext(authctx.GetAuthContext(c))

	return &serviceports.CaseThreadRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		Actor:      &actor,
	}, nil
}

func (h *Handler) getCase(c *gin.Context) {
	req, err := caseRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	view, err := h.cases.Get(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, view)
}

func (h *Handler) bindCase(c *gin.Context) {
	req, err := caseRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body bindCaseBody
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	thread, err := h.cases.Bind(c.Request.Context(), &serviceports.BindCaseRequest{
		CaseThreadRequest: *req,
		SubjectType:       body.SubjectType,
		SubjectID:         body.SubjectID,
	})
	h.respondCase(c, thread, err)
}

func (h *Handler) unbindCase(c *gin.Context) {
	req, err := caseRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	thread, err := h.cases.Unbind(c.Request.Context(), req)
	h.respondCase(c, thread, err)
}

func (h *Handler) snoozeCase(c *gin.Context) {
	req, err := caseRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body snoozeCaseBody
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	thread, err := h.cases.Snooze(c.Request.Context(), &serviceports.SnoozeCaseRequest{
		CaseThreadRequest: *req,
		Anchor:            body.Anchor,
		Until:             body.Until,
	})
	h.respondCase(c, thread, err)
}

func (h *Handler) wakeCase(c *gin.Context) {
	req, err := caseRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	thread, err := h.cases.Wake(c.Request.Context(), req)
	h.respondCase(c, thread, err)
}

func (h *Handler) awaitCaseReply(c *gin.Context) {
	req, err := caseRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body awaitReplyBody
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	wait, err := h.cases.AwaitReply(c.Request.Context(), &serviceports.AwaitCaseReplyRequest{
		CaseThreadRequest: *req,
		Party:             body.Party,
		PartyID:           body.PartyID,
		GiveUpAfterHours:  body.GiveUpAfterHours,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, wait)
}

func (h *Handler) respondCase(c *gin.Context, thread *conversation.Thread, err error) {
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, bindingOf(thread))
}
