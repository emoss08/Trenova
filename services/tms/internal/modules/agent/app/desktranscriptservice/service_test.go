package desktranscriptservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/desktranscriptexport"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

var (
	orgID   = pulid.MustNew("org_")
	buID    = pulid.MustNew("bu_")
	adminID = pulid.MustNew("usr_")
	ownerID = pulid.MustNew("usr_")
	agentID = pulid.MustNew("agdef_")
)

type fakeConversations struct {
	threads    map[pulid.ID]*conversation.Thread
	messages   map[pulid.ID][]conversation.Message
	matchLimit int
}

func (f *fakeConversations) List(
	context.Context,
	*desktranscriptports.ListConversationsRequest,
) (*desktranscriptports.ConversationPage, error) {
	items := make([]*conversation.Thread, 0, len(f.threads))
	cursors := make([]string, 0, len(f.threads))
	for _, thread := range f.threads {
		items = append(items, thread)
		cursors = append(cursors, "cursor")
	}

	return &desktranscriptports.ConversationPage{Items: items, EdgeCursors: cursors}, nil
}

func (f *fakeConversations) CountUpTo(
	_ context.Context,
	req *desktranscriptports.MatchConversationsRequest,
) (int, error) {
	return min(len(f.threads), req.Limit), nil
}

func (f *fakeConversations) Match(
	_ context.Context,
	req *desktranscriptports.MatchConversationsRequest,
) ([]desktranscriptports.ConversationRef, error) {
	f.matchLimit = req.Limit
	refs := make([]desktranscriptports.ConversationRef, 0, len(f.threads))
	for _, thread := range f.threads {
		refs = append(
			refs,
			desktranscriptports.ConversationRef{ID: thread.ID, UserID: thread.UserID},
		)
	}

	return refs, nil
}

func (f *fakeConversations) Get(
	_ context.Context,
	req desktranscriptports.GetConversationRequest,
) (*conversation.Thread, error) {
	thread, ok := f.threads[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Conversation not found")
	}

	return thread, nil
}

func (f *fakeConversations) ListByIDs(
	_ context.Context,
	req desktranscriptports.ConversationsByIDsRequest,
) ([]*conversation.Thread, error) {
	out := make([]*conversation.Thread, 0, len(req.IDs))
	for _, id := range req.IDs {
		if thread, ok := f.threads[id]; ok {
			out = append(out, thread)
		}
	}

	return out, nil
}

func (f *fakeConversations) CountMessages(
	_ context.Context,
	req desktranscriptports.ConversationsByIDsRequest,
) (map[pulid.ID]int, error) {
	counts := make(map[pulid.ID]int, len(req.IDs))
	for _, id := range req.IDs {
		counts[id] = len(f.messages[id])
	}

	return counts, nil
}

func (f *fakeConversations) ListMessages(
	_ context.Context,
	req *desktranscriptports.ThreadMessagesRequest,
) ([]conversation.Message, error) {
	var out []conversation.Message
	for _, id := range req.ThreadIDs {
		out = append(out, f.messages[id]...)
	}

	return out, nil
}

type fakeExports struct {
	desktranscriptports.ExportRepository
	created []*desktranscript.Export
	byID    map[pulid.ID]*desktranscript.Export
}

func (f *fakeExports) Create(
	_ context.Context,
	export *desktranscript.Export,
) (*desktranscript.Export, error) {
	export.ID = pulid.MustNew(desktranscript.ExportIDPrefix)
	f.created = append(f.created, export)

	return export, nil
}

func (f *fakeExports) Update(
	_ context.Context,
	export *desktranscript.Export,
) (*desktranscript.Export, error) {
	return export, nil
}

func (f *fakeExports) GetByID(
	_ context.Context,
	req desktranscriptports.GetExportRequest,
) (*desktranscript.Export, error) {
	export, ok := f.byID[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Desk transcript export not found")
	}

	return export, nil
}

type fakeUsers struct {
	tenantports.UserRepository
}

func (fakeUsers) GetByIDs(
	_ context.Context,
	req tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	out := make([]*tenant.User, 0, len(req.UserIDs))
	for _, id := range req.UserIDs {
		if id == ownerID {
			out = append(
				out,
				&tenant.User{ID: id, Name: "Dana Reyes", EmailAddress: "dana@example.com"},
			)
		}
	}

	return out, nil
}

type fakeDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository
}

func (fakeDefinitions) ListByIDs(
	context.Context,
	agentdefinitionports.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	return []*agentdefinition.Definition{{ID: agentID, Name: "Billing exceptions"}}, nil
}

type fakeRenderer struct{}

func (fakeRenderer) RenderConversation(
	_ context.Context,
	req *desktranscriptports.RenderRequest,
) (*desktranscriptports.RenderedConversation, error) {
	return &desktranscriptports.RenderedConversation{
		FileName: req.OwnerName + " - transcript.md",
		Markdown: "# " + req.Thread.Title,
		Record: desktranscriptports.ConversationRecord{
			Owner: desktranscriptports.RecordOwner{Name: req.OwnerName},
			Agent: req.AgentName,
		},
	}, nil
}

type fakeOrganizations struct{}

func (fakeOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

type fakePermissions struct {
	permissionports.PermissionEngine
	held map[permission.Operation]bool
}

func (f *fakePermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	allowed := req.Resource == permission.ResourceDeskTranscript.String() && f.held[req.Operation]

	return &permissionports.PermissionCheckResult{Allowed: allowed}, nil
}

type fakeAuditor struct {
	changes []*permissionports.SecurityChange
}

func (f *fakeAuditor) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	f.changes = append(f.changes, change)
}

type fakeStorage struct {
	kernelports.Client
}

func (fakeStorage) GetPresignedURL(
	context.Context,
	*kernelports.PresignedURLParams,
) (string, error) {
	return "https://files.example.com/export.zip", nil
}

type fakeWorkflows struct {
	kernelports.WorkflowStarter
	started []string
	err     error
}

func (f *fakeWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	f.started = append(f.started, options.ID)

	return nil, f.err
}

type rig struct {
	service       *Service
	conversations *fakeConversations
	exports       *fakeExports
	auditor       *fakeAuditor
	workflows     *fakeWorkflows
	permissions   *fakePermissions
	thread        *conversation.Thread
}

func newRig(t *testing.T, held ...permission.Operation) *rig {
	t.Helper()

	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		UserID:            ownerID,
		AgentDefinitionID: agentID,
		Title:             "Why is INV-20419 short-paid",
		Origin:            conversation.ThreadOriginDesk,
		Status:            conversation.ThreadStatusActive,
		CreatedAt:         1_700_000_000,
		LastMessageAt:     1_700_000_600,
	}
	conversations := &fakeConversations{
		threads: map[pulid.ID]*conversation.Thread{thread.ID: thread},
		messages: map[pulid.ID][]conversation.Message{thread.ID: {
			{
				ThreadID:  thread.ID,
				Role:      conversation.RoleUser,
				Content:   "Why?",
				CreatedAt: 1_700_000_100,
			},
			{
				ThreadID:  thread.ID,
				Role:      conversation.RoleAssistant,
				Content:   "Because.",
				CreatedAt: 1_700_000_200,
			},
		}},
	}
	exports := &fakeExports{byID: map[pulid.ID]*desktranscript.Export{}}
	auditor := &fakeAuditor{}
	workflows := &fakeWorkflows{}
	permissions := &fakePermissions{held: map[permission.Operation]bool{}}
	for _, op := range held {
		permissions.held[op] = true
	}
	logger := zap.NewNop()
	assembler := desktranscriptexport.NewAssembler(&desktranscriptexport.AssemblerParams{
		Conversations: conversations,
		Users:         fakeUsers{},
		Definitions:   fakeDefinitions{},
		Renderer:      fakeRenderer{},
	})
	exporter := desktranscriptexport.NewExports(&desktranscriptexport.ExportsParams{
		Exports:       exports,
		Conversations: conversations,
		Assembler:     assembler,
		Organizations: fakeOrganizations{},
		Logger:        logger,
	})

	return &rig{
		service: NewService(&ServiceParams{
			Conversations: conversations,
			ExportRepo:    exports,
			Assembler:     assembler,
			Exports:       exporter,
			Organizations: fakeOrganizations{},
			Permissions:   permissions,
			Auditor:       auditor,
			Storage:       fakeStorage{},
			Workflows:     workflows,
			Now:           func() time.Time { return time.Unix(1_700_001_000, 0) },
			Logger:        logger,
		}),
		conversations: conversations,
		exports:       exports,
		auditor:       auditor,
		workflows:     workflows,
		permissions:   permissions,
		thread:        thread,
	}
}

func person() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    adminID,
		UserID:         adminID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}

func assertRefused(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var refused *errortypes.AuthorizationError
	assert.True(t, errors.As(err, &refused), "want an authorization error, got %v", err)
}

func assertInvalid(t *testing.T, err error, path, message string) {
	t.Helper()
	require.Error(t, err)
	var invalid *errortypes.Error
	require.True(t, errors.As(err, &invalid), "want a validation error, got %v", err)
	assert.Equal(t, path, invalid.Path.String())
	assert.Equal(t, message, invalid.Message)
}

func TestHolderOfViewListsAndOpensButTakesNothingOut(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead)
	ctx := t.Context()

	list, err := r.service.ListConversations(ctx, &desktranscriptports.ListRequest{Actor: person()})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "Dana Reyes", list.Items[0].OwnerName)
	assert.Equal(t, "Billing exceptions", list.Items[0].AgentName)
	assert.Equal(t, 2, list.Items[0].MessageCount)

	opened, err := r.service.OpenConversation(ctx, desktranscriptports.ConversationRequest{
		Actor: person(),
		ID:    r.thread.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, opened.Transcript)
	assert.Len(t, opened.Transcript.Messages, 2)

	_, err = r.service.DownloadConversation(ctx, desktranscriptports.ConversationRequest{
		Actor: person(),
		ID:    r.thread.ID,
	})
	assertRefused(t, err)

	_, err = r.service.RequestExport(ctx, &desktranscriptports.RequestExportRequest{
		Actor:           person(),
		ConversationIDs: []pulid.ID{r.thread.ID},
	})
	assertRefused(t, err)

	_, err = r.service.ExportDownload(ctx, desktranscriptports.ExportDownloadRequest{
		Actor:    person(),
		ExportID: pulid.MustNew(desktranscript.ExportIDPrefix),
	})
	assertRefused(t, err)
	assert.Empty(t, r.exports.created)
}

func TestNonHolderIsRefusedEverything(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	ctx := t.Context()

	_, err := r.service.ListConversations(ctx, &desktranscriptports.ListRequest{Actor: person()})
	assertRefused(t, err)
	_, err = r.service.OpenConversation(ctx, desktranscriptports.ConversationRequest{
		Actor: person(),
		ID:    r.thread.ID,
	})
	assertRefused(t, err)
	_, err = r.service.PreviewExport(ctx, &desktranscriptports.PreviewRequest{Actor: person()})
	assertRefused(t, err)
	_, err = r.service.ListExports(ctx, &desktranscriptports.ListExportsServiceRequest{
		Actor: person(),
	})
	assertRefused(t, err)
	assert.Empty(t, r.auditor.changes)
}

func TestOnlyAPersonActingForThemselvesGetsIn(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)
	ctx := t.Context()

	agent := person()
	agent.PrincipalType = permissionports.PrincipalTypeAgent
	apiKey := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAPIKey,
		PrincipalID:    pulid.MustNew("ak_"),
		APIKeyID:       pulid.MustNew("ak_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
	support := person()
	support.UserID = pulid.ID(supportaccess.PrincipalIDPrefix + "01J0000000000000000000000")
	support.PrincipalID = support.UserID
	system := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeSystem,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}

	for name, actor := range map[string]*permissionports.RequestActor{
		"an agent":        agent,
		"an API key":      apiKey,
		"Trenova support": support,
		"the system":      system,
		"nobody":          nil,
	} {
		_, err := r.service.OpenConversation(ctx, desktranscriptports.ConversationRequest{
			Actor: actor,
			ID:    r.thread.ID,
		})
		assertRefused(t, err)
		_, err = r.service.DownloadConversation(ctx, desktranscriptports.ConversationRequest{
			Actor: actor,
			ID:    r.thread.ID,
		})
		assertRefused(t, err)
		assert.Empty(t, r.auditor.changes, name)
	}
}

func TestEveryLookIsAudited(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)
	ctx := t.Context()
	req := desktranscriptports.ConversationRequest{Actor: person(), ID: r.thread.ID}

	_, err := r.service.OpenConversation(ctx, req)
	require.NoError(t, err)
	file, err := r.service.DownloadConversation(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Dana Reyes - transcript.md", file.FileName)
	_, err = r.service.RequestExport(ctx, &desktranscriptports.RequestExportRequest{
		Actor:           person(),
		ConversationIDs: []pulid.ID{r.thread.ID},
	})
	require.NoError(t, err)

	require.Len(t, r.auditor.changes, 3)
	opened, downloaded, requested := r.auditor.changes[0], r.auditor.changes[1], r.auditor.changes[2]
	assert.Equal(t, actionOpened, opened.Action)
	assert.Equal(t, permission.OpRead, opened.Operation)
	assert.Equal(t, r.thread.ID.String(), opened.ResourceID)
	assert.Equal(t, adminID, opened.Actor.UserID)
	assert.Equal(t, "Dana Reyes", opened.After.(map[string]any)["ownerName"])
	assert.Equal(t, actionDownloaded, downloaded.Action)
	assert.Equal(t, permission.OpExport, downloaded.Operation)
	assert.Equal(t, actionExportRequested, requested.Action)
	state := requested.After.(map[string]any)
	assert.Equal(t, "Selection", state["kind"])
	assert.Equal(t, 1, state["conversationCount"])
	assert.Equal(t, []string{"Dana Reyes"}, state["ownerNames"])
	for _, change := range r.auditor.changes {
		assert.Equal(t, permission.ResourceDeskTranscript, change.Resource)
	}
}

func TestExportRequestIsExactlyOneShape(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)
	ctx := t.Context()
	filter := desktranscript.ConversationFilter{Timeframe: desktranscript.AllTime()}

	_, err := r.service.RequestExport(ctx, &desktranscriptports.RequestExportRequest{
		Actor:           person(),
		Filter:          &filter,
		ConversationIDs: []pulid.ID{r.thread.ID},
	})
	assertInvalid(t, err, "conversationIds",
		"Export either a timeframe or the conversations you selected, not both")

	_, err = r.service.RequestExport(ctx, &desktranscriptports.RequestExportRequest{
		Actor: person(),
	})
	assertInvalid(t, err, "filter", "Choose a timeframe or select conversations to export")
	assert.Empty(t, r.exports.created)
	assert.Empty(t, r.workflows.started)
}

func TestASelectedConversationOfAnotherOrganizationIsNotFound(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)

	_, err := r.service.RequestExport(t.Context(), &desktranscriptports.RequestExportRequest{
		Actor:           person(),
		ConversationIDs: []pulid.ID{r.thread.ID, pulid.MustNew("athr_")},
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, r.exports.created)
}

func TestTooManySelectedConversationsAreRefused(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)
	ids := make([]pulid.ID, desktranscript.MaxSelectedConversations+1)
	for idx := range ids {
		ids[idx] = pulid.MustNew("athr_")
	}

	_, err := r.service.RequestExport(t.Context(), &desktranscriptports.RequestExportRequest{
		Actor:           person(),
		ConversationIDs: ids,
	})

	assertInvalid(t, err, "conversationIds",
		"Select at most {0} conversations to download at once, or export a timeframe instead")
}

func TestAFilterExportFixesItsConversationsAndStartsOnce(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)
	filter := desktranscript.ConversationFilter{
		Timeframe: desktranscript.Timeframe{Preset: desktranscript.TimeframeLast30Days},
	}

	export, err := r.service.RequestExport(t.Context(), &desktranscriptports.RequestExportRequest{
		Actor:  person(),
		Filter: &filter,
	})

	require.NoError(t, err)
	assert.Equal(t, desktranscript.MaxExportConversations+1, r.conversations.matchLimit,
		"the match reads one past the limit so an export over it is refused, never cut")
	assert.Equal(t, []pulid.ID{r.thread.ID}, export.ThreadIDs)
	assert.Equal(t, desktranscript.ExportKindFilter, export.Kind)
	assert.Equal(t, desktranscript.ExportStatusRunning, export.Status)
	assert.Equal(t, []string{workflowIDPrefix + export.ID.String()}, r.workflows.started)
	assert.Equal(t, 1, export.Coverage.OwnerCount)
}

func TestAnExportThatCannotBeQueuedFails(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)
	r.workflows.err = errors.New("temporal is down")

	_, err := r.service.RequestExport(t.Context(), &desktranscriptports.RequestExportRequest{
		Actor:           person(),
		ConversationIDs: []pulid.ID{r.thread.ID},
	})

	require.Error(t, err)
	require.Len(t, r.exports.created, 1)
	assert.Equal(t, desktranscript.ExportStatusFailed, r.exports.created[0].Status)
}

func TestOnlyTheRequesterDownloadsAnExport(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)
	expires := int64(1_700_100_000)
	export := &desktranscript.Export{
		ID:                pulid.MustNew(desktranscript.ExportIDPrefix),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		RequestedByUserID: ownerID,
		Kind:              desktranscript.ExportKindSelection,
		Status:            desktranscript.ExportStatusSucceeded,
		ArtifactKey:       "desk-transcript-exports/org/dtex.zip",
		ArtifactExpiresAt: &expires,
	}
	r.exports.byID[export.ID] = export

	_, err := r.service.ExportDownload(t.Context(), desktranscriptports.ExportDownloadRequest{
		Actor:    person(),
		ExportID: export.ID,
	})
	assertRefused(t, err)

	export.RequestedByUserID = adminID
	download, err := r.service.ExportDownload(
		t.Context(),
		desktranscriptports.ExportDownloadRequest{
			Actor:    person(),
			ExportID: export.ID,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "https://files.example.com/export.zip", download.URL)
	require.Len(t, r.auditor.changes, 1)
	assert.Equal(t, actionExportDownloaded, r.auditor.changes[0].Action)
}

func TestAnExpiredExportCannotBeDownloaded(t *testing.T) {
	t.Parallel()

	r := newRig(t, permission.OpRead, permission.OpExport)
	expired := int64(1_700_000_000)
	export := &desktranscript.Export{
		ID:                pulid.MustNew(desktranscript.ExportIDPrefix),
		RequestedByUserID: adminID,
		Status:            desktranscript.ExportStatusSucceeded,
		ArtifactKey:       "desk-transcript-exports/org/dtex.zip",
		ArtifactExpiresAt: &expired,
	}
	r.exports.byID[export.ID] = export

	_, err := r.service.ExportDownload(t.Context(), desktranscriptports.ExportDownloadRequest{
		Actor:    person(),
		ExportID: export.ID,
	})

	require.Error(t, err)
	var business *errortypes.BusinessError
	assert.True(t, errors.As(err, &business))
	assert.Empty(t, r.auditor.changes)
}
