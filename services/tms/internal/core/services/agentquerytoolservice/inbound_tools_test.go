package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundMessages struct {
	message *inboundmessage.InboundMessage
	items   []*inboundmessage.InboundMessage

	gotGet  *repositories.GetInboundMessageByIDRequest
	gotList *repositories.ListInboundMessagesRequest
}

func (f *fakeInboundMessages) GetByID(
	_ context.Context,
	req repositories.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	f.gotGet = &req
	return f.message, nil
}

func (f *fakeInboundMessages) List(
	_ context.Context,
	req *repositories.ListInboundMessagesRequest,
) (*pagination.CursorListResult[*inboundmessage.InboundMessage], error) {
	f.gotList = req
	return &pagination.CursorListResult[*inboundmessage.InboundMessage]{Items: f.items}, nil
}

func TestGetInboundMessage_ReadsTheTenantsMessageWithItsAttachments(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	messages := &fakeInboundMessages{message: &inboundmessage.InboundMessage{
		ID:             pulid.MustNew("inm_"),
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationTender,
		Confidence:     0.62,
		FromName:       "Dana Ruiz",
		FromAddress:    "dana@shipper.example",
		Subject:        "Load for Thursday",
		TextBody:       "Please cover the attached load.",
		Mailbox:        &inboundmessage.Mailbox{Address: "loads@carrier.example"},
		Attachments: []*inboundmessage.InboundAttachment{
			{
				ID:         pulid.MustNew("inat_"),
				FileName:   "tender.pdf",
				Kind:       inboundmessage.AttachmentRateConfirmation,
				DocumentID: documentID,
			},
			{
				ID:          pulid.MustNew("inat_"),
				FileName:    "macro.xlsm",
				Kind:        inboundmessage.AttachmentUnknown,
				FailureText: "This file type is not accepted.",
			},
		},
	}}
	params := testParams(map[string]any{"messageId": messages.message.ID.String()})

	result, err := newGetInboundMessageTool(messages).Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, messages.gotGet)
	assert.Equal(t, messages.message.ID, messages.gotGet.ID)
	assert.Equal(t, params.OrganizationID, messages.gotGet.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, messages.gotGet.TenantInfo.BuID)
	assert.True(t, messages.gotGet.IncludeAttachments)

	view, ok := result.(inboundMessageView)
	require.True(t, ok)
	assert.True(t, view.NeedsReview)
	assert.Equal(t, "loads@carrier.example", view.Mailbox)
	assert.Contains(t, view.From, "dana@shipper.example")
	assert.Empty(t, view.MatchedShipmentID, "an unmatched message must not report a nil id")
	require.Len(t, view.Attachments, 2)
	assert.Equal(t, documentID.String(), view.Attachments[0].DocumentID)
	assert.Empty(t, view.Attachments[1].DocumentID)
	assert.Equal(t, "This file type is not accepted.", view.Attachments[1].Refused)
}

// A forwarded thread can run to pages; the desk reads the top of it and is told
// the rest was cut, so it does not reason as if it had read the whole thing.
func TestGetInboundMessage_BoundsTheBodyAndSaysSo(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("é", inboundBodyChars+50)
	messages := &fakeInboundMessages{message: &inboundmessage.InboundMessage{
		ID:       pulid.MustNew("inm_"),
		TextBody: long,
	}}

	result, err := newGetInboundMessageTool(messages).Query(
		t.Context(),
		testParams(map[string]any{"messageId": messages.message.ID.String()}),
	)
	require.NoError(t, err)

	view := result.(inboundMessageView)
	assert.Equal(t, inboundBodyChars, len([]rune(view.Body)))
	assert.True(t, view.BodyTruncated)

	messages.message.TextBody = "short"
	result, err = newGetInboundMessageTool(messages).Query(
		t.Context(),
		testParams(map[string]any{"messageId": messages.message.ID.String()}),
	)
	require.NoError(t, err)
	assert.False(t, result.(inboundMessageView).BodyTruncated)
}

func TestGetInboundMessage_RefusesWithoutAnID(t *testing.T) {
	t.Parallel()

	messages := &fakeInboundMessages{}
	_, err := newGetInboundMessageTool(messages).Query(t.Context(), testParams(map[string]any{}))

	require.Error(t, err)
	assert.Nil(t, messages.gotGet, "the repository must not be asked for nothing")
}

func TestListInboundMessages_TranslatesTheFilters(t *testing.T) {
	t.Parallel()

	mailboxID := pulid.MustNew("inmb_")
	cases := []struct {
		name           string
		params         map[string]any
		statuses       []inboundmessage.Status
		classification inboundmessage.Classification
		limit          int
		mailbox        pulid.ID
	}{
		{
			name:     "waiting means both states a person has to clear",
			params:   map[string]any{"status": "waiting"},
			statuses: []inboundmessage.Status{inboundmessage.StatusInReview, inboundmessage.StatusQuarantined},
			limit:    inboundListDefaultLimit,
		},
		{
			name:     "a single status reads case-insensitively",
			params:   map[string]any{"status": "actioned"},
			statuses: []inboundmessage.Status{inboundmessage.StatusActioned},
			limit:    inboundListDefaultLimit,
		},
		{
			name:           "a kind narrows by classification",
			params:         map[string]any{"classification": "StatusRequest"},
			classification: inboundmessage.ClassificationStatusRequest,
			limit:          inboundListDefaultLimit,
		},
		{
			name:   "an unknown status and kind filter nothing rather than everything wrongly",
			params: map[string]any{"status": "Deleted", "classification": "Spam"},
			limit:  inboundListDefaultLimit,
		},
		{
			name:   "the limit is capped",
			params: map[string]any{"limit": 500},
			limit:  inboundListLimit,
		},
		{
			name:    "a mailbox narrows by id",
			params:  map[string]any{"mailboxId": mailboxID.String(), "limit": 3},
			limit:   3,
			mailbox: mailboxID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			messages := &fakeInboundMessages{}
			params := testParams(tc.params)
			_, err := newListInboundMessagesTool(messages).Query(t.Context(), params)
			require.NoError(t, err)

			got := messages.gotList
			require.NotNil(t, got)
			assert.Equal(t, tc.statuses, got.Statuses)
			assert.Equal(t, tc.classification, got.Classification)
			assert.Equal(t, tc.limit, got.Cursor.Limit)
			assert.Equal(t, tc.mailbox, got.MailboxID)
			assert.Equal(t, params.OrganizationID, got.Filter.TenantInfo.OrgID)
		})
	}
}

func TestListInboundMessages_RowsCarryAPreviewNotTheBody(t *testing.T) {
	t.Parallel()

	messages := &fakeInboundMessages{items: []*inboundmessage.InboundMessage{{
		ID:          pulid.MustNew("inm_"),
		Status:      inboundmessage.StatusQuarantined,
		FromAddress: "ap@broker.example",
		Subject:     "Invoice 4471",
		TextBody:    strings.Repeat("word ", 400),
	}}}

	result, err := newListInboundMessagesTool(messages).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome := result.(searchOutcome)
	rows := outcome.Items.([]inboundMessageRow)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].NeedsReview)
	assert.LessOrEqual(t, len([]rune(rows[0].Preview)), inboundPreviewChars+1)
	assert.Contains(t, outcome.Columns, "subject")
}
