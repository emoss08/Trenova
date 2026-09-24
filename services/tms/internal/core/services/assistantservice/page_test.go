package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type provisioner struct {
	definitions map[string]*agentdefinition.Definition
	asked       []serviceports.EnsureSystemAgentRequest
}

func (p *provisioner) EnsureSystem(
	_ context.Context,
	req serviceports.EnsureSystemAgentRequest,
) (*agentdefinition.Definition, error) {
	p.asked = append(p.asked, req)
	definition, ok := p.definitions[req.SystemKey]
	if !ok {
		definition, _ = agentdefinition.NewPageAgent(req.SystemKey, req.TenantInfo, req.SystemKey)
		definition.ID = pulid.MustNew("agdef_")
		p.definitions[req.SystemKey] = definition
	}

	return definition, nil
}

type pageThreads struct {
	live     map[repositories.PageThreadKey]*conversation.Thread
	claimed  []*conversation.Thread
	archived []repositories.ArchiveSubjectThreadsRequest
}

func keyOf(thread *conversation.Thread) repositories.PageThreadKey {
	return repositories.PageThreadKey{
		TenantInfo: pagination.TenantInfo{
			OrgID: thread.OrganizationID,
			BuID:  thread.BusinessUnitID,
		},
		UserID:      thread.UserID,
		Origin:      thread.Origin,
		SubjectType: thread.SubjectType,
		SubjectID:   thread.SubjectID,
	}
}

func (p *pageThreads) GetPageThread(
	_ context.Context,
	key repositories.PageThreadKey,
) (*conversation.Thread, error) {
	key.TenantInfo.UserID = pulid.Nil
	if thread, ok := p.live[key]; ok {
		return thread, nil
	}

	return nil, errortypes.NewNotFoundError("Thread not found")
}

func (p *pageThreads) ClaimPageThread(
	_ context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	key := keyOf(thread)
	if existing, ok := p.live[key]; ok {
		return existing, nil
	}
	thread.ID = pulid.MustNew("athr_")
	p.live[key] = thread
	p.claimed = append(p.claimed, thread)

	return thread, nil
}

func (p *pageThreads) ArchiveSubjectThreads(
	_ context.Context,
	req repositories.ArchiveSubjectThreadsRequest,
) (int, error) {
	p.archived = append(p.archived, req)
	closed := 0
	for key, thread := range p.live {
		if key.Origin == req.Origin && key.SubjectID == req.SubjectID {
			thread.Status = conversation.ThreadStatusArchived
			delete(p.live, key)
			closed++
		}
	}

	return closed, nil
}

type pageFixture struct {
	svc           *Service
	provisioner   *provisioner
	threads       *pageThreads
	conversations *creatingConversations
	permissions   *subjectPermissions
	actor         *serviceports.RequestActor
}

func newPageFixture(allowed ...string) *pageFixture {
	permissions := &subjectPermissions{allowed: map[string]bool{"assistant:create": true}}
	for _, key := range allowed {
		permissions.allowed[key] = true
	}
	fixture := &pageFixture{
		provisioner:   &provisioner{definitions: map[string]*agentdefinition.Definition{}},
		threads:       &pageThreads{live: map[repositories.PageThreadKey]*conversation.Thread{}},
		conversations: &creatingConversations{},
		permissions:   permissions,
		actor:         testActor(),
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

	return f.svc.OpenPageThread(t.Context(), &serviceports.OpenPageThreadRequest{
		TenantInfo:  f.actor.TenantInfo(),
		Origin:      origin,
		SubjectType: subjectType,
		SubjectID:   subjectID,
	}, f.actor)
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
	require.Len(t, fixture.threads.claimed, 1)
	assert.Nil(t, fixture.conversations.created, "a document's conversation is claimed on its key")

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

	require.NotEmpty(t, fixture.provisioner.asked)
	assert.Equal(
		t,
		agentdefinition.SystemKeyImportAssistant,
		fixture.provisioner.asked[0].SystemKey,
	)
	assert.Equal(t,
		fixture.provisioner.definitions[agentdefinition.SystemKeyImportAssistant].ID,
		first.AgentDefinitionID,
	)
}

func TestOpenPageThread_AFormulaNotYetSavedHasItsOwnConversation(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("formula_template:read")

	thread, err := fixture.open(t, conversation.ThreadOriginFormula, "", pulid.Nil)
	require.NoError(t, err)

	require.NotNil(t, fixture.conversations.created)
	assert.Empty(t, fixture.threads.claimed)
	assert.False(t, thread.HasSubject())
	assert.False(t, thread.Tainted(), "a formula a person is writing is not outside content")
	assert.Equal(t, agentdefinition.SystemKeyFormulaAssistant,
		fixture.provisioner.asked[0].SystemKey)

	templateID := pulid.MustNew("ft_")
	saved, err := fixture.open(t, conversation.ThreadOriginFormula, "", templateID)
	require.NoError(t, err)
	assert.Equal(t, agent.SubjectFormulaTemplate, saved.SubjectType)
	assert.Equal(t, templateID, saved.SubjectID)
	assert.False(t, saved.Tainted())
}

func TestOpenPageThread_RefusesWithoutTheAssistantOrTheRecord(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")

	noAssistant := newPageFixture("document:read")
	noAssistant.permissions.allowed = map[string]bool{"document:read": true}
	_, err := noAssistant.open(t, conversation.ThreadOriginImport, "", documentID)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, noAssistant.provisioner.asked, "nothing is created for a person refused")

	noDocument := newPageFixture()
	_, err = noDocument.open(t, conversation.ThreadOriginImport, "", documentID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do not have access to the record")
	assert.Empty(t, noDocument.threads.claimed)
}

func TestOpenPageThread_RefusesAnAgentTurnedOff(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("document:read")
	disabled, _ := agentdefinition.NewPageAgent(
		agentdefinition.SystemKeyImportAssistant, fixture.actor.TenantInfo(), "Import helper",
	)
	disabled.ID = pulid.MustNew("agdef_")
	disabled.Enabled = false
	fixture.provisioner.definitions[agentdefinition.SystemKeyImportAssistant] = disabled

	_, err := fixture.open(t, conversation.ThreadOriginImport, "", pulid.MustNew("doc_"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Import helper is turned off")
	assert.Empty(t, fixture.threads.claimed)
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
	assert.Empty(t, fixture.provisioner.asked)
}

func TestOpenPageThread_StartsAfreshWhenTheAgentWasReplaced(t *testing.T) {
	t.Parallel()

	fixture := newPageFixture("document:read")
	documentID := pulid.MustNew("doc_")
	first, err := fixture.open(t, conversation.ThreadOriginImport, "", documentID)
	require.NoError(t, err)

	replaced, _ := agentdefinition.NewPageAgent(
		agentdefinition.SystemKeyImportAssistant, fixture.actor.TenantInfo(), "Import",
	)
	replaced.ID = pulid.MustNew("agdef_")
	fixture.provisioner.definitions[agentdefinition.SystemKeyImportAssistant] = replaced

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

func importDraft() *agent.PageContext {
	return &agent.PageContext{
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
	require.NoError(t, fixture.svc.assertPageTurn(ctx, importThread, importDraft(), fixture.actor))
	require.NoError(t, fixture.svc.assertPageTurn(ctx, importThread, nil, fixture.actor))

	formulaPage := &agent.PageContext{Path: "/", Draft: &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormula,
		Formula: &pagedraft.Formula{SchemaID: "shipment"},
	}}
	err := fixture.svc.assertPageTurn(ctx, importThread, formulaPage, fixture.actor)
	require.Error(t, err, "an import conversation does not take a formula draft")

	desk := pageThread(conversation.ThreadOriginDesk, "")
	require.NoError(t, fixture.svc.assertPageTurn(ctx, desk, nil, fixture.actor))
	err = fixture.svc.assertPageTurn(ctx, desk, importDraft(), fixture.actor)
	require.Error(t, err, "a draft rides only on a page's conversation")

	closed := pageThread(conversation.ThreadOriginImport, agent.SubjectDocument)
	closed.Status = conversation.ThreadStatusArchived
	err = fixture.svc.assertPageTurn(ctx, closed, importDraft(), fixture.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was closed")

	fixture.permissions.allowed["document:read"] = false
	err = fixture.svc.assertPageTurn(ctx, importThread, importDraft(), fixture.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}
