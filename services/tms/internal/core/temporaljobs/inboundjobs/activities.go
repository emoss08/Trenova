package inboundjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Inbound *inboundmessageservice.Service
	Logger  *zap.Logger
}

type Activities struct {
	inbound *inboundmessageservice.Service
	l       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{inbound: p.Inbound, l: p.Logger.Named("inbound-activities")}
}

// SettleInboundMessageActivity reads a staged message and decides what it is.
//
// The name carries its domain because Temporal's activity registry is keyed by
// method name across a whole task queue, and this one shares system-queue with
// several other packages.
func (a *Activities) SettleInboundMessageActivity(
	ctx context.Context,
	payload *ProcessInboundMessagePayload,
) (*ProcessInboundMessageResult, error) {
	result, err := a.inbound.ProcessMessage(
		ctx,
		payload.MessageID,
		pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
	)
	if err != nil {
		a.l.Error("failed to settle an inbound message",
			zap.String("messageId", payload.MessageID.String()), zap.Error(err))

		return nil, err
	}

	return &ProcessInboundMessageResult{
		MessageID:      result.MessageID,
		Status:         result.Status,
		Classification: result.Classification,
		Handled:        result.Handled,
		Matched:        result.Matched,
	}, nil
}

// FailInboundMessageActivity records that the pipeline gave up.
//
// A message left mid-pipeline is worse than one that failed visibly: it looks
// like it is still being worked on, so nobody goes to look at it.
func (a *Activities) FailInboundMessageActivity(
	ctx context.Context,
	payload *FailInboundMessagePayload,
) error {
	return a.inbound.MarkFailed(
		ctx,
		payload.MessageID,
		pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
		payload.Code,
		payload.Reason,
	)
}

// ListInboundAttachmentsActivity reports the files that still have a document
// pipeline run owed to them.
//
// A file already carrying a document id is left out, so a workflow retry
// finalizes nothing twice.
func (a *Activities) ListInboundAttachmentsActivity(
	ctx context.Context,
	payload *ProcessInboundMessagePayload,
) (*ListInboundAttachmentsResult, error) {
	refs, err := a.inbound.PendingAttachments(ctx, payload.MessageID, tenantOf(payload.BasePayload))
	if err != nil {
		return nil, err
	}

	return &ListInboundAttachmentsResult{Attachments: refs}, nil
}

// RecordInboundAttachmentActivity writes down what the upload produced.
func (a *Activities) RecordInboundAttachmentActivity(
	ctx context.Context,
	payload *RecordInboundAttachmentPayload,
) error {
	return a.inbound.RecordAttachmentDocument(
		ctx,
		payload.Attachment,
		payload.DocumentID,
		payload.FailureText,
		tenantOf(payload.BasePayload),
	)
}

// PollInboundAttachmentActivity reads how far the document pipeline has got,
// or records that it ran out of time.
func (a *Activities) PollInboundAttachmentActivity(
	ctx context.Context,
	payload *PollInboundAttachmentPayload,
) (*AttachmentExtractionState, error) {
	tenantInfo := tenantOf(payload.BasePayload)

	state, err := a.inbound.PollAttachmentExtraction(ctx, payload.Attachment, tenantInfo)
	if err != nil {
		return nil, err
	}
	if state.Terminal || !payload.GiveUp {
		return state, nil
	}

	if err = a.inbound.GiveUpOnAttachment(
		ctx, payload.Attachment, state.Status, tenantInfo,
	); err != nil {
		a.l.Error("could not record that an attachment ran out of time",
			zap.String("attachmentId", payload.Attachment.AttachmentID.String()),
			zap.Error(err))
	}
	state.Terminal = true

	return state, nil
}

func tenantOf(base temporaltype.BasePayload) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: base.OrganizationID, BuID: base.BusinessUnitID}
}
