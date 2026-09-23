package inboundjobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"

	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Inbound *inboundmessageservice.Service
	Tenants repositories.TenantSyncRepository
	Logger  *zap.Logger
}

// settledPurger is the retention slice of the inbox, narrow so the sweep can be
// tested against the tenants it walks rather than a database.
type settledPurger interface {
	PurgeSettled(ctx context.Context, req inboundmessageservice.PurgeSettledRequest) (int, error)
}

type Activities struct {
	inbound *inboundmessageservice.Service
	purger  settledPurger
	tenants repositories.TenantSyncRepository
	l       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		inbound: p.Inbound,
		purger:  p.Inbound,
		tenants: p.Tenants,
		l:       p.Logger.Named("inbound-activities"),
	}
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
		// A model that could not be asked is asked again, the way the
		// provider's answer says, until the last attempt sends the message
		// to review.
		inboundmessageservice.RetryModelFailures(func(err error) bool {
			return modelcall.Transient(err) && !modelcall.FinalAttempt(ctx, settleAttempts)
		}),
	)
	if errors.Is(err, inboundmessageservice.ErrClassificationUnavailable) {
		return nil, modelcall.Classify(err)
	}
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

const (
	// inboundRetentionBatch bounds one delete, so a first sweep over a
	// long-running deployment never holds a transaction across a backlog.
	inboundRetentionBatch = 500
	// inboundRetentionPasses bounds the batches one tenant gets per run, so a
	// very large backlog ends in a predictable time and the next run continues.
	inboundRetentionPasses = 20
)

// InboundMessageRetentionActivity removes settled mail older than the
// retention window from every tenant, a bounded batch at a time. A tenant
// whose sweep fails is recorded and stepped over; the next run picks it up.
func (a *Activities) InboundMessageRetentionActivity(
	ctx context.Context,
) (*InboundMessageRetentionResult, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	before := timeutils.NowUnix() - inboundmessageservice.RetentionDays*24*60*60
	result := &InboundMessageRetentionResult{}
	for _, org := range organizations {
		activity.RecordHeartbeat(ctx, org.ID.String())

		tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
		for range inboundRetentionPasses {
			deleted, pErr := a.purger.PurgeSettled(ctx, inboundmessageservice.PurgeSettledRequest{
				TenantInfo: tenant,
				Before:     before,
				Limit:      inboundRetentionBatch,
			})
			if pErr != nil {
				a.l.Warn("inbound retention failed for an organization",
					zap.String("organizationId", org.ID.String()), zap.Error(pErr))
				result.Failed = append(result.Failed, org.ID.String())

				break
			}
			result.Deleted += deleted
			if deleted < inboundRetentionBatch {
				break
			}
		}
	}

	a.l.Info("inbound retention sweep complete",
		zap.Int("deleted", result.Deleted), zap.Int("failed", len(result.Failed)))

	return result, nil
}
