package workforcetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageconflictservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func previewOf(
	t *testing.T,
	tool agentports.AgentTool,
	params agentports.ToolExecuteParams,
) *agent.ToolPreview {
	t.Helper()

	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)

	return preview
}

/*
draft_inbound_reply drafts and nothing more. With a body it saves the agent's
own words as the draft; without one it asks Triage's drafter, whose words are
not known until it runs, so that preview says it is partial. Neither sends.
*/
func TestDraftInboundReply_SavesTheBodyOrAsksTheDrafter(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	tool := newDraftInboundReplyTool(f, f)
	f.draftPlan = &inboundreplyservice.DraftPlan{
		Message: f.message,
		Draft: &triage.Reply{
			ToAddress: "dana@shipper.example",
			Subject:   "Re: Where is 88213?",
			Body:      "It delivers tomorrow before noon.",
		},
	}

	written := agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"body":      "  It delivers tomorrow before noon.  ",
	})
	preview := previewOf(t, tool, written)
	assert.False(t, preview.Partial)
	assert.Contains(t, preview.Summary, "Nothing is sent")
	assert.Equal(t, agent.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 0).Operation)
	assert.Empty(t, f.written)

	require.NoError(t, tool.Execute(t.Context(), written))
	saved := f.written[0].(*inboundreplyservice.SaveDraftRequest)
	assert.Equal(t, "It delivers tomorrow before noon.", saved.Body)
	assert.False(t, saved.ReplaceAttachments, "a person's attachments on the draft are kept")
	assert.Equal(t, written.Actor.UserID, saved.UserID)

	asked := agenttoolstest.DeskParams(map[string]any{"messageId": f.message.ID.String()})
	f.draftPlan.Draft = &triage.Reply{DraftedByNova: true}
	assert.True(t, previewOf(t, tool, asked).Partial, "the drafter's words are not known yet")
	require.NoError(t, tool.Execute(t.Context(), asked))
	drafted := f.written[1].(*inboundreplyservice.DraftReplyRequest)
	assert.Equal(t, triagenova.Interactive, drafted.Mode)
	assert.Equal(t, f.message.ID, drafted.MessageID)

	require.Error(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"subject":   "Re: hello",
	})), "a subject alone is not a draft")
	assert.Len(t, f.written, 2)
}

func TestDraftInboundReply_SaysItReplacesTheDraftAlreadyThere(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	f.draftPlan = &inboundreplyservice.DraftPlan{
		Message: f.message,
		Current: &triage.Reply{Subject: "Re: Where is 88213?", Body: "A person's words"},
		Draft:   &triage.Reply{Subject: "Re: Where is 88213?", Body: "The agent's words"},
	}

	params := agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"body":      "The agent's words",
	})
	preview := previewOf(t, newDraftInboundReplyTool(f, f), params)
	assert.Contains(t, preview.Summary, "replacing the draft already there")
	body := agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "body")
	assert.Equal(t, "A person's words", body.Before)
	assert.Equal(t, "The agent's words", body.After)
}

func TestDiscardInboundReplyDraft_ThrowsAwayOnlyADraftThatExists(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	f.draftPlan = &inboundreplyservice.DraftPlan{Message: f.message}
	tool := newDiscardInboundReplyDraftTool(f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.message.ID.String()})

	agenttoolstest.RequireWarning(t, previewOf(t, tool, params), agent.PreviewWarningWouldFail)
	require.ErrorIs(t, tool.Execute(t.Context(), params), errNoReplyDraft)
	assert.Empty(t, f.written)

	f.draftPlan.Current = &triage.Reply{Subject: "Re: Where is 88213?", Body: "Draft"}
	preview := previewOf(t, tool, params)
	assert.Equal(t, agent.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, preview, 0).Operation)

	require.NoError(t, tool.Execute(t.Context(), params))
	call := f.written[0].(discardDraftCall)
	assert.Equal(t, f.message.ID, call.messageID)
	assert.Equal(t, params.OrganizationID, call.tenant.OrgID)
}

func TestDraftInboundLoad_ShowsTheDraftItReadsOrSaysNovaReadsIt(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	tool := newDraftInboundLoadTool(f, f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.message.ID.String()})

	f.loadDraft = &triageworkservice.LoadDraftPlan{Message: f.message, NeedsNova: true}
	assert.True(t, previewOf(t, tool, params).Partial)

	f.loadDraft = &triageworkservice.LoadDraftPlan{
		Message: f.message,
		Draft: &inboundmessage.LoadDraft{
			Source: "tender.pdf",
			Fields: []inboundmessage.LoadDraftField{
				{Key: inboundmessage.LoadFieldOrigin, Value: "Columbus, OH"},
			},
		},
	}
	preview := previewOf(t, tool, params)
	assert.False(t, preview.Partial)
	assert.Equal(t, "Columbus, OH",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "Origin").After)
	assert.Empty(t, f.written)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, f.message.ID, f.written[0].(*triageworkservice.LoadRequest).MessageID)
}

func TestCreateInboundLoad_CreatesFromTheCheckedFieldsOnce(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	f.createLoad = &triageworkservice.CreateLoadPlan{
		Message: f.message,
		Load: &inboundmessageports.CreateTriageLoadRequest{
			Fields: []inboundmessageports.TriageLoadFieldValue{
				{Key: inboundmessage.LoadFieldOrigin, Value: "Columbus, OH"},
			},
		},
	}
	tool := newCreateInboundLoadTool(f)
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"fields": []any{
			map[string]any{"key": "Origin", "value": "Columbus, OH"},
			map[string]any{"key": "Destination", "value": "Louisville, KY"},
		},
	})

	preview := previewOf(t, tool, params)
	created := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, permission.ResourceShipment, created.Resource)
	assert.Empty(t, f.written)

	result, err := tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.NotEmpty(t, result.IDs["shipmentId"])
	req := f.written[0].(*triageworkservice.CreateLoadRequest)
	assert.Equal(t, "key-1", req.IdempotencyKey)
	assert.Equal(t, []triageworkservice.LoadFieldInput{
		{Key: inboundmessage.LoadFieldOrigin, Value: "Columbus, OH"},
		{Key: inboundmessage.LoadFieldDestination, Value: "Louisville, KY"},
	}, req.Fields)

	require.Error(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"fields":    []any{map[string]any{"key": "Color", "value": "red"}},
	})), "a field a load does not have is refused")

	f.createLoad = &triageworkservice.CreateLoadPlan{Message: f.message, AlreadyCreated: true}
	agenttoolstest.RequireWarning(t, previewOf(t, tool, params), agent.PreviewWarningWouldFail)
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Len(t, f.written, 1, "a message that made its load does not make another")
}

func TestDiscardInboundLoad_ThrowsAwayOnlyADraftThatExists(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	after := *f.message
	f.loadDiscard = &triageworkservice.LoadDiscardPlan{Message: f.message, After: &after}
	tool := newDiscardInboundLoadTool(f, f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.message.ID.String()})

	agenttoolstest.RequireWarning(t, previewOf(t, tool, params), agent.PreviewWarningWouldFail)
	require.ErrorIs(t, tool.Execute(t.Context(), params), errNoDraftedLoad)
	assert.Empty(t, f.written)

	f.message.LoadDraft = &inboundmessage.LoadDraft{Fields: []inboundmessage.LoadDraftField{
		{Key: inboundmessage.LoadFieldOrigin, Value: "Columbus, OH"},
	}}
	require.NoError(t, tool.Execute(t.Context(), params))
	_, discarded := f.written[0].(discardLoadCall)
	assert.True(t, discarded)
}

func TestFileInboundAttachment_FilesOnlyTheNamedMessagesAttachment(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	attachment := &inboundmessage.InboundAttachment{
		ID:        pulid.MustNew("imsga_"),
		MessageID: f.message.ID,
		FileName:  "pod.pdf",
	}
	f.message.Attachments = []*inboundmessage.InboundAttachment{attachment}
	shipmentID := pulid.MustNew("shp_")
	filed := *attachment
	filed.FiledShipmentID = shipmentID
	filed.FiledAs = "Proof of Delivery"
	f.filePlan = &triageworkservice.FilePlan{
		Message:      f.message,
		Attachment:   attachment,
		After:        &filed,
		DocumentType: &documenttype.DocumentType{Name: "Proof of Delivery"},
	}
	tool := newFileInboundAttachmentTool(f, f)
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId":    f.message.ID.String(),
		"attachmentId": attachment.ID.String(),
		"shipmentId":   shipmentID.String(),
	})

	preview := previewOf(t, tool, params)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	where := agenttoolstest.FieldByPath(t, change, "filedShipmentId")
	require.NotNil(t, where.AfterRef)
	assert.Equal(t, shipmentID, where.AfterRef.ID)
	assert.Empty(t, f.written)

	require.NoError(t, tool.Execute(t.Context(), params))
	req := f.written[0].(*triageworkservice.FileRequest)
	assert.Equal(t, attachment.ID, req.AttachmentID)
	assert.Equal(t, shipmentID, req.ShipmentID)
	assert.Empty(t, req.Pages)

	other := agenttoolstest.DeskParams(map[string]any{
		"messageId":    f.message.ID.String(),
		"attachmentId": pulid.MustNew("imsga_").String(),
		"shipmentId":   shipmentID.String(),
	})
	require.Error(t, tool.Execute(t.Context(), other),
		"an attachment of another message is refused")
	assert.Len(t, f.written, 1)
}

func TestFileInboundAttachment_ReadsEachPageRange(t *testing.T) {
	t.Parallel()

	documentType := pulid.MustNew("dt_")
	call, err := fileRequest(&agentports.ToolExecuteParams{
		Actor: agenttoolstest.DeskParams(nil).Actor,
		Params: map[string]any{
			"messageId":    pulid.MustNew("imsg_").String(),
			"attachmentId": pulid.MustNew("imsga_").String(),
			"shipmentId":   pulid.MustNew("shp_").String(),
			"pages": []any{
				map[string]any{"fromPage": 1, "toPage": 2, "documentTypeId": documentType.String()},
				map[string]any{"fromPage": 3, "toPage": 3, "documentTypeId": documentType.String()},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []triageworkservice.PageRange{
		{FromPage: 1, ToPage: 2, DocumentTypeID: documentType},
		{FromPage: 3, ToPage: 3, DocumentTypeID: documentType},
	}, call.file.Pages)

	_, err = fileRequest(&agentports.ToolExecuteParams{
		Actor: agenttoolstest.DeskParams(nil).Actor,
		Params: map[string]any{
			"messageId":    pulid.MustNew("imsg_").String(),
			"attachmentId": pulid.MustNew("imsga_").String(),
			"shipmentId":   pulid.MustNew("shp_").String(),
			"pages": []any{
				map[string]any{"fromPage": 3, "toPage": 2, "documentTypeId": documentType.String()},
			},
		},
	})
	require.Error(t, err, "a range must run first page before last")
}

func TestUnfileInboundAttachment_SaysWhatASplitLoses(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	attachment := &inboundmessage.InboundAttachment{
		ID:              pulid.MustNew("imsga_"),
		MessageID:       f.message.ID,
		FileName:        "stack.pdf",
		FiledShipmentID: pulid.MustNew("shp_"),
		FiledAs:         "2 documents (Bill of Lading)",
	}
	f.message.Attachments = []*inboundmessage.InboundAttachment{attachment}
	unfiled := *attachment
	unfiled.FiledShipmentID = pulid.Nil
	unfiled.FiledAs = ""
	f.unfilePlan = &triageworkservice.UnfilePlan{
		Message:    f.message,
		Attachment: attachment,
		After:      &unfiled,
		Removed: []inboundmessage.SplitDocument{
			{
				FromPage: 1, ToPage: 2,
				DocumentID: pulid.MustNew("doc_"), DocumentType: "Bill of Lading",
			},
			{
				FromPage: 3, ToPage: 3,
				DocumentID: pulid.MustNew("doc_"), DocumentType: "Bill of Lading",
			},
		},
	}
	tool := newUnfileInboundAttachmentTool(f, f)
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId":    f.message.ID.String(),
		"attachmentId": attachment.ID.String(),
	})

	preview := previewOf(t, tool, params)
	require.Len(t, preview.Changes, 3)
	assert.Equal(t, agent.PreviewOperationDelete, preview.Changes[1].Operation)
	assert.Equal(t, agent.PreviewOperationDelete, preview.Changes[2].Operation)
	assert.Contains(t, preview.Summary, "deleting the 2 documents")
	assert.Empty(t, f.written)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, attachment.ID, f.written[0].(*triageworkservice.UnfileRequest).AttachmentID)
}

func TestLogInboundException_HoldsTheShipmentWithTheCountGiven(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	after := *f.message
	after.ActionState.ExceptionKind = inboundmessage.ExceptionShortage
	f.exceptionPlan = &triageworkservice.ExceptionPlan{
		Message: f.message,
		After:   &after,
		Hold: &inboundmessageports.LogTriageExceptionRequest{
			Kind:     inboundmessage.ExceptionShortage,
			Quantity: 2,
			Unit:     "pallets",
			Source:   "From an email from Dana@Shipper.example: Where is 88213?",
		},
		Summary: "2 pallets short",
	}
	tool := newLogInboundExceptionTool(f, f)
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"kind":      "Shortage",
		"quantity":  2,
		"unit":      "pallets",
	})

	preview := previewOf(t, tool, params)
	hold := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceShipmentHold, hold.Resource)
	assert.Equal(t, agent.PreviewOperationCreate, hold.Operation)
	assert.Empty(t, f.written)

	require.NoError(t, tool.Execute(t.Context(), params))
	req := f.written[0].(*triageworkservice.LogExceptionRequest)
	assert.Equal(t, inboundmessage.ExceptionShortage, req.Kind)
	assert.Equal(t, 2, req.Quantity)
	assert.Equal(t, "pallets", req.Unit)

	id := f.message.ID.String()
	for name, bad := range map[string]map[string]any{
		"no unit":      {"messageId": id, "kind": "Damage", "quantity": 1},
		"zero":         {"messageId": id, "kind": "Damage", "quantity": 0, "unit": "case"},
		"unknown kind": {"messageId": id, "kind": "Lost", "quantity": 1, "unit": "case"},
	} {
		assert.Errorf(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(bad)), name)
	}
	assert.Len(t, f.written, 1)
}

/*
Settling a disputed detention charge is a person's decision: the tool only
ever proposes it, and its preview shows the billed amount a waiver gives up.
*/
func TestSettleDetentionDispute_ShowsTheMoneyAWaiverGivesUp(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	f.message.Classification = inboundmessage.ClassificationDetentionDispute
	after := *f.message
	after.ActionState.StopsDecision = inboundmessage.StopsDecisionWaive
	f.stopsPlan = &triageworkservice.StopsPlan{Message: f.message, After: &after}
	amount := "112.50"
	f.stopTimes = &inboundmessageports.StopTimes{
		OccurrenceID:           pulid.MustNew("dto_"),
		Currency:               "USD",
		StopName:               "Louisville Store",
		BilledDetentionMinutes: 90,
		BilledDetentionAmount:  &amount,
	}
	tool := newSettleDetentionDisputeTool(f, f)
	waive := agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"decision":  "Waive",
	})

	preview := previewOf(t, tool, waive)
	require.Len(t, preview.Changes, 2)
	money := preview.Changes[1].Money
	require.NotNil(t, money, "a waiver shows the charge it gives up")
	assert.Equal(t, "USD", money.Currency)
	assert.Contains(t, preview.Summary, "90 billed minutes")
	assert.Empty(t, f.written)

	require.NoError(t, tool.Execute(t.Context(), waive))
	req := f.written[0].(*triageworkservice.DecideStopsRequest)
	assert.Equal(t, inboundmessage.StopsDecisionWaive, req.Decision)
	assert.Same(t, waive.Actor, req.Actor, "the detention is settled as the caller")

	stand := agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"decision":  "StandBy",
	})
	after.ActionState.StopsDecision = inboundmessage.StopsDecisionStandBy
	assert.Len(t, previewOf(t, tool, stand).Changes, 1, "standing by gives nothing up")

	f.stopsPlan = &triageworkservice.StopsPlan{
		Message:   f.message,
		After:     f.message,
		Unchanged: true,
	}
	agenttoolstest.RequireWarning(t, previewOf(t, tool, stand), agent.PreviewWarningWouldFail)
}

func TestResolveInboundConflict_SettlesTheConflictNamed(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	before := inboundmessage.Conflict{
		ID:   "dup:imsga_1",
		Kind: inboundmessage.ConflictDuplicateDocument,
	}
	settled := before
	settled.Resolution = inboundmessage.ConflictReplaceExisting
	f.conflictPlan = &triageconflictservice.ConflictPlan{
		Message: f.message,
		Before:  before,
		After:   settled,
	}
	tool := newResolveInboundConflictTool(fakeConflictDesk{f})
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId":  f.message.ID.String(),
		"conflictId": "dup:imsga_1",
		"resolution": "ReplaceExisting",
	})

	preview := previewOf(t, tool, params)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "ReplaceExisting", agenttoolstest.FieldByPath(t, change, "resolution").After)
	assert.Empty(t, f.written)

	require.NoError(t, tool.Execute(t.Context(), params))
	req := f.written[0].(*triageconflictservice.ResolveRequest)
	assert.Equal(t, "dup:imsga_1", req.ConflictID)
	assert.Equal(t, inboundmessage.ConflictReplaceExisting, req.Resolution)
	assert.Equal(t, params.Actor.UserID, req.ActorID)
}

func TestReopenInboundConflict_ReopensTheConflictNamed(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	before := inboundmessage.Conflict{
		ID:         "load:0",
		Kind:       inboundmessage.ConflictOverlappingLoad,
		Resolution: inboundmessage.ConflictSkipDuplicate,
	}
	reopened := before
	reopened.Resolution = ""
	f.conflictPlan = &triageconflictservice.ConflictPlan{
		Message: f.message,
		Before:  before,
		After:   reopened,
	}
	tool := newReopenInboundConflictTool(f, fakeConflictDesk{f})
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId":  f.message.ID.String(),
		"conflictId": "load:0",
	})

	preview := previewOf(t, tool, params)
	assert.Contains(t, preview.Summary, "repeated load")
	assert.Empty(t, f.written)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "load:0", f.written[0].(*triageconflictservice.ReopenRequest).ConflictID)
}
