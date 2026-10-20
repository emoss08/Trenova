package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundMessages struct {
	message *inboundmessage.InboundMessage
	items   []*inboundmessage.InboundMessage

	gotGet  *inboundmessageports.GetInboundMessageByIDRequest
	gotList *inboundmessageports.ListInboundMessagesRequest
}

func (f *fakeInboundMessages) GetByID(
	_ context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	f.gotGet = &req
	return f.message, nil
}

func (f *fakeInboundMessages) List(
	_ context.Context,
	req *inboundmessageports.ListInboundMessagesRequest,
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

	result, err := newGetInboundMessageTool(messages, nil).Query(t.Context(), params)
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

	result, err := newGetInboundMessageTool(messages, nil).Query(
		t.Context(),
		testParams(map[string]any{"messageId": messages.message.ID.String()}),
	)
	require.NoError(t, err)

	view := result.(inboundMessageView)
	assert.Equal(t, inboundBodyChars, len([]rune(view.Body)))
	assert.True(t, view.BodyTruncated)

	messages.message.TextBody = "short"
	result, err = newGetInboundMessageTool(messages, nil).Query(
		t.Context(),
		testParams(map[string]any{"messageId": messages.message.ID.String()}),
	)
	require.NoError(t, err)
	assert.False(t, result.(inboundMessageView).BodyTruncated)
}

func TestGetInboundMessage_RefusesWithoutAnID(t *testing.T) {
	t.Parallel()

	messages := &fakeInboundMessages{}
	_, err := newGetInboundMessageTool(messages, nil).
		Query(t.Context(), testParams(map[string]any{}))

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
			name:   "waiting means both states a person has to clear",
			params: map[string]any{"status": "waiting"},
			statuses: []inboundmessage.Status{
				inboundmessage.StatusInReview,
				inboundmessage.StatusQuarantined,
			},
			limit: inboundListDefaultLimit,
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
			_, err := newListInboundMessagesTool(messages, nil, nil).Query(t.Context(), params)
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

	result, err := newListInboundMessagesTool(
		messages, nil, nil,
	).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome := result.(searchOutcome)
	rows := outcome.Items.([]inboundMessageRow)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].NeedsReview)
	assert.LessOrEqual(t, len([]rune(rows[0].Preview)), inboundPreviewChars+1)
	assert.Contains(t, outcome.Columns, "subject")
}

func TestListInboundMessages_RowsHandOutTheMailboxAndNoneOfItsSecrets(t *testing.T) {
	t.Parallel()

	mailboxID := pulid.MustNew("imbx_")
	messages := &fakeInboundMessages{items: []*inboundmessage.InboundMessage{{
		ID:          pulid.MustNew("inm_"),
		MailboxID:   mailboxID,
		Status:      inboundmessage.StatusInReview,
		FromAddress: "ap@broker.example",
		Subject:     "Invoice 4471",
		Mailbox: &inboundmessage.Mailbox{
			ID:             mailboxID,
			Address:        "billing@carrier.example",
			TokenHash:      "stored-token-hash",
			SigningSecret:  "provider-signing-secret",
			ProviderAPIKey: "re_provider_key",
			TestToken:      "mailbox-test-token",
		},
	}}}

	result, err := newListInboundMessagesTool(
		messages, nil, nil,
	).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome := result.(searchOutcome)
	rows := outcome.Items.([]inboundMessageRow)
	require.Len(t, rows, 1)
	assert.Equal(t, mailboxID.String(), rows[0].MailboxID)
	assert.Equal(t, "billing@carrier.example", rows[0].Mailbox)
	assert.Contains(t, outcome.Columns, "mailboxId")

	encoded, err := sonic.MarshalString(result)
	require.NoError(t, err)
	for _, secret := range []string{
		"stored-token-hash", "provider-signing-secret", "re_provider_key", "mailbox-test-token",
	} {
		assert.NotContains(t, encoded, secret)
	}
}

func TestGetInboundMessage_NamesTheMailboxIDTheListFiltersBy(t *testing.T) {
	t.Parallel()

	mailboxID := pulid.MustNew("imbx_")
	messages := &fakeInboundMessages{message: &inboundmessage.InboundMessage{
		ID:        pulid.MustNew("inm_"),
		MailboxID: mailboxID,
		Mailbox:   &inboundmessage.Mailbox{ID: mailboxID, Address: "loads@carrier.example"},
	}}

	result, err := newGetInboundMessageTool(messages, nil).Query(
		t.Context(),
		testParams(map[string]any{"messageId": messages.message.ID.String()}),
	)
	require.NoError(t, err)

	view := result.(inboundMessageView)
	assert.Equal(t, mailboxID.String(), view.MailboxID)
	assert.Equal(t, "loads@carrier.example", view.Mailbox)

	schema := newListInboundMessagesTool(messages, nil, nil).ParamSchema()
	properties, _ := schema["properties"].(map[string]any)
	mailbox, _ := properties["mailboxId"].(map[string]any)
	description, _ := mailbox["description"].(string)
	assert.Contains(t, description, "inbox_get_inbound_message")
	assert.NotContains(t, description, "No tool lists")
}

func TestListInboundMessages_RefusesAnUnknownFilterRatherThanDroppingIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params map[string]any
		want   string
	}{
		{
			name:   "an unknown status",
			params: map[string]any{"status": "Deleted"},
			want:   `status "Deleted" is not one the inbox has`,
		},
		{
			name:   "an unknown kind",
			params: map[string]any{"classification": "Spam"},
			want:   `classification "Spam" is not a kind`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			messages := &fakeInboundMessages{}
			_, err := newListInboundMessagesTool(messages, nil, nil).Query(
				t.Context(),
				testParams(tc.params),
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Contains(t, err.Error(), "leave it out")
			assert.Nil(t, messages.gotList, "nothing is listed under a filter that was dropped")
		})
	}
}

func TestListInboundMessages_RanksByWhatTheMailSaysWhenGivenWords(t *testing.T) {
	t.Parallel()

	message := &inboundmessage.InboundMessage{
		ID:             pulid.MustNew("imsg_"),
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationDetentionDispute,
		ReceivedAt:     1_790_000_000,
	}
	searcher := &fakeRetrievalSearcher{messages: &agentports.InboundMessageSearchResult{
		Hits: []agentports.InboundMessageSearchHit{{
			Message: message,
			Subject: "Detention charges on invoice 7781",
			From:    "ap@globex.test",
			Snippet: "We dispute the detention.",
			Match:   agentports.RetrievalMatchWords,
		}},
		Semantics: agentports.RetrievalSemantics{Used: true},
	}}
	messages := &fakeInboundMessages{}
	tool := newListInboundMessagesTool(messages, searcher, &fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"query": "  detention dispute ",
	}))
	require.NoError(t, err)

	assert.Nil(t, messages.gotList, "words are ranked by the searcher, not listed newest first")
	require.Len(t, searcher.requests, 1)
	asked := searcher.requests[0]
	assert.Equal(t, "detention dispute", asked.Query)
	assert.Equal(t, inboundSearchDefaultLimit, asked.Limit)
	assert.NotNil(t, asked.Access)
	assert.Equal(t, agentports.InboundMessageFilter{}, asked.Messages)

	outcome := result.(*retrievalOutcome)
	assert.Equal(t, "words and meaning", outcome.SearchedBy)
	rows := outcome.Items.([]inboundSearchRow)
	require.Len(t, rows, 1)
	assert.Equal(t, message.ID.String(), rows[0].ID)
	assert.Equal(t, "Detention charges on invoice 7781", rows[0].Subject)
	assert.Equal(t, "We dispute the detention.", rows[0].Snippet)
	assert.True(t, rows[0].NeedsReview)
	assert.Equal(t, "DetentionDispute", rows[0].Classification)
	assert.Equal(t, "words", rows[0].Match)
	assert.Equal(t, []agent.RecordRef{
		{EntityType: agent.TaintEntityInboundMessage, ID: message.ID.String()},
	}, outcome.TaintedRecords(), "every message returned is marked as outside text")
	assert.Contains(t, outcome.SearchedFor, `text matching "detention dispute"`)
}

func TestListInboundMessages_RanksWordsWithinTheFilters(t *testing.T) {
	t.Parallel()

	mailboxID := pulid.MustNew("imbx_")
	searcher := &fakeRetrievalSearcher{messages: &agentports.InboundMessageSearchResult{
		Hits: []agentports.InboundMessageSearchHit{},
	}}
	messages := &fakeInboundMessages{}
	tool := newListInboundMessagesTool(messages, searcher, &fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"query":          "reschedule the delivery appointment",
		"status":         "waiting",
		"classification": "StatusRequest",
		"mailboxId":      mailboxID.String(),
		"limit":          500,
	}))
	require.NoError(t, err)

	assert.Nil(t, messages.gotList)
	require.Len(t, searcher.requests, 1)
	asked := searcher.requests[0]
	assert.Equal(t, inboundListLimit, asked.Limit)
	assert.Equal(t, agentports.InboundMessageFilter{
		Statuses: []inboundmessage.Status{
			inboundmessage.StatusInReview,
			inboundmessage.StatusQuarantined,
		},
		Classification: inboundmessage.ClassificationStatusRequest,
		MailboxID:      mailboxID,
	}, asked.Messages)

	outcome := result.(*retrievalOutcome)
	assert.Zero(t, outcome.Count)
	assert.NotEmpty(t, outcome.Note)
	assert.Empty(t, outcome.TaintedRecords())
	for _, term := range []string{"status waiting", "kind StatusRequest", "mailbox " + mailboxID.String()} {
		assert.Contains(t, outcome.SearchedFor, term)
	}
}

func TestListInboundMessages_ListsTheNewestWithoutWords(t *testing.T) {
	t.Parallel()

	searcher := &fakeRetrievalSearcher{}
	messages := &fakeInboundMessages{items: []*inboundmessage.InboundMessage{{
		ID:          pulid.MustNew("inm_"),
		Status:      inboundmessage.StatusInReview,
		FromAddress: "ap@broker.example",
		Subject:     "Invoice 4471",
	}}}
	tool := newListInboundMessagesTool(messages, searcher, &fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"query":  "   ",
		"status": "InReview",
	}))
	require.NoError(t, err)

	assert.Empty(t, searcher.requests, "blank words list the newest rather than search")
	require.NotNil(t, messages.gotList)
	assert.Empty(t, messages.gotList.Filter.Query, "the list no longer runs its own text match")
	assert.Equal(t, []inboundmessage.Status{inboundmessage.StatusInReview}, messages.gotList.Statuses)
	assert.Equal(t, inboundListDefaultLimit, messages.gotList.Cursor.Limit)

	outcome := result.(searchOutcome)
	rows := outcome.Items.([]inboundMessageRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "Invoice 4471", rows[0].Subject)
}

func TestListInboundMessages_RefusesABadFilterBeforeSearching(t *testing.T) {
	t.Parallel()

	searcher := &fakeRetrievalSearcher{}
	tool := newListInboundMessagesTool(&fakeInboundMessages{}, searcher, &fakePermissions{allowed: true})

	for _, params := range []map[string]any{
		{"query": "detention", "status": "Deleted"},
		{"query": "detention", "classification": "Spam"},
		{"query": "detention", "mailboxId": "not-an-id"},
	} {
		_, err := tool.Query(t.Context(), testParams(params))
		require.Error(t, err, params)
	}
	assert.Empty(t, searcher.requests)
}

func TestInboundMessageFilter_AdmitsOnlyWhatItNames(t *testing.T) {
	t.Parallel()

	mailboxID := pulid.MustNew("imbx_")
	message := &inboundmessage.InboundMessage{
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationTender,
		MailboxID:      mailboxID,
	}

	assert.True(t, (&agentports.InboundMessageFilter{}).Admits(message))
	assert.False(t, (&agentports.InboundMessageFilter{}).Admits(nil))
	assert.True(t, (&agentports.InboundMessageFilter{
		Statuses:       []inboundmessage.Status{inboundmessage.StatusQuarantined, inboundmessage.StatusInReview},
		Classification: inboundmessage.ClassificationTender,
		MailboxID:      mailboxID,
	}).Admits(message))
	assert.False(t, (&agentports.InboundMessageFilter{
		Statuses: []inboundmessage.Status{inboundmessage.StatusActioned},
	}).Admits(message))
	assert.False(t, (&agentports.InboundMessageFilter{
		Classification: inboundmessage.ClassificationStatusRequest,
	}).Admits(message))
	assert.False(t, (&agentports.InboundMessageFilter{MailboxID: pulid.MustNew("imbx_")}).Admits(message))
}
