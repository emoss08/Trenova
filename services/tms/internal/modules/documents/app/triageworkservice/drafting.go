package triageworkservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triageworkjobs/triageworkcontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.temporal.io/sdk/client"
)

type DraftManyRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MessageIDs []pulid.ID
}

func (s *Service) DraftMany(ctx context.Context, req *DraftManyRequest) (int, error) {
	ids := sliceutils.Dedupe(req.MessageIDs)
	if len(ids) == 0 {
		return 0, nil
	}
	if len(ids) > triageworkcontract.MaxDraftBatch {
		return 0, errortypes.NewValidationError(
			"messageIds", errortypes.ErrInvalid,
			"Draft at most {0} messages at once", triageworkcontract.MaxDraftBatch,
		)
	}
	if !s.nova.Available(triagenova.Background) {
		return 0, triagenova.ErrUnavailable
	}
	if s.workflows == nil || !s.workflows.Enabled() {
		return 0, errortypes.NewBusinessError(
			"Drafting many messages at once needs a background worker, which this installation does not run",
		)
	}

	batchID := pulid.MustNew("tdrf_")
	if _, err := s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:            triageworkcontract.DraftManyWorkflowID(batchID),
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		StaticSummary: "Draft replies and loads for Triage",
	}, triageworkcontract.DraftManyWorkflowName, &triageworkcontract.DraftManyPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			UserID:         req.UserID,
			Timestamp:      s.now(),
		},
		UserID:     req.UserID,
		MessageIDs: ids,
	}); err != nil {
		return 0, err
	}

	return len(ids), nil
}

func (s *Service) DraftOne(ctx context.Context, payload *triageworkcontract.DraftOnePayload) (bool, error) {
	tenantInfo := pagination.TenantInfo{OrgID: payload.OrganizationID, BuID: payload.BusinessUnitID}
	message, err := s.message(ctx, tenantInfo, payload.MessageID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return false, nil
		}

		return false, err
	}
	if !message.NeedsReview() {
		return false, nil
	}

	latest, err := s.latestReply(ctx, message)
	if err != nil {
		return false, err
	}

	drafted := false
	for _, action := range triage.Actions(triage.ActionInput{Message: message, Reply: latest}) {
		if !action.NovaCanDo || action.Done {
			continue
		}
		switch action.Kind {
		case triage.ActionReply:
			if _, err = s.replies.DraftReplyFor(ctx, message, triagenova.Background); err != nil {
				return drafted, err
			}
			drafted = true
		case triage.ActionBuildLoad:
			if _, err = s.draftLoad(ctx, message, triagenova.Background); err != nil {
				return drafted, err
			}
			drafted = true
		case triage.ActionFile, triage.ActionCheckStops, triage.ActionLogException,
			triage.ActionBlockSender, triage.ActionLink:
		}
	}

	return drafted, nil
}

func (s *Service) latestReply(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (*triage.Reply, error) {
	latest, err := s.latest.LatestByMessageIDs(ctx, tenantOf(message), []pulid.ID{message.ID})
	if err != nil {
		return nil, err
	}

	return latest[message.ID], nil
}

func (s *Service) DraftReplyFor(ctx context.Context, message *inboundmessage.InboundMessage) error {
	_, err := s.replies.DraftReplyFor(ctx, message, triagenova.Background)

	return err
}

func (s *Service) DraftLoadFor(ctx context.Context, message *inboundmessage.InboundMessage) error {
	if message.Attachments == nil {
		attachments, err := s.messages.ListAttachments(ctx, message.ID, tenantOf(message))
		if err != nil {
			return err
		}
		message.Attachments = attachments
	}
	_, err := s.draftLoad(ctx, message, triagenova.Background)

	return err
}

var _ inboundmessageports.TriageDrafter = (*Service)(nil)

var _ inboundmessageports.TriageFiler = (*Service)(nil)
