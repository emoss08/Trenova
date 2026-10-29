package assistantthreads

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type pageFixture struct {
	svc           *Service
	provisioner   *assistanttest.Provisioner
	threads       *assistanttest.PageThreads
	conversations *assistanttest.CreatingConversations
	permissions   *assistanttest.SubjectPermissions
	actor         *permissionports.RequestActor
}

func newPageFixture(allowed ...string) *pageFixture {
	permissions := &assistanttest.SubjectPermissions{
		Allowed: map[string]bool{"assistant:create": true},
	}
	for _, key := range allowed {
		permissions.Allowed[key] = true
	}
	fixture := &pageFixture{
		provisioner: &assistanttest.Provisioner{
			Definitions: map[string]*agentdefinition.Definition{},
		},
		threads: &assistanttest.PageThreads{
			Live: map[conversationports.PageThreadKey]*conversation.Thread{},
		},
		conversations: &assistanttest.CreatingConversations{},
		permissions:   permissions,
		actor:         assistanttest.Actor(),
	}
	fixture.svc = &Service{
		logger:        zap.NewNop(),
		conversations: fixture.conversations,
		permissions:   permissions,
		systemAgents:  fixture.provisioner,
		pageThreads:   fixture.threads,
	}

	return fixture
}

func (f *pageFixture) open(
	t *testing.T,
	origin conversation.ThreadOrigin,
	subjectType agent.SubjectType,
	subjectID pulid.ID,
) (*conversation.Thread, error) {
	t.Helper()

	opened, err := f.svc.OpenPageThread(t.Context(), &assistantports.OpenPageThreadRequest{
		TenantInfo:  f.actor.TenantInfo(),
		Origin:      origin,
		SubjectType: subjectType,
		SubjectID:   subjectID,
	}, f.actor)
	if err != nil {
		return nil, err
	}

	return opened.Thread, nil
}

func TestOpenPageThread_NamesTheAgentThePageTalksTo(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("formula_template:read")
	opened, err := fixture.svc.OpenPageThread(t.Context(), &assistantports.OpenPageThreadRequest{
		TenantInfo: fixture.actor.TenantInfo(),
		Origin:     conversation.ThreadOriginFormula,
	}, fixture.actor)
	require.NoError(t, err)

	assert.Equal(t, opened.Thread.AgentDefinitionID, opened.Agent.ID)
	assert.Equal(t, agentdefinition.SystemKeyFormulaAssistant, opened.Agent.SystemKey)
	assert.Equal(t, agenttemplate.TemplateFormulaAssistant, opened.Agent.Template)
	assert.NotEmpty(t, opened.Agent.Name)
	assert.NotEmpty(t, opened.Agent.ToolNames)
	assert.NotEmpty(t, opened.Agent.Starters)
}

func TestOpenPageThread_OpensOneHiddenTaintedConversationPerDocument(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("document:read")
	documentID := pulid.MustNew("doc_")

	first, err := fixture.open(
		t,
		conversation.ThreadOriginImport,
		agent.SubjectDocument,
		documentID,
	)
	require.NoError(t, err)
	second, err := fixture.open(t, conversation.ThreadOriginImport, "", documentID)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "the same person and document land in one conversation")
	require.Len(t, fixture.threads.Claimed, 1)
	assert.Nil(t, fixture.conversations.Created, "a document's conversation is claimed on its key")

	assert.Equal(t, conversation.ThreadOriginImport, first.Origin)
	assert.False(t, first.Origin.Listed())
	assert.Equal(t, agent.SubjectDocument, first.SubjectType)
	assert.Equal(t, documentID, first.SubjectID)
	assert.Equal(t, fixture.actor.UserID, first.UserID)
	assert.True(t, first.CanContinue)
	require.True(t, first.Tainted(), "a document someone outside wrote taints its conversation")
	require.Len(t, first.Taint.Marks, 1)
	assert.Equal(t, agent.TaintSourceDocument, first.Taint.Marks[0].Source)
	assert.Equal(t, documentID.String(), first.Taint.Marks[0].Ref.ID)

	require.NotEmpty(t, fixture.provisioner.Asked)
	assert.Equal(
		t,
		agentdefinition.SystemKeyImportAssistant,
		fixture.provisioner.Asked[0].SystemKey,
	)
	assert.Equal(t,
		fixture.provisioner.Definitions[agentdefinition.SystemKeyImportAssistant].ID,
		first.AgentDefinitionID,
	)
}

func TestOpenPageThread_AFormulaNotYetSavedHasItsOwnConversation(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("formula_template:read")

	thread, err := fixture.open(t, conversation.ThreadOriginFormula, "", pulid.Nil)
	require.NoError(t, err)

	require.NotNil(t, fixture.conversations.Created)
	assert.Empty(t, fixture.threads.Claimed)
	assert.False(t, thread.HasSubject())
	assert.False(t, thread.Tainted(), "a formula a person is writing is not outside content")
	assert.Equal(t, agentdefinition.SystemKeyFormulaAssistant,
		fixture.provisioner.Asked[0].SystemKey)

	templateID := pulid.MustNew("ft_")
	saved, err := fixture.open(t, conversation.ThreadOriginFormula, "", templateID)
	require.NoError(t, err)
	assert.Equal(t, agent.SubjectFormulaTemplate, saved.SubjectType)
	assert.Equal(t, templateID, saved.SubjectID)
	assert.False(t, saved.Tainted())
}

func TestOpenPageThread_ALibraryHasItsOwnConversationWithTheFormulaAssistant(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("formula_library:read")

	libraryID := pulid.MustNew("fl_")
	thread, err := fixture.open(t, conversation.ThreadOriginFormula, agent.SubjectFormulaLibrary,
		libraryID)
	require.NoError(t, err)
	assert.Equal(t, agent.SubjectFormulaLibrary, thread.SubjectType)
	assert.Equal(t, libraryID, thread.SubjectID)
	assert.Equal(t, "Formula library", thread.Title)
	assert.Equal(t, agentdefinition.SystemKeyFormulaAssistant,
		fixture.provisioner.Asked[0].SystemKey)

	libraryPage := &agentpagecontext.PageContext{Path: "/", Draft: &pagedraft.Draft{
		Surface:        pagedraft.SurfaceFormulaLibrary,
		FormulaLibrary: &pagedraft.FormulaLibrary{Source: "library \"org.accessorials\"\n"},
	}}
	templatePage := &agentpagecontext.PageContext{Path: "/", Draft: &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormula,
		Formula: &pagedraft.Formula{Source: "tariff \"Per Mile\"\n"},
	}}
	require.NoError(
		t,
		fixture.svc.access().AssertPageTurn(t.Context(), thread, libraryPage, fixture.actor),
	)
	require.Error(
		t,
		fixture.svc.access().AssertPageTurn(t.Context(), thread, templatePage, fixture.actor),
		"a library's conversation takes only the library editor's draft",
	)

	_, err = fixture.open(t, conversation.ThreadOriginFormula, agent.SubjectFormulaLibrary,
		pulid.MustNew("ft_"))
	require.Error(t, err, "a template id is not a library")
}

func TestOpenPageThread_RefusesWithoutTheAssistantOrTheRecord(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")

	noAssistant := newPageFixture("document:read")
	noAssistant.permissions.Allowed = map[string]bool{"document:read": true}
	_, err := noAssistant.open(t, conversation.ThreadOriginImport, "", documentID)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, noAssistant.provisioner.Asked, "nothing is created for a person refused")

	noDocument := newPageFixture()
	_, err = noDocument.open(t, conversation.ThreadOriginImport, "", documentID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do not have access to the record")
	assert.Empty(t, noDocument.threads.Claimed)
}

func TestOpenPageThread_RefusesAnAgentTurnedOff(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("document:read")
	disabled, _ := agentpreset.NewPageAgent(
		agentdefinition.SystemKeyImportAssistant, fixture.actor.TenantInfo(), "Import helper",
	)
	disabled.ID = pulid.MustNew("agdef_")
	disabled.Enabled = false
	fixture.provisioner.Definitions[agentdefinition.SystemKeyImportAssistant] = disabled

	_, err := fixture.open(t, conversation.ThreadOriginImport, "", pulid.MustNew("doc_"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Import helper is turned off")
	assert.Empty(t, fixture.threads.Claimed)
}

func TestOpenPageThread_RefusesARecordOfTheWrongKind(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("document:read", "formula_template:read")

	_, err := fixture.open(t, conversation.ThreadOriginImport, "", pulid.Nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs the document")

	_, err = fixture.open(t, conversation.ThreadOriginImport, "", pulid.MustNew("shp_"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is a shipment, not a document")

	_, err = fixture.open(t, conversation.ThreadOriginFormula, agent.SubjectDocument,
		pulid.MustNew("doc_"))
	require.Error(t, err)

	_, err = fixture.open(t, conversation.ThreadOriginDesk, "", pulid.Nil)
	require.Error(t, err)
	assert.Empty(t, fixture.provisioner.Asked)
}

func TestOpenPageThread_StartsAfreshWhenTheAgentWasReplaced(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("document:read")
	documentID := pulid.MustNew("doc_")
	first, err := fixture.open(t, conversation.ThreadOriginImport, "", documentID)
	require.NoError(t, err)

	replaced, _ := agentpreset.NewPageAgent(
		agentdefinition.SystemKeyImportAssistant, fixture.actor.TenantInfo(), "Import",
	)
	replaced.ID = pulid.MustNew("agdef_")
	fixture.provisioner.Definitions[agentdefinition.SystemKeyImportAssistant] = replaced

	second, err := fixture.open(t, conversation.ThreadOriginImport, "", documentID)
	require.NoError(t, err)
	assert.NotEqual(t, first.ID, second.ID)
	assert.Equal(t, replaced.ID, second.AgentDefinitionID)
	assert.Equal(t, conversation.ThreadStatusArchived, first.Status)
}

func pageThread(origin conversation.ThreadOrigin, subject agent.SubjectType) *conversation.Thread {
	thread := &conversation.Thread{
		ID:     pulid.MustNew("athr_"),
		Origin: origin,
		Status: conversation.ThreadStatusActive,
	}
	if subject != "" {
		thread.SubjectType = subject
		thread.SubjectID = pulid.MustNew(subject.IDPrefix())
	}

	return thread
}

func importDraft() *agentpagecontext.PageContext {
	return &agentpagecontext.PageContext{
		Path: "/shipment-management/shipments/import",
		Draft: &pagedraft.Draft{
			Surface:        pagedraft.SurfaceShipmentImport,
			ShipmentImport: &pagedraft.ShipmentImport{},
		},
	}
}

func TestAssertPageTurn(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("document:read")
	ctx := t.Context()

	importThread := pageThread(conversation.ThreadOriginImport, agent.SubjectDocument)
	require.NoError(
		t,
		fixture.svc.access().AssertPageTurn(ctx, importThread, importDraft(), fixture.actor),
	)
	require.NoError(t, fixture.svc.access().AssertPageTurn(ctx, importThread, nil, fixture.actor))

	formulaPage := &agentpagecontext.PageContext{Path: "/", Draft: &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormula,
		Formula: &pagedraft.Formula{Source: "tariff \"Per Mile\"\n"},
	}}
	libraryPage := &agentpagecontext.PageContext{Path: "/", Draft: &pagedraft.Draft{
		Surface:        pagedraft.SurfaceFormulaLibrary,
		FormulaLibrary: &pagedraft.FormulaLibrary{Source: "library \"org.accessorials\"\n"},
	}}
	err := fixture.svc.access().AssertPageTurn(ctx, importThread, formulaPage, fixture.actor)
	require.Error(t, err, "an import conversation does not take a formula draft")

	newFormula := pageThread(conversation.ThreadOriginFormula, "")
	require.NoError(
		t,
		fixture.svc.access().AssertPageTurn(ctx, newFormula, formulaPage, fixture.actor),
	)
	require.NoError(
		t,
		fixture.svc.access().AssertPageTurn(ctx, newFormula, libraryPage, fixture.actor),
		"an unsaved formula or library takes either editor's draft",
	)
	require.Error(
		t,
		fixture.svc.access().AssertPageTurn(ctx, newFormula, importDraft(), fixture.actor),
	)

	desk := pageThread(conversation.ThreadOriginDesk, "")
	require.NoError(t, fixture.svc.access().AssertPageTurn(ctx, desk, nil, fixture.actor))
	err = fixture.svc.access().AssertPageTurn(ctx, desk, importDraft(), fixture.actor)
	require.Error(t, err, "a draft rides only on a page's conversation")

	closed := pageThread(conversation.ThreadOriginImport, agent.SubjectDocument)
	closed.Status = conversation.ThreadStatusArchived
	err = fixture.svc.access().AssertPageTurn(ctx, closed, importDraft(), fixture.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was closed")

	fixture.permissions.Allowed["document:read"] = false
	err = fixture.svc.access().AssertPageTurn(ctx, importThread, importDraft(), fixture.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}
