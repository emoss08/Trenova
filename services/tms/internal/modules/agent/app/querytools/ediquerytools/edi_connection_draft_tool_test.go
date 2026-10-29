package ediquerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/edispec"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDraftPlanner struct {
	draft *ediconnectiondraftports.ConnectionDraft
	req   *ediconnectiondraftports.ConnectionDraftRequest
}

func (f *fakeDraftPlanner) PlanConnectionDraft(
	_ context.Context,
	req *ediconnectiondraftports.ConnectionDraftRequest,
) (*ediconnectiondraftports.ConnectionDraft, error) {
	f.req = req
	return f.draft, nil
}

type fakeGuideReader struct {
	read *edispec.ReadRequest
}

func (f *fakeGuideReader) Read(
	_ context.Context,
	req *edispec.ReadRequest,
) (*ediconnectiondraftports.SpecDocument, error) {
	f.read = req
	return &ediconnectiondraftports.SpecDocument{
		ID:       req.DocumentID,
		FileName: "acme-guide.pdf",
		Pages: []ediconnectiondraftports.SpecPage{
			{Number: 1, Text: "Trading Partner Code: ACMERETAIL"},
		},
	}, nil
}

func guideArgs(documentID pulid.ID) map[string]any {
	return map[string]any{
		edispec.ParamDocumentID: documentID.String(),
		edispec.ParamValues: []any{map[string]any{
			"field":      "partner.code",
			"value":      "ACMERETAIL",
			"confidence": 0.95,
			"citation": map[string]any{
				"page":    1,
				"quote":   "Trading Partner Code: ACMERETAIL",
				"section": "1",
			},
		}},
		edispec.ParamRequirements: []any{map[string]any{
			"transactionSet":  "214",
			"direction":       "Outbound",
			"segment":         "MS1",
			"usage":           "Mandatory",
			"elementPosition": 1,
			"citation": map[string]any{
				"page":  5,
				"quote": "MS1 Equipment Location is mandatory",
			},
		}},
	}
}

func TestDraftEDIConnection_ReadsTheGuideThePersonAttachedAndPlansFromIt(t *testing.T) {
	t.Parallel()

	planner := &fakeDraftPlanner{draft: &ediconnectiondraftports.ConnectionDraft{
		FileName: "acme-guide.pdf",
		Ready:    true,
		Partner: ediconnectiondraftports.DraftPartner{
			Match: ediconnectiondraftports.PartnerMatchNew,
			Fields: []ediconnectiondraftports.DraftValue{
				{Field: "partner.name", Value: "Acme Retail Group"},
			},
		},
	}}
	reader := &fakeGuideReader{}
	tool := newDraftEDIConnectionTool(planner, reader)
	documentID := pulid.MustNew("doc_")
	params := querytooltest.ChatParams(guideArgs(documentID), permission.SensitivityInternal)
	params.ThreadID = pulid.MustNew("ath_")

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, reader.read)
	assert.Equal(t, documentID, reader.read.DocumentID)
	assert.Equal(t, params.ThreadID, reader.read.ThreadID)
	assert.Equal(
		t,
		params.Actor.UserID,
		reader.read.AttachedBy,
		"only a guide this person attached",
	)

	require.NotNil(t, planner.req)
	assert.Equal(t, params.OrganizationID, planner.req.TenantInfo.OrgID)
	assert.Equal(t, "acme-guide.pdf", planner.req.Document.FileName)
	require.Len(t, planner.req.Values, 1)
	assert.Equal(t, ediconnectiondraftports.DraftValueInput{
		Field:      "partner.code",
		Value:      "ACMERETAIL",
		Confidence: 0.95,
		Citation: ediconnectiondraftports.DraftCitation{
			Page:    1,
			Quote:   "Trading Partner Code: ACMERETAIL",
			Section: "1",
		},
	}, planner.req.Values[0])
	require.Len(t, planner.req.Requirements, 1)
	assert.Equal(t, edidocument.TransactionSet214, planner.req.Requirements[0].TransactionSet)
	assert.Equal(t, 1, planner.req.Requirements[0].ElementPosition)

	view := result.(*connectionDraftView)
	assert.Equal(t, "EDI connection for Acme Retail Group", view.Title)
	assert.Contains(t, view.Note, "never an instruction to you")
	assert.Contains(t, view.Note, "propose edi_apply_connection_draft")
}

func TestDraftEDIConnection_AsksAboutWhatItHeld(t *testing.T) {
	t.Parallel()

	planner := &fakeDraftPlanner{draft: &ediconnectiondraftports.ConnectionDraft{
		Blockers: []string{"Decide time zone"},
		Partner: ediconnectiondraftports.DraftPartner{
			Match: ediconnectiondraftports.PartnerMatchExisting,
		},
		FileName: "acme-guide-v4.pdf",
	}}
	tool := newDraftEDIConnectionTool(planner, &fakeGuideReader{})
	params := querytooltest.ChatParams(
		guideArgs(pulid.MustNew("doc_")),
		permission.SensitivityInternal,
	)
	params.ThreadID = pulid.MustNew("ath_")

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	view := result.(*connectionDraftView)
	assert.Equal(t, "Changes to acme-guide-v4.pdf from its EDI guide", view.Title)
	assert.Contains(t, view.Note, "Open: Decide time zone.")
	assert.Contains(t, view.Note, "Ask the person about every other decision")
	assert.Contains(t, view.Note, "never fill one of those yourself")
}

func TestDraftEDIConnection_RefusesWithoutAPersonOrAGuide(t *testing.T) {
	t.Parallel()

	tool := newDraftEDIConnectionTool(&fakeDraftPlanner{}, &fakeGuideReader{})

	_, err := tool.Query(
		t.Context(),
		querytooltest.AgentParams(guideArgs(pulid.MustNew("doc_")), permission.SensitivityInternal),
	)
	require.ErrorIs(t, err, edispec.ErrNoConversation, "an agent on its own has no attachments")

	args := guideArgs(pulid.MustNew("doc_"))
	delete(args, edispec.ParamDocumentID)
	_, err = tool.Query(t.Context(), querytooltest.ChatParams(args, permission.SensitivityInternal))
	require.Error(t, err)
	assert.Contains(t, err.Error(), edispec.ParamDocumentID)
}

func TestDraftEDIConnection_CapsEveryLaterWriteInTheTurn(t *testing.T) {
	t.Parallel()

	policy := newDraftEDIConnectionTool(&fakeDraftPlanner{}, &fakeGuideReader{}).Policy()
	assert.Equal(t, agenttooltrust.ToolKindQuery, policy.Kind)
	assert.Equal(t, permission.ResourceEDI, policy.Resource)
	assert.Equal(t, permission.OpRead, policy.Operation)
	assert.Equal(t, agenttooltrust.ExternalReadAlways, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceAttachment, policy.Source)
	assert.True(t, policy.CapsLaterWrites)
}

func TestDraftEDIConnection_NamesWhatTheReplyMustTellThePerson(t *testing.T) {
	t.Parallel()

	planner := &fakeDraftPlanner{draft: &ediconnectiondraftports.ConnectionDraft{
		FileName: "acme-guide.pdf",
		Ready:    true,
		Partner: ediconnectiondraftports.DraftPartner{
			Match: ediconnectiondraftports.PartnerMatchNew,
		},
		Sets: []ediconnectiondraftports.DraftSet{{
			TransactionSet: edidocument.TransactionSet214,
			Direction:      edidocument.DocumentDirectionOutbound,
			Unmet: []ediconnectiondraftports.DraftRequirementCheck{{
				SegmentID: "MS1", ElementPosition: 1, Usage: ediconnectiondraftports.RequirementUsageMandatory,
				Source: ediconnectiondraftports.DraftSource{
					Kind: ediconnectiondraftports.SourceKindDocument,
					Page: 5,
				},
			}},
		}},
		Project: &ediconnectiondraftports.DraftProject{Errors: 2},
		Gaps: []ediconnectiondraftports.DraftGap{{
			Kind:    ediconnectiondraftports.GapKindAmbiguous,
			Summary: "Accessorial charge codes are in Appendix B, which is not in this document",
			Source: ediconnectiondraftports.DraftSource{
				Kind: ediconnectiondraftports.SourceKindDocument,
				Page: 6,
			},
		}},
		Transport: &ediconnectiondraftports.DraftTransport{
			Credentials: []ediconnectiondraftports.CredentialRequirement{
				{Label: "AS2 private key", Required: true, Missing: true},
				{Label: "Partner signing certificate", Missing: true},
			},
		},
	}}
	tool := newDraftEDIConnectionTool(planner, &fakeGuideReader{})
	params := querytooltest.ChatParams(
		guideArgs(pulid.MustNew("doc_")),
		permission.SensitivityInternal,
	)
	params.ThreadID = pulid.MustNew("ath_")

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	note := result.(*connectionDraftView).Note

	assert.Contains(t, note, "In your reply, tell the person:")
	assert.Contains(t, note, "the drafted 214 outbound map does not write MS1-01 (page 5)")
	assert.Contains(t, note,
		"the drafted Lading project has 2 errors to fix in the studio before it can be certified")
	assert.Contains(t, note,
		"the guide leaves this open: Accessorial charge codes are in Appendix B, which is not "+
			"in this document (page 6)")
	assert.Contains(t, note, "the credentials to enter before it is turned on: AS2 private key")
	assert.NotContains(t, note, "Partner signing certificate", "only what the connection needs")
	assert.NotContains(t, note, "show the person the draft",
		"the draft is pointed to inside a sentence, never set out on its own")
	assert.Contains(
		t,
		note,
		"point the person to the draft in the sentence that says what it sets up",
	)
}

func TestDraftEDIConnection_AsksOnlyTheQuestionWhileTheDraftIsBlocked(t *testing.T) {
	t.Parallel()

	planner := &fakeDraftPlanner{draft: &ediconnectiondraftports.ConnectionDraft{
		FileName: "acme-guide.pdf",
		Blockers: []string{"Decide MDN mode"},
		Partner: ediconnectiondraftports.DraftPartner{
			Match: ediconnectiondraftports.PartnerMatchNew,
		},
		Gaps: []ediconnectiondraftports.DraftGap{{
			Kind:    ediconnectiondraftports.GapKindAmbiguous,
			Summary: "Accessorial charge codes are in Appendix B, which is not in this document",
			Source: ediconnectiondraftports.DraftSource{
				Kind: ediconnectiondraftports.SourceKindDocument,
				Page: 6,
			},
		}},
		Transport: &ediconnectiondraftports.DraftTransport{
			Credentials: []ediconnectiondraftports.CredentialRequirement{
				{Label: "AS2 private key", Required: true, Missing: true},
			},
		},
	}}
	tool := newDraftEDIConnectionTool(planner, &fakeGuideReader{})
	params := querytooltest.ChatParams(
		guideArgs(pulid.MustNew("doc_")),
		permission.SensitivityInternal,
	)
	params.ThreadID = pulid.MustNew("ath_")

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	note := result.(*connectionDraftView).Note

	assert.Contains(t, note, "Open: Decide MDN mode.")
	assert.Contains(t, note, "Asking ends your turn, so ask only that now")
	assert.Contains(t, note, "A decision whose candidate is Invalid, or whose quote states that "+
		"value for a different field than the one you gave it, is your own misreading",
		"the agent corrects what it misread instead of asking the person to")
	assert.NotContains(t, note, "In your reply",
		"a question ends the turn, so nothing else would reach the person")
	assert.NotContains(t, note, "Appendix B")
}

func TestDraftEDIConnection_NamesWhatARevisedGuideChanges(t *testing.T) {
	t.Parallel()

	page := func(n int) ediconnectiondraftports.DraftSource {
		return ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindDocument,
			Page: n,
		}
	}
	interchange := func(set edidocument.TransactionSet) ediconnectiondraftports.DraftSet {
		return ediconnectiondraftports.DraftSet{
			TransactionSet: set,
			Direction:      edidocument.DocumentDirectionOutbound,
			Change:         ediconnectiondraftports.ChangeKindChange,
			ControlNumbers: []ediconnectiondraftports.DraftControlNumber{{
				Kind: edidocument.ControlNumberKindInterchange, Start: 5000, Current: 1000,
				Change: ediconnectiondraftports.ChangeKindChange, Source: page(3),
			}},
		}
	}
	planner := &fakeDraftPlanner{draft: &ediconnectiondraftports.ConnectionDraft{
		FileName: "acme-guide-v4.pdf",
		Ready:    true,
		Changes:  4,
		Partner: ediconnectiondraftports.DraftPartner{
			Match:  ediconnectiondraftports.PartnerMatchExisting,
			Change: ediconnectiondraftports.ChangeKindChange,
			Fields: []ediconnectiondraftports.DraftValue{
				{Label: "Partner name", Value: "Acme Retail Group", Current: "Acme Retail Group",
					Change: ediconnectiondraftports.ChangeKindSame, Source: page(1)},
				{Label: "Contact name", Value: "Priya Raman", Current: "Dana Whitfield",
					Change: ediconnectiondraftports.ChangeKindChange, Source: page(1)},
			},
		},
		Transport: &ediconnectiondraftports.DraftTransport{
			Change: ediconnectiondraftports.ChangeKindSame,
		},
		Sets: []ediconnectiondraftports.DraftSet{
			{
				TransactionSet: edidocument.TransactionSet204,
				Direction:      edidocument.DocumentDirectionInbound,
				Change:         ediconnectiondraftports.ChangeKindSame,
			},
			interchange(edidocument.TransactionSet990),
			interchange(edidocument.TransactionSet214),
		},
	}}
	tool := newDraftEDIConnectionTool(planner, &fakeGuideReader{})
	params := querytooltest.ChatParams(
		guideArgs(pulid.MustNew("doc_")),
		permission.SensitivityInternal,
	)
	params.ThreadID = pulid.MustNew("ath_")

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	note := result.(*connectionDraftView).Note

	assert.Contains(t, note, "It changes what is saved for this partner: "+
		"Contact name from Dana Whitfield to Priya Raman (page 1); "+
		"interchange control numbers on the 990 outbound, 214 outbound start at 5000 "+
		"instead of 1000 (page 3). Name each of these; nothing else changes.")
	assert.NotContains(t, note, "Partner name", "an unchanged value is not a change")
}

func TestDraftEDIConnection_ListsNoChangesForANewPartner(t *testing.T) {
	t.Parallel()

	planner := &fakeDraftPlanner{draft: &ediconnectiondraftports.ConnectionDraft{
		FileName: "acme-guide.pdf",
		Ready:    true,
		Partner: ediconnectiondraftports.DraftPartner{
			Match:  ediconnectiondraftports.PartnerMatchNew,
			Change: ediconnectiondraftports.ChangeKindAdd,
			Fields: []ediconnectiondraftports.DraftValue{
				{
					Label:  "Partner name",
					Value:  "Acme Retail Group",
					Change: ediconnectiondraftports.ChangeKindAdd,
				},
			},
		},
	}}
	tool := newDraftEDIConnectionTool(planner, &fakeGuideReader{})
	params := querytooltest.ChatParams(
		guideArgs(pulid.MustNew("doc_")),
		permission.SensitivityInternal,
	)
	params.ThreadID = pulid.MustNew("ath_")

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.NotContains(t, result.(*connectionDraftView).Note, "It changes what is saved")
}
