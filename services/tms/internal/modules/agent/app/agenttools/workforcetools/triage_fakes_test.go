package workforcetools

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

type fakeTriage struct {
	message *inboundmessage.InboundMessage
	planErr error

	queuePlan     *inboundmessageservice.QueuePlan
	senderPlan    *triageworkservice.SenderPlan
	draftPlan     *inboundreplyservice.DraftPlan
	loadDraft     *triageworkservice.LoadDraftPlan
	loadDiscard   *triageworkservice.LoadDiscardPlan
	createLoad    *triageworkservice.CreateLoadPlan
	filePlan      *triageworkservice.FilePlan
	unfilePlan    *triageworkservice.UnfilePlan
	exceptionPlan *triageworkservice.ExceptionPlan
	stopsPlan     *triageworkservice.StopsPlan
	conflictPlan  *triageconflictservice.ConflictPlan
	stopTimes     *inboundmessageports.StopTimes

	planned []any
	written []any
}

func (f *fakeTriage) plan(req any) error {
	f.planned = append(f.planned, req)

	return f.planErr
}

func (f *fakeTriage) write(req any) {
	f.written = append(f.written, req)
}

func (f *fakeTriage) GetByID(
	_ context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	if f.message == nil || f.message.ID != req.ID {
		return nil, errortypes.NewNotFoundError("InboundMessage not found")
	}
	copied := *f.message

	return &copied, nil
}

func (f *fakeTriage) bulk() *inboundmessageports.BulkResult {
	return &inboundmessageports.BulkResult{
		Messages: []*inboundmessage.InboundMessage{f.message},
		Affected: 1,
	}
}

func (f *fakeTriage) Assign(
	_ context.Context,
	req *inboundmessageservice.AssignRequest,
) (*inboundmessageports.BulkResult, error) {
	f.write(req)

	return f.bulk(), nil
}

func (f *fakeTriage) PlanAssign(
	_ context.Context,
	req *inboundmessageservice.AssignRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.queuePlan, f.plan(req)
}

func (f *fakeTriage) Reopen(
	_ context.Context,
	req *inboundmessageservice.ReopenRequest,
) (*inboundmessageports.BulkResult, error) {
	f.write(req)

	return f.bulk(), nil
}

func (f *fakeTriage) PlanReopen(
	_ context.Context,
	req *inboundmessageservice.ReopenRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.queuePlan, f.plan(req)
}

func (f *fakeTriage) Snooze(
	_ context.Context,
	req *inboundmessageservice.SnoozeRequest,
) (*inboundmessageports.BulkResult, error) {
	f.write(req)

	return f.bulk(), nil
}

func (f *fakeTriage) PlanSnooze(
	_ context.Context,
	req *inboundmessageservice.SnoozeRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.queuePlan, f.plan(req)
}

func (f *fakeTriage) Wake(
	_ context.Context,
	req *inboundmessageservice.MessageActionRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanWake(
	_ context.Context,
	req *inboundmessageservice.MessageActionRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.queuePlan, f.plan(req)
}

func (f *fakeTriage) StopWaiting(
	_ context.Context,
	req *inboundmessageservice.MessageActionRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(stopWaitingCall{req})

	return f.message, nil
}

func (f *fakeTriage) PlanStopWaiting(
	_ context.Context,
	req *inboundmessageservice.MessageActionRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.queuePlan, f.plan(stopWaitingCall{req})
}

type stopWaitingCall struct {
	req *inboundmessageservice.MessageActionRequest
}

func (f *fakeTriage) WaitForReply(
	_ context.Context,
	req *inboundmessageservice.WaitForReplyRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanWaitForReply(
	_ context.Context,
	req *inboundmessageservice.WaitForReplyRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.queuePlan, f.plan(req)
}

func (f *fakeTriage) Unlink(
	_ context.Context,
	req *inboundmessageservice.UnlinkRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanUnlink(
	_ context.Context,
	req *inboundmessageservice.UnlinkRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.queuePlan, f.plan(req)
}

func (f *fakeTriage) HandOff(
	_ context.Context,
	req *inboundmessageservice.HandOffRequest,
) (*inboundmessageports.HandOffResult, error) {
	f.write(req)

	return &inboundmessageports.HandOffResult{ToUserID: req.ToUserID, Affected: 3}, nil
}

func (f *fakeTriage) PlanHandOff(
	_ context.Context,
	req *inboundmessageservice.HandOffRequest,
) (*inboundmessageservice.QueuePlan, error) {
	return f.queuePlan, f.plan(req)
}

func (f *fakeTriage) BlockSender(
	_ context.Context,
	req *triageworkservice.BlockSenderRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanBlockSender(
	_ context.Context,
	req *triageworkservice.BlockSenderRequest,
) (*triageworkservice.SenderPlan, error) {
	return f.senderPlan, f.plan(req)
}

func (f *fakeTriage) UnblockSender(
	_ context.Context,
	req *triageworkservice.UnblockSenderRequest,
) (*inboundmessage.Mailbox, error) {
	f.write(req)

	return f.senderPlan.Mailbox, nil
}

func (f *fakeTriage) PlanUnblockSender(
	_ context.Context,
	req *triageworkservice.UnblockSenderRequest,
) (*triageworkservice.SenderPlan, error) {
	return f.senderPlan, f.plan(req)
}

func (f *fakeTriage) DraftReply(
	_ context.Context,
	req *inboundreplyservice.DraftReplyRequest,
) (*triage.Reply, error) {
	f.write(req)

	return &triage.Reply{MessageID: req.MessageID, Subject: "Re: drafted"}, nil
}

func (f *fakeTriage) PlanDraftReply(
	_ context.Context,
	req *inboundreplyservice.DraftReplyRequest,
) (*inboundreplyservice.DraftPlan, error) {
	return f.draftPlan, f.plan(req)
}

func (f *fakeTriage) SaveDraft(
	_ context.Context,
	req *inboundreplyservice.SaveDraftRequest,
) (*triage.Reply, error) {
	f.write(req)

	return &triage.Reply{MessageID: req.MessageID, Subject: req.Subject, Body: req.Body}, nil
}

func (f *fakeTriage) PlanSaveDraft(
	_ context.Context,
	req *inboundreplyservice.SaveDraftRequest,
) (*inboundreplyservice.DraftPlan, error) {
	return f.draftPlan, f.plan(req)
}

type discardDraftCall struct {
	tenant    pagination.TenantInfo
	messageID pulid.ID
}

func (f *fakeTriage) DiscardDraft(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundmessage.InboundMessage, error) {
	f.write(discardDraftCall{tenant: tenantInfo, messageID: messageID})

	return f.message, nil
}

func (f *fakeTriage) PlanDiscardDraft(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundreplyservice.DraftPlan, error) {
	return f.draftPlan, f.plan(discardDraftCall{tenant: tenantInfo, messageID: messageID})
}

func (f *fakeTriage) DraftLoad(
	_ context.Context,
	req *triageworkservice.LoadRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanDraftLoad(
	_ context.Context,
	req *triageworkservice.LoadRequest,
) (*triageworkservice.LoadDraftPlan, error) {
	return f.loadDraft, f.plan(req)
}

func (f *fakeTriage) DiscardLoad(
	_ context.Context,
	req *triageworkservice.LoadRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(discardLoadCall{req})

	return f.message, nil
}

type discardLoadCall struct {
	req *triageworkservice.LoadRequest
}

func (f *fakeTriage) PlanDiscardLoad(
	_ context.Context,
	req *triageworkservice.LoadRequest,
) (*triageworkservice.LoadDiscardPlan, error) {
	return f.loadDiscard, f.plan(discardLoadCall{req})
}

func (f *fakeTriage) CreateLoad(
	_ context.Context,
	req *triageworkservice.CreateLoadRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)
	created := *f.message
	created.ActionState.CreatedShipmentID = pulid.MustNew("shp_")

	return &created, nil
}

func (f *fakeTriage) PlanCreateLoad(
	_ context.Context,
	req *triageworkservice.CreateLoadRequest,
) (*triageworkservice.CreateLoadPlan, error) {
	return f.createLoad, f.plan(req)
}

func (f *fakeTriage) File(
	_ context.Context,
	req *triageworkservice.FileRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanFile(
	_ context.Context,
	req *triageworkservice.FileRequest,
) (*triageworkservice.FilePlan, error) {
	return f.filePlan, f.plan(req)
}

func (f *fakeTriage) Unfile(
	_ context.Context,
	req *triageworkservice.UnfileRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanUnfile(
	_ context.Context,
	req *triageworkservice.UnfileRequest,
) (*triageworkservice.UnfilePlan, error) {
	return f.unfilePlan, f.plan(req)
}

func (f *fakeTriage) LogException(
	_ context.Context,
	req *triageworkservice.LogExceptionRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanLogException(
	_ context.Context,
	req *triageworkservice.LogExceptionRequest,
) (*triageworkservice.ExceptionPlan, error) {
	return f.exceptionPlan, f.plan(req)
}

func (f *fakeTriage) DecideStops(
	_ context.Context,
	req *triageworkservice.DecideStopsRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f *fakeTriage) PlanDecideStops(
	_ context.Context,
	req *triageworkservice.DecideStopsRequest,
) (*triageworkservice.StopsPlan, error) {
	return f.stopsPlan, f.plan(req)
}

func (f *fakeTriage) StopTimes(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*inboundmessageports.StopTimes, error) {
	return f.stopTimes, nil
}

type fakeConflictDesk struct {
	*fakeTriage
}

func (f fakeConflictDesk) Resolve(
	_ context.Context,
	req *triageconflictservice.ResolveRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f fakeConflictDesk) PlanResolve(
	_ context.Context,
	req *triageconflictservice.ResolveRequest,
) (*triageconflictservice.ConflictPlan, error) {
	return f.conflictPlan, f.plan(req)
}

func (f fakeConflictDesk) Reopen(
	_ context.Context,
	req *triageconflictservice.ReopenRequest,
) (*inboundmessage.InboundMessage, error) {
	f.write(req)

	return f.message, nil
}

func (f fakeConflictDesk) PlanReopen(
	_ context.Context,
	req *triageconflictservice.ReopenRequest,
) (*triageconflictservice.ConflictPlan, error) {
	return f.conflictPlan, f.plan(req)
}
