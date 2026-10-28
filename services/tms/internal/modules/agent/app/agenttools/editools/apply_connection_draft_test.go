package editools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/edispec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const draftDigest = "sha256:5f1c"

type fakeConnectionDrafts struct {
	plan     *ediports.ConnectionDraftPlan
	guard    *agenttoolstest.WriteGuard
	planned  *ediports.ConnectionDraftRequest
	applied  *ediports.ApplyConnectionDraftRequest
	result   *ediports.AppliedConnectionDraft
	previews int
}

func (f *fakeConnectionDrafts) PreviewConnectionDraft(
	_ context.Context,
	req *ediports.ConnectionDraftRequest,
) (*ediports.ConnectionDraftPlan, error) {
	f.previews++
	f.planned = req
	return f.plan, nil
}

func (f *fakeConnectionDrafts) ApplyConnectionDraft(
	_ context.Context,
	req *ediports.ApplyConnectionDraftRequest,
) (*ediports.AppliedConnectionDraft, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.applied = req
	if f.result != nil {
		return f.result, nil
	}
	return &ediports.AppliedConnectionDraft{}, nil
}

type fakeSpecReader struct {
	read *edispec.ReadRequest
}

func (f *fakeSpecReader) Read(
	_ context.Context,
	req *edispec.ReadRequest,
) (*ediports.SpecDocument, error) {
	f.read = req
	return &ediports.SpecDocument{
		ID:       req.DocumentID,
		FileName: "acme-guide.pdf",
		Pages:    []ediports.SpecPage{{Number: 1, Text: "Trading Partner Code: ACMERETAIL"}},
	}, nil
}

func readyPlan() *ediports.ConnectionDraftPlan {
	existingID := pulid.MustNew("edip_")
	return &ediports.ConnectionDraftPlan{
		Draft: &ediports.ConnectionDraft{
			FileName:       "acme-guide.pdf",
			PageCount:      7,
			DocumentDigest: draftDigest,
			Ready:          true,
			Changes:        3,
			Partner: ediports.DraftPartner{
				Match:  ediports.PartnerMatchExisting,
				Fields: []ediports.DraftValue{{Field: "partner.name", Value: "Acme Retail Group"}},
			},
			Connection: ediports.DraftConnection{Sets: []ediports.DraftSetDirection{
				{TransactionSet: edi.TransactionSet204, Direction: edi.DocumentDirectionInbound},
			}},
			Transport: &ediports.DraftTransport{Credentials: []ediports.CredentialRequirement{
				{Label: "Partner signing certificate", Missing: true},
				{
					Label:    "AS2 private key for your signing certificate",
					Required: true,
					Missing:  true,
				},
			}},
			Sets: []ediports.DraftSet{{
				TransactionSet: edi.TransactionSet204,
				Direction:      edi.DocumentDirectionInbound,
				Unmet:          []ediports.DraftRequirementCheck{{SegmentID: "MS1"}},
			}},
			Gaps: []ediports.DraftGap{{Summary: "Appendix B is missing"}},
		},
		Partner: ediports.PlannedPartner{
			Before: &edi.EDIPartner{
				ID:          existingID,
				Name:        "Acme Retail Group",
				ContactName: "Lee Park",
				Version:     4,
			},
			After: &edi.EDIPartner{
				ID: existingID, Name: "Acme Retail Group", ContactName: "Dana Whitfield", Version: 4,
			},
			Change: ediports.ChangeKindChange,
		},
		Project: &ediports.PlannedProject{
			Files: []lading.File{
				{Path: "partner.lading", Source: "partner acme \"Acme Retail Group\"\n"},
				{
					Path:   "guides/204.lading",
					Source: "guide acme204 from x12.004010.204 \"Acme 204\"\n",
				},
			},
			Samples: []lading.Sample{{Path: "samples/spec-page-7.x12", Content: "ISA*00~"}},
			Change:  ediports.ChangeKindAdd,
		},
		Transport: &ediports.PlannedTransport{
			After: &edi.EDICommunicationProfile{
				Name:   "Acme AS2",
				Status: domaintypes.StatusInactive,
			},
			Change: ediports.ChangeKindSame,
		},
		ControlNumbers: []ediports.PlannedControlNumber{{
			After: &edi.EDIControlNumberSequence{
				Kind:      edi.ControlNumberKindInterchange,
				NextValue: 1000,
			},
			Change: ediports.ChangeKindAdd,
		}},
	}
}

func draftArgs(documentID pulid.ID) map[string]any {
	return map[string]any{
		edispec.ParamDocumentID:     documentID.String(),
		edispec.ParamDocumentDigest: draftDigest,
		edispec.ParamValues: []any{map[string]any{
			"field":      "partner.code",
			"value":      "ACMERETAIL",
			"confidence": 0.95,
			"citation":   map[string]any{"page": 1, "quote": "Trading Partner Code: ACMERETAIL"},
		}},
	}
}

func onThread(
	params agenttoolports.ToolExecuteParams,
	thread pulid.ID,
) agenttoolports.ToolExecuteParams {
	params.ThreadID = thread
	return params
}

func TestApplyConnectionDraft_PreviewsEveryRecordItWouldWrite(t *testing.T) {
	t.Parallel()

	drafts := &fakeConnectionDrafts{plan: readyPlan(), guard: &agenttoolstest.WriteGuard{}}
	reader := &fakeSpecReader{}
	tool := NewApplyConnectionDraftTool(drafts, reader).(*ApplyConnectionDraftTool)
	thread := pulid.MustNew("ath_")
	documentID := pulid.MustNew("doc_")

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		drafts.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(
				t.Context(),
				onThread(agenttoolstest.AgentParamsFor(draftArgs(documentID)), thread),
			)
		},
	)

	require.NotNil(t, reader.read)
	assert.Equal(t, thread, reader.read.ThreadID)
	assert.True(t, reader.read.AttachedBy.IsNil(), "an approver need not be the uploader")
	require.NotNil(t, drafts.planned)
	assert.Equal(t, "acme-guide.pdf", drafts.planned.Document.FileName)
	require.Len(t, drafts.planned.Values, 1)
	assert.Equal(t, 1, drafts.planned.Values[0].Citation.Page)

	require.Len(t, preview.Changes, 3, "the unchanged transport is left out")
	partner := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationUpdate, partner.Operation)
	require.NotNil(t, partner.Version)
	assert.Equal(t, int64(4), *partner.Version)
	contact := agenttoolstest.FieldByPath(t, partner, "contactName")
	assert.Equal(t, "Lee Park", contact.Before)
	assert.Equal(t, "Dana Whitfield", contact.After)
	assert.Equal(
		t,
		agent.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 1).Operation,
	)
	assert.Equal(
		t,
		agent.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 2).Operation,
	)

	assert.Contains(t, preview.Summary, "Would update the EDI connection with Acme Retail Group")
	assert.Contains(t, preview.Summary, draftDigest)
	assert.Contains(t, preview.Summary, "enter: AS2 private key for your signing certificate.")
	assert.NotContains(t, preview.Summary, "Partner signing certificate", "only what is required")
	assert.Contains(t, preview.Summary, "leave 1 of the guide's mandatory elements unwritten")
	assert.Empty(t, preview.Warnings)
}

func TestApplyConnectionDraft_WarnsWhenTheDraftCannotBeApplied(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		change   func(*ediports.ConnectionDraftPlan)
		contains string
	}{
		"a guide read differently": {
			change:   func(plan *ediports.ConnectionDraftPlan) { plan.Draft.DocumentDigest = "sha256:0000" },
			contains: "reads differently",
		},
		"a decision left open": {
			change: func(plan *ediports.ConnectionDraftPlan) {
				plan.Draft.Ready = false
				plan.Draft.Blockers = []string{"Decide time zone"}
			},
			contains: "Decide time zone",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			plan := readyPlan()
			tc.change(plan)
			drafts := &fakeConnectionDrafts{plan: plan, guard: &agenttoolstest.WriteGuard{}}
			tool := NewApplyConnectionDraftTool(drafts, &fakeSpecReader{}).(*ApplyConnectionDraftTool)
			params := onThread(
				agenttoolstest.AgentParamsFor(draftArgs(pulid.MustNew("doc_"))),
				pulid.MustNew("ath_"),
			)

			preview, err := tool.Preview(t.Context(), params)
			require.NoError(t, err)
			require.NotEmpty(t, preview.Warnings)
			assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
			assert.Contains(t, preview.Warnings[0].Message, tc.contains)

			err = tool.Validate(t.Context(), params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.contains)
		})
	}
}

func TestApplyConnectionDraft_WritesOnlyWhatAPersonApproved(t *testing.T) {
	t.Parallel()

	drafts := &fakeConnectionDrafts{plan: readyPlan(), guard: &agenttoolstest.WriteGuard{}}
	tool := NewApplyConnectionDraftTool(drafts, &fakeSpecReader{})
	thread := pulid.MustNew("ath_")
	args := draftArgs(pulid.MustNew("doc_"))

	require.ErrorIs(t,
		tool.Execute(t.Context(), onThread(agenttoolstest.AgentParamsFor(args), thread)),
		ErrEDIDecisionNeedsAPerson)
	assert.Nil(t, drafts.applied)

	approved := onThread(agenttoolstest.ApprovedParams(args), thread)
	require.NoError(t, tool.Execute(t.Context(), approved))
	require.NotNil(t, drafts.applied)
	assert.Equal(t, draftDigest, drafts.applied.ExpectedDocumentDigest)
	assert.Equal(t, approved.Actor, drafts.applied.Actor)
	assert.Equal(t, "acme-guide.pdf", drafts.applied.Draft.Document.FileName)
}

func TestApplyConnectionDraft_NeedsTheDigestAndAConversation(t *testing.T) {
	t.Parallel()

	tool := NewApplyConnectionDraftTool(&fakeConnectionDrafts{plan: readyPlan()}, &fakeSpecReader{})
	validator := tool.(agenttoolports.ToolValidator)
	args := draftArgs(pulid.MustNew("doc_"))
	delete(args, edispec.ParamDocumentDigest)

	err := validator.Validate(
		t.Context(),
		onThread(agenttoolstest.AgentParamsFor(args), pulid.MustNew("ath_")),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), edispec.ParamDocumentDigest)
}

func TestApplyConnectionDraft_IsAlwaysAPersonsDecision(t *testing.T) {
	t.Parallel()

	policy := NewApplyConnectionDraftTool(nil, nil).Policy()
	assert.Equal(t, agent.ToolKindAction, policy.Kind)
	assert.Equal(t, permission.ResourceEDI, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, agent.ExternalReadAlways, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceAttachment, policy.Source)
}
