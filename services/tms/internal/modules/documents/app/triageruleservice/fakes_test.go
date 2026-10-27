package triageruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type fakeMessages struct {
	inboundmessageports.InboundMessageRepository

	stored      *inboundmessage.InboundMessage
	attachments []*inboundmessage.InboundAttachment
}

func (f *fakeMessages) GetByID(
	_ context.Context,
	_ inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	copied := *f.stored

	return &copied, nil
}

func (f *fakeMessages) ListAttachments(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*inboundmessage.InboundAttachment, error) {
	return f.attachments, nil
}

type fakeFiler struct {
	filed int
	calls int
	got   inboundmessageports.FileOnMatchedRequest
	err   error
}

func (f *fakeFiler) FileOnMatchedShipment(
	_ context.Context,
	req inboundmessageports.FileOnMatchedRequest,
) (int, error) {
	f.calls++
	f.got = req

	return f.filed, f.err
}

type fakeDrafter struct {
	replies int
	loads   int
	onLoad  func()
}

func (f *fakeDrafter) DraftReplyFor(context.Context, *inboundmessage.InboundMessage) error {
	f.replies++

	return nil
}

func (f *fakeDrafter) DraftLoadFor(context.Context, *inboundmessage.InboundMessage) error {
	f.loads++
	if f.onLoad != nil {
		f.onLoad()
	}

	return nil
}

type fakeParties struct {
	inboundmessageports.InboundPartyFinder

	customerID pulid.ID
	carrierID  pulid.ID
}

func (f *fakeParties) FindCustomerByEmail(
	context.Context,
	pagination.TenantInfo,
	string,
) (pulid.ID, bool, error) {
	return f.customerID, f.customerID.IsNotNil(), nil
}

func (f *fakeParties) FindCarrierByEmail(
	context.Context,
	pagination.TenantInfo,
	string,
) (pulid.ID, bool, error) {
	return f.carrierID, f.carrierID.IsNotNil(), nil
}

type fakeReplies struct {
	inboundmessageports.InboundReplyReader

	latest map[pulid.ID]*triage.Reply
}

func (f *fakeReplies) LatestByMessageIDs(
	context.Context,
	pagination.TenantInfo,
	[]pulid.ID,
) (map[pulid.ID]*triage.Reply, error) {
	return f.latest, nil
}

type fakeConflicts struct {
	conflicts []inboundmessage.Conflict
	calls     int
}

func (f *fakeConflicts) CheckConflicts(
	context.Context,
	*inboundmessage.InboundMessage,
) ([]inboundmessage.Conflict, error) {
	f.calls++

	return f.conflicts, nil
}

type fakeActivities struct {
	recorded []*triage.Activity
}

func (f *fakeActivities) RecordActivity(_ context.Context, activities ...*triage.Activity) error {
	f.recorded = append(f.recorded, activities...)

	return nil
}

type fakeMailboxes struct {
	inboundmessageports.InboundMailboxRepository

	known map[pulid.ID]*inboundmessage.Mailbox
}

func (f *fakeMailboxes) GetByID(
	_ context.Context,
	req inboundmessageports.GetMailboxByIDRequest,
) (*inboundmessage.Mailbox, error) {
	if mailbox, ok := f.known[req.ID]; ok {
		return mailbox, nil
	}

	return nil, errortypes.NewNotFoundError("Mailbox not found")
}

func (f *fakeMailboxes) List(
	context.Context,
	*inboundmessageports.ListMailboxesRequest,
) (*pagination.ListResult[*inboundmessage.Mailbox], error) {
	items := make([]*inboundmessage.Mailbox, 0, len(f.known))
	for _, mailbox := range f.known {
		items = append(items, mailbox)
	}

	return &pagination.ListResult[*inboundmessage.Mailbox]{
		Items:      items,
		Total:      len(items),
		TotalExact: true,
	}, nil
}

type fixedOrganizations struct {
	timezone string
}

func (f fixedOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: f.timezone}, nil
}
