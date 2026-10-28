package inboundmessageresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/loaderfields"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// optionalPulid renders a nullable id. A nil id is absent rather than an empty
// string, because "" would read on the client as an id that is known and blank.
func inboundOptionalID(id pulid.ID) *string {
	if id.IsNil() {
		return nil
	}
	value := id.String()

	return &value
}

func inboundMessageConnection(
	result *pagination.CursorListResult[*inboundmessage.InboundMessage],
) (*gqlmodel.InboundMessageConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *inboundmessage.InboundMessage, cursor string) *gqlmodel.InboundMessageEdge {
			return &gqlmodel.InboundMessageEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.InboundMessageEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.InboundMessageConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

// inboundPreviewLength is how much of a body a list row shows: about two lines
// at the reading pane's narrowest.
const inboundPreviewLength = 180

func mailboxSettings(input gqlmodel.InboundMailboxInput) inboundmessageservice.MailboxSettings {
	return inboundmessageservice.MailboxSettings{
		Name:          input.Name,
		Address:       input.Address,
		Provider:      input.Provider,
		Purpose:       stringutils.FromPtr(input.Purpose),
		ReviewPolicy:  input.ReviewPolicy,
		MinConfidence: input.MinConfidence,
		Status:        input.Status,
	}
}

func unixPtr(value int64) *int {
	if value == 0 {
		return nil
	}
	converted := int(value)

	return &converted
}

func responseTargetMinutes(message *inboundmessage.InboundMessage) int {
	if message.DueAt <= message.ReceivedAt {
		return 0
	}

	return int((message.DueAt - message.ReceivedAt) / 60)
}

func handledByNova(message *inboundmessage.InboundMessage) bool {
	return message.HandledByRuleID.IsNotNil() || message.HandledAction != "" ||
		message.Status == inboundmessage.StatusClassified
}

func snoozeView(message *inboundmessage.InboundMessage) *inboundmessageports.SnoozeView {
	if !message.IsSnoozed() {
		return nil
	}

	view := &inboundmessageports.SnoozeView{
		WatchShipmentID: message.SnoozeWatchID,
		SnoozedAt:       message.SnoozedAt,
		SnoozedBy:       message.SnoozedBy,
		Event:           stringutils.NilIfEmpty(message.SnoozeEvent),
		DocumentTypeID:  pulid.PtrOrNil(message.SnoozeDocumentTypeID),
	}
	if message.SnoozedUntil > 0 {
		until := message.SnoozedUntil
		view.Until = &until
	}

	return view
}

func (d *Deps) visibleShipment(
	ctx context.Context,
	shipmentID pulid.ID,
) (*shipmentports.ShipmentSummary, error) {
	if shipmentID.IsNil() {
		return nil, nil //nolint:nilnil // a missing or hidden record resolves to null
	}

	authCtx, err := d.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !d.HasPermission(ctx, authCtx, permission.ResourceShipment, permission.OpRead) {
		return nil, nil //nolint:nilnil // a missing or hidden record resolves to null
	}
	if err = loaderfields.RequireLoaders(ctx); err != nil {
		return nil, err
	}

	return loaderfields.OptionalMatch(loaders.ShipmentSummaryByID(ctx, shipmentID.String()))
}

func (r *MutationResolver) messageAction(
	ctx context.Context,
	id string,
) (*inboundmessageservice.MessageActionRequest, error) {
	authCtx, err := r.RequirePermission(
		ctx, permission.ResourceInboundMessage, permission.OpUpdate,
	)
	if err != nil {
		return nil, err
	}

	messageID, err := pulid.MustParse(id)
	if err != nil {
		return nil, err
	}

	return &inboundmessageservice.MessageActionRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		ActorID:    actorutil.FromAuthContext(authCtx).UserID,
		MessageID:  messageID,
	}, nil
}

func (d *Deps) requireViewer(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, err := d.RequirePermission(
		ctx, permission.ResourceInboundMessage, permission.OpRead,
	)
	if err != nil {
		return nil, err
	}
	if authCtx.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Saved views belong to a signed-in person")
	}

	return authCtx, nil
}

func folderRequest(
	input *gqlmodel.InboundMessagesInput,
	tenantInfo pagination.TenantInfo,
	viewerID pulid.ID,
) (*inboundmessageports.ListInboundMessagesRequest, error) {
	req := &inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Query:      stringutils.FromPtr(input.Query),
		},
		Statuses:        input.Statuses,
		CollapseThreads: gqlscalar.BoolValue(input.CollapseThreads),
		ThreadKey:       stringutils.FromPtr(input.ThreadKey),
		ViewerID:        viewerID,
	}
	if input.Lane != nil {
		req.Lane = *input.Lane
	}
	if input.Sort != nil {
		req.Sort = *input.Sort
	}
	if input.Classification != nil {
		req.Classification = *input.Classification
	}
	if input.Since != nil {
		req.Since = int64(*input.Since)
	}

	var err error
	if req.MailboxID, err = gqlid.OptionalID(input.MailboxID); err != nil {
		return nil, err
	}
	if req.ShipmentID, err = gqlid.OptionalID(input.ShipmentID); err != nil {
		return nil, err
	}
	if req.OwnerID, err = gqlid.OptionalID(input.OwnerID); err != nil {
		return nil, err
	}
	if err = inboundmessageservice.PrepareList(req); err != nil {
		return nil, err
	}

	return req, nil
}

func selectionOf(
	input gqlmodel.TriageSelectionInput,
	tenantInfo pagination.TenantInfo,
	viewerID pulid.ID,
) (inboundmessageservice.Selection, error) {
	if len(input.Ids) > 0 && input.Folder != nil {
		return inboundmessageservice.Selection{}, errortypes.NewValidationError(
			"selection", errortypes.ErrInvalid, "Choose some messages or a folder, not both",
		)
	}
	if len(input.Ids) > 0 {
		ids, err := gqlid.ParseIDs(input.Ids)
		if err != nil {
			return inboundmessageservice.Selection{}, err
		}

		return inboundmessageservice.Selection{IDs: ids}, nil
	}
	if input.Folder == nil {
		return inboundmessageservice.Selection{}, errortypes.NewValidationError(
			"selection", errortypes.ErrRequired, "Choose some messages or a folder",
		)
	}

	folder, err := folderRequest(input.Folder, tenantInfo, viewerID)
	if err != nil {
		return inboundmessageservice.Selection{}, err
	}

	return inboundmessageservice.Selection{Folder: folder}, nil
}
