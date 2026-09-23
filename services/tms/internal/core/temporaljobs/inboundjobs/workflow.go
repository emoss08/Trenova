package inboundjobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// errNoDocument is the finalize workflow reporting success without a document.
// It is its own error because it means the upload session was consumed and
// there is nothing to extract — a different thing from the upload failing.
var errNoDocument = errors.New("upload finalization produced no document")

// settleOptions allow for a model call and a handful of lookups. The timeout is
// generous because a provider under load is slow rather than broken, and the
// retry is bounded because a message that will not classify after four tries
// wants a person, not a fifth.
// settleAttempts bounds the settle's retries. A model that cannot be asked is
// asked again on each of them; on the last, the message goes to review.
const settleAttempts = 4

var settleOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    settleAttempts,
	},
}

// failOptions are separate and tighter. Recording a failure is a single write,
// and retrying it for minutes would leave the message looking unprocessed for
// exactly as long as the thing that is trying to say it failed.
var failOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval: 2 * time.Second,
		MaximumAttempts: 3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        temporaltype.ProcessInboundMessageWorkflowName,
			Fn:          ProcessInboundMessageWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Reads a staged inbound message and decides what it is.",
		},
		{
			Name:        temporaltype.InboundMessageRetentionWorkflowName,
			Fn:          InboundMessageRetentionWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Removes inbound messages settled more than six months ago.",
		},
	}
}

// retentionOptions give a sweep over every tenant room to finish, heartbeating
// per tenant so a worker lost halfway is noticed in minutes rather than hours.
var retentionOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval: 30 * time.Second,
		MaximumAttempts: 3,
	},
}

// InboundMessageRetentionWorkflow removes the mail nobody needs any more.
func InboundMessageRetentionWorkflow(
	ctx workflow.Context,
) (*InboundMessageRetentionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, retentionOptions)

	var a *Activities
	result := new(InboundMessageRetentionResult)
	if err := workflow.ExecuteActivity(ctx, a.InboundMessageRetentionActivity).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Inbound message retention workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

// ProcessInboundMessageWorkflow settles one message.
//
// When settling fails for good, the workflow does not simply end: it records
// the failure on the message so the inbox shows it waiting on a person. A
// message left at Received looks like one still being worked on, and that is
// the state nobody goes and checks.
func ProcessInboundMessageWorkflow(
	ctx workflow.Context,
	payload *ProcessInboundMessagePayload,
) (*ProcessInboundMessageResult, error) {
	var a *Activities

	settleCtx := workflow.WithActivityOptions(ctx, settleOptions)

	// Attachments first: what a message is about is often only legible once its
	// files have been read, so classifying before they land would throw away
	// the strongest signal the message carries.
	processAttachments(ctx, payload)

	result := new(ProcessInboundMessageResult)
	err := workflow.ExecuteActivity(settleCtx, a.SettleInboundMessageActivity, payload).
		Get(settleCtx, result)
	if err == nil {
		return result, nil
	}

	workflow.GetLogger(ctx).Error("inbound message could not be settled",
		"messageId", payload.MessageID.String(), "error", err)

	failCtx := workflow.WithActivityOptions(ctx, failOptions)
	failErr := workflow.ExecuteActivity(failCtx, a.FailInboundMessageActivity,
		&FailInboundMessagePayload{
			BasePayload: payload.BasePayload,
			MessageID:   payload.MessageID,
			Code:        "SETTLE_FAILED",
			Reason:      err.Error(),
		}).Get(failCtx, nil)
	if failErr != nil {
		// Both failed. The original is the one worth reporting: the second is
		// only the attempt to write the first one down.
		workflow.GetLogger(ctx).Error("could not record the failure either",
			"messageId", payload.MessageID.String(), "error", failErr)
	}

	return nil, err
}

// The attachment pass has three separate timeouts because the three things it
// waits on fail differently: an upload finalization is a bounded piece of work,
// a poll is one read, and the extraction itself is a model call queued behind
// whatever else the tenant is extracting.
var (
	attachmentOptions = workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    3 * time.Second,
			BackoffCoefficient: 2,
			MaximumAttempts:    3,
		},
	}
	// extractionDeadline is how long a file is given before the pipeline stops
	// waiting on it. It is a deadline rather than an unbounded wait because
	// EnqueueExtraction returns nil on every one of its gates — a disabled
	// profile, a missing worker, a tenant control, even a failed workflow start
	// — so a document whose extraction was never begun sits at Pending with no
	// error anywhere, and waiting on it would be waiting forever.
	extractionDeadline = 10 * time.Minute
	extractionPoll     = 20 * time.Second
)

// processAttachments carries each file through the document pipeline.
//
// Nothing here fails the message. A forwarded tender whose PDF will not parse
// is still a tender somebody has to answer, and a message discarded because one
// of its files was unreadable is a message the sender believes was received.
func processAttachments(ctx workflow.Context, payload *ProcessInboundMessagePayload) {
	var a *Activities

	actCtx := workflow.WithActivityOptions(ctx, attachmentOptions)
	logger := workflow.GetLogger(ctx)

	listed := new(ListInboundAttachmentsResult)
	if err := workflow.ExecuteActivity(actCtx, a.ListInboundAttachmentsActivity, payload).
		Get(actCtx, listed); err != nil {
		logger.Error("could not read a message's attachments",
			"messageId", payload.MessageID.String(), "error", err)

		return
	}

	for _, ref := range listed.Attachments {
		documentID, err := finalizeAttachment(ctx, payload, ref)
		if err != nil {
			logger.Error("an inbound attachment could not be finalized",
				"attachmentId", ref.AttachmentID.String(), "error", err)
			recordAttachment(actCtx, payload, ref, pulid.Nil,
				"The file could not be stored as a document.")

			continue
		}

		recordAttachment(actCtx, payload, ref, documentID, "")
		awaitExtraction(ctx, payload, ref)
	}
}

func finalizeAttachment(
	ctx workflow.Context,
	payload *ProcessInboundMessagePayload,
	ref AttachmentRef,
) (pulid.ID, error) {
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: "document-upload-finalize-" + ref.SessionID.String(),
		TaskQueue:  temporaltype.UploadTaskQueue,
	})

	finalized := new(documentuploadjobs.FinalizeUploadResult)
	if err := workflow.ExecuteChildWorkflow(
		childCtx,
		documentuploadjobs.FinalizeDocumentUploadWorkflow,
		&documentuploadjobs.FinalizeUploadPayload{
			BasePayload:   payload.BasePayload,
			SessionID:     ref.SessionID,
			PrincipalType: services.PrincipalTypeSystem,
		},
	).Get(ctx, finalized); err != nil {
		return pulid.Nil, err
	}
	if finalized.DocumentID == nil || finalized.DocumentID.IsNil() {
		return pulid.Nil, errNoDocument
	}

	return *finalized.DocumentID, nil
}

// awaitExtraction polls the document's content status rather than waiting on
// the document.extracted event. That event publishes from exactly one place —
// the draft upsert — so a document that fails anywhere before it publishes
// nothing at all, and a workflow waiting on it would hang to its timeout with
// no record of what went wrong.
func awaitExtraction(
	ctx workflow.Context,
	payload *ProcessInboundMessagePayload,
	ref AttachmentRef,
) {
	var a *Activities

	actCtx := workflow.WithActivityOptions(ctx, attachmentOptions)
	deadline := workflow.Now(ctx).Add(extractionDeadline)

	for {
		giveUp := !workflow.Now(ctx).Add(extractionPoll).Before(deadline)

		state := new(AttachmentExtractionState)
		err := workflow.ExecuteActivity(actCtx, a.PollInboundAttachmentActivity,
			&PollInboundAttachmentPayload{
				BasePayload: payload.BasePayload,
				Attachment:  ref,
				GiveUp:      giveUp,
			}).Get(actCtx, state)
		if err != nil {
			workflow.GetLogger(ctx).Error("could not read an attachment's extraction",
				"attachmentId", ref.AttachmentID.String(), "error", err)

			return
		}
		if state.Terminal || giveUp {
			return
		}

		if err = workflow.Sleep(ctx, extractionPoll); err != nil {
			return
		}
	}
}

func recordAttachment(
	ctx workflow.Context,
	payload *ProcessInboundMessagePayload,
	ref AttachmentRef,
	documentID pulid.ID,
	failureText string,
) {
	var a *Activities

	if err := workflow.ExecuteActivity(ctx, a.RecordInboundAttachmentActivity,
		&RecordInboundAttachmentPayload{
			BasePayload: payload.BasePayload,
			Attachment:  ref,
			DocumentID:  documentID,
			FailureText: failureText,
		}).Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Error("could not record an inbound attachment's document",
			"attachmentId", ref.AttachmentID.String(), "error", err)
	}
}
