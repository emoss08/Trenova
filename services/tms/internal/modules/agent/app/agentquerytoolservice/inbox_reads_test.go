package agentquerytoolservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inboxList(
	t *testing.T,
	name string,
	reader graphqlreadports.GraphQLReader,
) agenttoolports.AgentQueryTool {
	t.Helper()

	for _, spec := range inboxReadSpecs() {
		if spec.name != name {
			continue
		}
		tool, err := newGraphQLReadTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
		require.NoError(t, err)
		return tool
	}
	require.FailNow(t, "no spec "+name)

	return nil
}

func inboundList(
	t *testing.T,
	reader graphqlreadports.GraphQLReader,
	searcher retrievalports.RetrievalSearcher,
) agenttoolports.AgentQueryTool {
	t.Helper()

	tool, err := provideListInboundMessagesTool(reader, &fakePermissions{allowed: true}, searcher)
	require.NoError(t, err)

	return tool
}

func inputOf(t *testing.T, req *graphqlreadports.GraphQLReadRequest) map[string]any {
	t.Helper()

	require.NotNil(t, req)
	input, ok := req.Variables["input"].(map[string]any)
	require.True(t, ok, "the read sends its input: %v", req.Variables)

	return input
}

// Without words the inbox is read through the Desk's operation, every filter
// sent inside the input the screen pages by, and each row names its sender
// once, as a mail client shows it.
func TestInboundMessages_ListThroughTheDesksOperationWithoutWords(t *testing.T) {
	t.Parallel()

	mailboxID, ownerID := pulid.MustNew("imbx_").String(), pulid.MustNew("usr_").String()
	reader := &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskInboundMessages": {Data: connectionPage("inboundMessages", map[string]any{
			"id": "imsg_1", "fromName": "Dana Ruiz", "fromAddress": "dana@shipper.example",
			"subject": "Load for Thursday", "status": "InReview", "needsReview": true,
			"preview": "Please cover the attached load.",
		})},
	}}
	searcher := &fakeRetrievalSearcher{}
	tool := inboundList(t, reader, searcher)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{
		"query":          "  ",
		"status":         []any{"InReview", "Quarantined"},
		"classification": "Tender",
		"mailboxId":      mailboxID,
		"ownerId":        ownerID,
		"sortBy":         "Due",
		"limit":          3,
	}))
	require.NoError(t, err)

	assert.Empty(t, searcher.requests, "blank words list the newest rather than search")
	input := inputOf(t, reader.sent("DeskInboundMessages"))
	assert.Equal(t, map[string]any{
		"first":          3,
		"statuses":       []any{"InReview", "Quarantined"},
		"classification": "Tender",
		"mailboxId":      mailboxID,
		"ownerId":        ownerID,
		"sort":           "Due",
	}, input)
	outcome, ok := data.(graphQLListOutcome)
	require.True(t, ok)
	require.Len(t, outcome.Items, 1)
	row := outcome.Items[0]
	assert.Equal(t, "Dana Ruiz <dana@shipper.example>", row[inboundFieldFrom])
	assert.NotContains(t, row, inboundFieldFromName)
	assert.NotContains(t, row, inboundFieldFromAddress)
	assert.True(t, outcome.HasMore)
	assert.NotEmpty(t, outcome.NextCursor)

	next, err := tool.Query(t.Context(), keyedCall(map[string]any{
		"after": outcome.NextCursor, "query": "  ", "status": []any{"InReview", "Quarantined"},
		"classification": "Tender", "mailboxId": mailboxID, "ownerId": ownerID,
		"sortBy": "Due", "limit": 3,
	}))
	require.NoError(t, err)
	require.NotNil(t, next)
	paged := inputOf(t, reader.requests[len(reader.requests)-1])
	assert.Equal(t, nextPage, paged["after"], "the next page reads after the root's own cursor")
}

// An id the call names by something other than an id is refused before
// anything is read.
func TestInboundMessages_RefuseAMailboxThatIsNotAnID(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{}
	tool := inboundList(t, reader, &fakeRetrievalSearcher{})

	_, err := tool.Query(t.Context(), keyedCall(map[string]any{"mailboxId": "loads@carrier"}))
	require.Error(t, err)
	assert.Empty(t, reader.requests)
}

// Words are ranked by what the mail says, on the retrieval index, and no
// GraphQL read is made: the screen's search matches text only.
func TestInboundMessages_RankByWhatTheMailSaysWhenGivenWords(t *testing.T) {
	t.Parallel()

	message := &inboundmessage.InboundMessage{
		ID:             pulid.MustNew("imsg_"),
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationDetentionDispute,
		ReceivedAt:     1_790_000_000,
	}
	searcher := &fakeRetrievalSearcher{messages: &retrievalports.InboundMessageSearchResult{
		Hits: []retrievalports.InboundMessageSearchHit{{
			Message: message,
			Subject: "Detention charges on invoice 7781",
			From:    "ap@globex.test",
			Snippet: "We dispute the detention.",
			Match:   retrievalports.RetrievalMatchWords,
		}},
		Semantics: retrievalports.RetrievalSemantics{Used: true},
	}}
	reader := &keyedReader{}
	tool := inboundList(t, reader, searcher)

	result, err := tool.Query(t.Context(), keyedCall(map[string]any{
		"query": "  detention dispute ",
	}))
	require.NoError(t, err)

	assert.Empty(t, reader.requests, "words are ranked by the searcher, not listed newest first")
	require.Len(t, searcher.requests, 1)
	asked := searcher.requests[0]
	assert.Equal(t, "detention dispute", asked.Query)
	assert.Equal(t, inboundSearchDefaultLimit, asked.Limit)
	assert.NotNil(t, asked.Access)
	assert.Equal(t, retrievalports.InboundMessageFilter{}, asked.Messages)

	outcome := result.(*retrievalOutcome)
	assert.Equal(t, "words and meaning", outcome.SearchedBy)
	rows := outcome.Items.([]inboundSearchRow)
	require.Len(t, rows, 1)
	assert.Equal(t, message.ID.String(), rows[0].ID)
	assert.Equal(t, "We dispute the detention.", rows[0].Snippet)
	assert.True(t, rows[0].NeedsReview)
	assert.Equal(t, "words", rows[0].Match)
	assert.Equal(t, []agentrecord.RecordRef{
		{EntityType: agent.TaintEntityInboundMessage, ID: message.ID.String()},
	}, outcome.TaintedRecords(), "every message returned is marked as outside text")
	assert.Contains(t, outcome.SearchedFor, `text matching "detention dispute"`)
}

// Words are ranked within the filters the retrieval index keeps, and the
// limit is held to a search's own bound.
func TestInboundMessages_RankWordsWithinTheFilters(t *testing.T) {
	t.Parallel()

	mailboxID := pulid.MustNew("imbx_")
	searcher := &fakeRetrievalSearcher{messages: &retrievalports.InboundMessageSearchResult{
		Hits: []retrievalports.InboundMessageSearchHit{},
	}}
	tool := inboundList(t, &keyedReader{}, searcher)

	result, err := tool.Query(t.Context(), keyedCall(map[string]any{
		"query":          "reschedule the delivery appointment",
		"status":         []any{"InReview", "Quarantined"},
		"classification": "StatusRequest",
		"mailboxId":      mailboxID.String(),
		"limit":          500,
	}))
	require.NoError(t, err)

	require.Len(t, searcher.requests, 1)
	asked := searcher.requests[0]
	assert.Equal(t, inboundSearchMost, asked.Limit)
	assert.Equal(t, retrievalports.InboundMessageFilter{
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
}

// A filter only the listing applies is refused beside words, rather than
// dropped, and so is a bad value, before anything is searched.
func TestInboundMessages_RefuseWhatASearchCannotApply(t *testing.T) {
	t.Parallel()

	searcher := &fakeRetrievalSearcher{}
	reader := &keyedReader{}
	tool := inboundList(t, reader, searcher)

	cases := map[string]map[string]any{
		"an owner":        {"query": "detention", "ownerId": pulid.MustNew("usr_").String()},
		"an order":        {"query": "detention", "sortBy": "Due"},
		"a folder":        {"query": "detention", "lane": "Snoozed"},
		"a status":        {"query": "detention", "status": []any{"Deleted"}},
		"a kind":          {"query": "detention", "classification": "Spam"},
		"a mailbox":       {"query": "detention", "mailboxId": "not-an-id"},
		"a second page":   {"query": "detention", "after": "eyJ9"},
		"one thread only": {"query": "detention", "threadKey": "<a@b>"},
	}
	for name, params := range cases {
		_, err := tool.Query(t.Context(), keyedCall(params))
		require.Error(t, err, name)
	}
	_, err := tool.Query(t.Context(), keyedCall(map[string]any{
		"query": "detention", "ownerId": pulid.MustNew("usr_").String(),
	}))
	assert.ErrorContains(t, err, `"ownerId" lists the inbox and does not apply with query`)
	assert.Empty(t, searcher.requests)
	assert.Empty(t, reader.requests)
}

func TestInboundMessageFilter_AdmitsOnlyWhatItNames(t *testing.T) {
	t.Parallel()

	mailboxID := pulid.MustNew("imbx_")
	message := &inboundmessage.InboundMessage{
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationTender,
		MailboxID:      mailboxID,
	}

	assert.True(t, (&retrievalports.InboundMessageFilter{}).Admits(message))
	assert.False(t, (&retrievalports.InboundMessageFilter{}).Admits(nil))
	assert.True(t, (&retrievalports.InboundMessageFilter{
		Statuses: []inboundmessage.Status{
			inboundmessage.StatusQuarantined, inboundmessage.StatusInReview,
		},
		Classification: inboundmessage.ClassificationTender,
		MailboxID:      mailboxID,
	}).Admits(message))
	assert.False(t, (&retrievalports.InboundMessageFilter{
		Statuses: []inboundmessage.Status{inboundmessage.StatusActioned},
	}).Admits(message))
	assert.False(t, (&retrievalports.InboundMessageFilter{
		Classification: inboundmessage.ClassificationStatusRequest,
	}).Admits(message))
	assert.False(t, (&retrievalports.InboundMessageFilter{
		MailboxID: pulid.MustNew("imbx_"),
	}).Admits(message))
}

// The schema says a search's limit and a listing's apart, and names the
// tool a mailbox's id comes from.
func TestInboundMessages_DescribeBothReads(t *testing.T) {
	t.Parallel()

	properties, _ := inboundList(t, &keyedReader{}, &fakeRetrievalSearcher{}).
		ParamSchema()["properties"].(map[string]any)
	limit, _ := properties[paramLimit].(map[string]any)
	assert.Contains(t, limit["description"], "when you give words")
	mailbox, _ := properties[paramMailboxID].(map[string]any)
	assert.Contains(t, mailbox["description"], "inbox_get_inbound_message")
	assert.Contains(t, properties, paramStatus)
	assert.NotContains(t, properties, argStatuses)
	assert.Contains(t, properties, paramSortBy)
}

// A forwarded thread can run to pages; the desk reads the top of it and is
// told the rest was cut, so it does not reason as if it had read it whole.
func TestInboundMessage_GetBoundsTheBodyAndSaysSo(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("imsg_").String()
	answer := func(body string) *keyedReader {
		return &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
			"DeskInboundMessage": {Data: map[string]any{"inboundMessage": map[string]any{
				"id": id, "textBody": body, "fromName": "", "fromAddress": "ap@broker.example",
				"mailbox": map[string]any{"name": "Loads", "address": "loads@carrier.example"},
			}}},
		}}
	}
	spec := inboundMessageGetSpec()

	reader := answer(strings.Repeat("é", inboundBodyChars+50))
	tool, err := newGraphQLGetTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)
	data, err := tool.Query(t.Context(), keyedCall(map[string]any{"messageId": id}))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"id": id}, reader.sent("DeskInboundMessage").Variables)
	record := data.(map[string]any)
	assert.Len(t, []rune(record[inboundFieldBody].(string)), inboundBodyChars)
	assert.Equal(t, true, record["bodyTruncated"])
	assert.Equal(t, "ap@broker.example", record[inboundFieldFrom])

	reader = answer("short")
	tool, err = newGraphQLGetTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)
	data, err = tool.Query(t.Context(), keyedCall(map[string]any{"messageId": id}))
	require.NoError(t, err)
	assert.NotContains(t, data.(map[string]any), "bodyTruncated")
}

// The profiles listed are the active ones unless the call names others,
// as only an active profile sends.
func TestEmailProfiles_ListTheActiveOnesUnlessAsked(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskEmailProfiles": {Data: connectionPage("emailProfiles", map[string]any{
			"id": "emlprof_1", "name": "Operations", "senderEmail": "ops@carrier.example",
		})},
	}}
	tool := inboxList(t, "inbox_list_email_profiles", reader)
	spec := keyedSpec(t, "inbox_list_email_profiles")
	reached, err := newGraphQLReadTool(spec, reader, &restrictedRole{})
	require.NoError(t, err)

	call := keyedCall(map[string]any{"query": "ops"})
	call.DataAccessCeiling = permission.SensitivityRestricted
	_, err = reached.Query(t.Context(), call)
	require.NoError(t, err)
	input := inputOf(t, reader.sent("DeskEmailProfiles"))
	assert.Equal(t, "ops", input["query"])
	assert.Equal(t, []map[string]any{{"field": "status", "operator": "in",
		"value": []string{"Active"}}}, input["fieldFilters"])

	_, err = tool.Query(t.Context(), keyedCall(map[string]any{"status": []any{"Inactive"}}))
	require.NoError(t, err)
	input = inputOf(t, reader.requests[len(reader.requests)-1])
	assert.Equal(t, []map[string]any{{"field": "status", "operator": "in",
		"value": []string{"Inactive"}}}, input["fieldFilters"])
}

// The stacks still waiting on a person are listed a few at a time, as each
// carries every document it was split into; one stack named by id is read
// instead, and the list's filters are refused beside it.
func TestCaptureBatches_ListOpenStacksOrReadOne(t *testing.T) {
	t.Parallel()

	batchID := pulid.MustNew("cbat_").String()
	reader := &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskCaptureBatches": {Data: connectionPage("captureBatches", map[string]any{
			"id": batchID, "status": "Ready", "items": []any{map[string]any{"id": "citm_1"}},
		})},
		"DeskCaptureBatch": {Data: map[string]any{"captureBatch": map[string]any{
			"id": batchID, "status": "Ready",
		}}},
	}}
	tool := inboxList(t, "inbox_list_capture_batches", reader)

	properties, _ := tool.ParamSchema()["properties"].(map[string]any)
	limit, _ := properties[paramLimit].(map[string]any)
	assert.Equal(t, captureBatchesMost, limit["maximum"])

	_, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)
	input := inputOf(t, reader.sent("DeskCaptureBatches"))
	assert.Equal(t, captureBatchesPage, input["first"])
	assert.Equal(t, []any{"Ready", "PartiallyFiled", "Failed"}, input["statuses"])

	_, err = tool.Query(t.Context(), keyedCall(map[string]any{"limit": 500}))
	require.NoError(t, err)
	input = inputOf(t, reader.requests[len(reader.requests)-1])
	assert.Equal(t, captureBatchesMost, input["first"], "a limit above the bound is clamped")

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramCaptureBatchID: batchID,
	}))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"id": batchID}, reader.sent("DeskCaptureBatch").Variables)
	outcome, _ := pageRecord(data)
	batch, _ := outcome[keyCaptureBatch].(map[string]any)
	assert.Equal(t, batchID, batch["id"])

	_, err = tool.Query(t.Context(), keyedCall(map[string]any{
		paramCaptureBatchID: batchID, "status": []any{"Filed"},
	}))
	assert.ErrorContains(t, err, `parameter "status" lists records`)
}

// The people who can take messages are read with the queue's settings, the
// on-call person marked among them and the day's end written as a time.
func TestTriageAssignees_MarkWhoIsOnCall(t *testing.T) {
	t.Parallel()

	onCall := pulid.MustNew("usr_").String()
	reader := &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
		"TriageAssignees": {Data: map[string]any{"triageAssignees": []any{
			map[string]any{"waiting": 3, "user": map[string]any{"id": onCall, "name": "Ana"}},
			map[string]any{"waiting": 0, "user": map[string]any{"id": "usr_2", "name": "Bo"}},
		}}},
		"TriageSettings": {Data: map[string]any{"triageSettings": map[string]any{
			"onCallUserId": onCall, "endOfDayMinute": 1050, "timezone": "America/Chicago",
		}}},
	}}
	tool := inboxList(t, "inbox_list_triage_assignees", reader)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)
	assert.Empty(t, reader.sent("TriageSettings").Variables)
	outcome := data.(map[string]any)
	items := outcome[outcomeItems].([]map[string]any)
	require.Len(t, items, 2)
	assert.Equal(t, true, items[0]["onCall"])
	assert.NotContains(t, items[1], "onCall")
	settings := outcome[keyTriageSettings].(map[string]any)
	assert.Equal(t, "17:30", settings["endOfDay"])
}
