package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageconflictservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type FakeTriage struct {
	Message *inboundmessage.InboundMessage
	PlanErr error

	QueuePlan       *inboundmessageservice.QueuePlan
	SenderPlan      *triageworkservice.SenderPlan
	DraftPlan       *inboundreplyservice.DraftPlan
	LoadDraft       *triageworkservice.LoadDraftPlan
	LoadDiscard     *triageworkservice.LoadDiscardPlan
	CreateLoadPlan  *triageworkservice.CreateLoadPlan
	FilePlan        *triageworkservice.FilePlan
	UnfilePlan      *triageworkservice.UnfilePlan
	ExceptionPlan   *triageworkservice.ExceptionPlan
	StopsPlan       *triageworkservice.StopsPlan
	ConflictPlan    *triageconflictservice.ConflictPlan
	StopTimesResult *inboundmessageports.StopTimes

	Planned []any
	Written []any
}

func (f *FakeTriage) plan(req any) error {
	f.Planned = append(f.Planned, req)

	return f.PlanErr
}

func (f *FakeTriage) write(req any) {
	f.Written = append(f.Written, req)
}

func (f *FakeTriage) GetByID(
	_ context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	if f.Message == nil || f.Message.ID != req.ID {
		return nil, errortypes.NewNotFoundError("InboundMessage not found")
	}
	copied := *f.Message

	return &copied, nil
}

func (f *FakeTriage) bulk() *inboundmessageports.BulkResult {
	return &inboundmessageports.BulkResult{
		Messages: []*inboundmessage.InboundMessage{f.Message},
		Affected: 1,
	}
}

func (f *FakeTriage) Assign(
	_ context.Context,
	req *inboundmessageservice.AssignRequest,
) (*inboundmessageports.BulkResult, error) {
	f.write(req)

	return f.bulk(), nil
}

func (f *FakeTriage) PlanAssign(
	_ context.Context,
	req *inboundmessageservice.AssignRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.QueuePlan, f.plan(req)
}

func (f *FakeTriage) Reopen(
	_ context.Context,
	req *inboundmessageservice.ReopenRequest,
) (*inboundmessageports.BulkResult, error) {
	f.write(req)

	return f.bulk(), nil
}

func (f *FakeTriage) PlanReopen(
	_ context.Context,
	req *inboundmessageservice.ReopenRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.QueuePlan, f.plan(req)
}

func (f *FakeTriage) Snooze(
	_ context.Context,
	req *inboundmessageservice.SnoozeRequest,
) (*inboundmessageports.BulkResult, error) {
	f.write(req)

	return f.bulk(), nil
}

func (f *FakeTriage) PlanSnooze(
	_ context.Context,
	req *inboundmessageservice.SnoozeRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.QueuePlan, f.plan(req)
}

func (f *FakeTriage) Wake(
	_ context.Context,
	req *inboundmessageservice.MessageActionRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanWake(
	_ context.Context,
	req *inboundmessageservice.MessageActionRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.QueuePlan, f.plan(req)
}

func (f *FakeTriage) StopWaiting(
	_ context.Context,
	req *inboundmessageservice.MessageActionRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(StopWaitingCall{req})

	return f.Message, nil
}

func (f *FakeTriage) PlanStopWaiting(
	_ context.Context,
	req *inboundmessageservice.MessageActionRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.QueuePlan, f.plan(StopWaitingCall{req})
}

type StopWaitingCall struct {
	Req *inboundmessageservice.MessageActionRequest
}

func (f *FakeTriage) WaitForReply(
	_ context.Context,
	req *inboundmessageservice.WaitForReplyRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanWaitForReply(
	_ context.Context,
	req *inboundmessageservice.WaitForReplyRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.QueuePlan, f.plan(req)
}

func (f *FakeTriage) Unlink(
	_ context.Context,
	req *inboundmessageservice.UnlinkRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanUnlink(
	_ context.Context,
	req *inboundmessageservice.UnlinkRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.QueuePlan, f.plan(req)
}

func (f *FakeTriage) HandOff(
	_ context.Context,
	req *inboundmessageservice.HandOffRequest,
) (*inboundmessageports.HandOffResult, error) {
	f.write(req)

	return &inboundmessageports.HandOffResult{ToUserID: req.ToUserID, Affected: 3}, nil
}

func (f *FakeTriage) PlanHandOff(
	_ context.Context,
	req *inboundmessageservice.HandOffRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.QueuePlan, f.plan(req)
}

func (f *FakeTriage) BlockSender(
	_ context.Context,
	req *triageworkservice.BlockSenderRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanBlockSender(
	_ context.Context,
	req *triageworkservice.BlockSenderRequest,
) (*triageworkservice.SenderPlan, error) {
	return f.SenderPlan, f.plan(req)
}

func (f *FakeTriage) UnblockSender(
	_ context.Context,
	req *triageworkservice.UnblockSenderRequest,
) (*inboundmessage.Mailbox, error) {
	f.write(req)

	return f.SenderPlan.Mailbox, nil
}

func (f *FakeTriage) PlanUnblockSender(
	_ context.Context,
	req *triageworkservice.UnblockSenderRequest,
) (*triageworkservice.SenderPlan, error) {
	return f.SenderPlan, f.plan(req)
}

func (f *FakeTriage) DraftReply(
	_ context.Context,
	req *inboundreplyservice.DraftReplyRequest,
) (*triage.Reply, error) {
	f.write(req)

	return &triage.Reply{MessageID: req.MessageID, Subject: "Re: drafted"}, nil
}

func (f *FakeTriage) PlanDraftReply(
	_ context.Context,
	req *inboundreplyservice.DraftReplyRequest,
) (*inboundreplyservice.DraftPlan, error) {
	return f.DraftPlan, f.plan(req)
}

func (f *FakeTriage) SaveDraft(
	_ context.Context,
	req *inboundreplyservice.SaveDraftRequest,
) (*triage.Reply, error) {
	f.write(req)

	return &triage.Reply{MessageID: req.MessageID, Subject: req.Subject, Body: req.Body}, nil
}

func (f *FakeTriage) PlanSaveDraft(
	_ context.Context,
	req *inboundreplyservice.SaveDraftRequest,
) (*inboundreplyservice.DraftPlan, error) {
	return f.DraftPlan, f.plan(req)
}

type DiscardDraftCall struct {
	Tenant    pagination.TenantInfo
	MessageID pulid.ID
}

func (f *FakeTriage) DiscardDraft(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundmessage.InboundMessage, error) {
	f.write(DiscardDraftCall{Tenant: tenantInfo, MessageID: messageID})

	return f.Message, nil
}

func (f *FakeTriage) PlanDiscardDraft(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundreplyservice.DraftPlan, error) {
	return f.DraftPlan, f.plan(DiscardDraftCall{Tenant: tenantInfo, MessageID: messageID})
}

func (f *FakeTriage) DraftLoad(
	_ context.Context,
	req *triageworkservice.LoadRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanDraftLoad(
	_ context.Context,
	req *triageworkservice.LoadRequest,
) (*triageworkservice.LoadDraftPlan, error) {
	return f.LoadDraft, f.plan(req)
}

func (f *FakeTriage) DiscardLoad(
	_ context.Context,
	req *triageworkservice.LoadRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(DiscardLoadCall{req})

	return f.Message, nil
}

type DiscardLoadCall struct {
	Req *triageworkservice.LoadRequest
}

func (f *FakeTriage) PlanDiscardLoad(
	_ context.Context,
	req *triageworkservice.LoadRequest,
) (*triageworkservice.LoadDiscardPlan, error) {
	return f.LoadDiscard, f.plan(DiscardLoadCall{req})
}

func (f *FakeTriage) CreateLoad(
	_ context.Context,
	req *triageworkservice.CreateLoadRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)
	created := *f.Message
	created.ActionState.CreatedShipmentID = pulid.MustNew("shp_")

	return &created, nil
}

func (f *FakeTriage) PlanCreateLoad(
	_ context.Context,
	req *triageworkservice.CreateLoadRequest,
) (*triageworkservice.CreateLoadPlan, error) {
	return f.CreateLoadPlan, f.plan(req)
}

func (f *FakeTriage) File(
	_ context.Context,
	req *triageworkservice.FileRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanFile(
	_ context.Context,
	req *triageworkservice.FileRequest,
) (*triageworkservice.FilePlan, error) {
	return f.FilePlan, f.plan(req)
}

func (f *FakeTriage) Unfile(
	_ context.Context,
	req *triageworkservice.UnfileRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanUnfile(
	_ context.Context,
	req *triageworkservice.UnfileRequest,
) (*triageworkservice.UnfilePlan, error) {
	return f.UnfilePlan, f.plan(req)
}

func (f *FakeTriage) LogException(
	_ context.Context,
	req *triageworkservice.LogExceptionRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanLogException(
	_ context.Context,
	req *triageworkservice.LogExceptionRequest,
) (*triageworkservice.ExceptionPlan, error) {
	return f.ExceptionPlan, f.plan(req)
}

func (f *FakeTriage) DecideStops(
	_ context.Context,
	req *triageworkservice.DecideStopsRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f *FakeTriage) PlanDecideStops(
	_ context.Context,
	req *triageworkservice.DecideStopsRequest,
) (*triageworkservice.StopsPlan, error) {
	return f.StopsPlan, f.plan(req)
}

func (f *FakeTriage) StopTimes(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*inboundmessageports.StopTimes, error) {
	return f.StopTimesResult, nil
}

type FakeConflictDesk struct {
	*FakeTriage
}

func (f FakeConflictDesk) Resolve(
	_ context.Context,
	req *triageconflictservice.ResolveRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f FakeConflictDesk) PlanResolve(
	_ context.Context,
	req *triageconflictservice.ResolveRequest,
) (*triageconflictservice.ConflictPlan, error) {
	return f.ConflictPlan, f.plan(req)
}

func (f FakeConflictDesk) Reopen(
	_ context.Context,
	req *triageconflictservice.ReopenRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.Message, nil
}

func (f FakeConflictDesk) PlanReopen(
	_ context.Context,
	req *triageconflictservice.ReopenRequest,
) (*triageconflictservice.ConflictPlan, error) {
	return f.ConflictPlan, f.plan(req)
}
