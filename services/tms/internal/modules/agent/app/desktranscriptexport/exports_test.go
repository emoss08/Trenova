package desktranscriptexport

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/storage/uploadpipe"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testOrg   = pulid.MustNew("org_")
	testBU    = pulid.MustNew("bu_")
	testOwner = pulid.MustNew("usr_")
	testAgent = pulid.MustNew("agdef_")
)

type pagedConversations struct {
	desktranscriptports.ConversationRepository
	threads  map[pulid.ID]*conversation.Thread
	reads    [][]pulid.ID
	messages map[pulid.ID][]conversation.Message
}

func (p *pagedConversations) ListByIDs(
	_ context.Context,
	req desktranscriptports.ConversationsByIDsRequest,
) ([]*conversation.Thread, error) {
	p.reads = append(p.reads, req.IDs)
	out := make([]*conversation.Thread, 0, len(req.IDs))
	for _, id := range req.IDs {
		if thread, ok := p.threads[id]; ok {
			out = append(out, thread)
		}
	}

	return out, nil
}

func (p *pagedConversations) CountMessages(
	_ context.Context,
	req desktranscriptports.ConversationsByIDsRequest,
) (map[pulid.ID]int, error) {
	counts := map[pulid.ID]int{}
	for _, id := range req.IDs {
		counts[id] = len(p.messages[id])
	}

	return counts, nil
}

func (p *pagedConversations) ListMessages(
	_ context.Context,
	req *desktranscriptports.ThreadMessagesRequest,
) ([]conversation.Message, error) {
	var out []conversation.Message
	for _, id := range req.ThreadIDs {
		out = append(out, p.messages[id]...)
	}

	return out, nil
}

type memoryExports struct {
	desktranscriptports.ExportRepository
	export *desktranscript.Export
}

func (m *memoryExports) GetByID(
	_ context.Context,
	req desktranscriptports.GetExportRequest,
) (*desktranscript.Export, error) {
	if m.export == nil || m.export.ID != req.ID {
		return nil, errortypes.NewNotFoundError("Desk transcript export not found")
	}

	return m.export, nil
}

func (m *memoryExports) Update(
	_ context.Context,
	export *desktranscript.Export,
) (*desktranscript.Export, error) {
	m.export = export

	return export, nil
}

type namedUsers struct{ tenantports.UserRepository }

func (namedUsers) GetByIDs(
	_ context.Context,
	req tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	return []*tenant.User{
		{ID: testOwner, Name: "Dana Reyes", EmailAddress: "dana@example.com"},
	}, nil
}

type namedAgents struct {
	agentdefinitionports.AgentDefinitionRepository
}

func (namedAgents) ListByIDs(
	context.Context,
	agentdefinitionports.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	return []*agentdefinition.Definition{{ID: testAgent, Name: "Billing exceptions"}}, nil
}

type plainRenderer struct{}

func (plainRenderer) RenderConversation(
	_ context.Context,
	req *desktranscriptports.RenderRequest,
) (*desktranscriptports.RenderedConversation, error) {
	return &desktranscriptports.RenderedConversation{
		ArchiveFolder: req.OwnerName,
		ArchiveName:   "2026-03-01 - " + req.Thread.Title + ".md",
		Markdown:      "# " + req.Thread.Title,
		Record: desktranscriptports.ConversationRecord{
			Owner: desktranscriptports.RecordOwner{Name: req.OwnerName, Email: req.OwnerEmail},
			Agent: req.AgentName,
			Title: req.Thread.Title,
		},
	}, nil
}

type utcOrganization struct{}

func (utcOrganization) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "UTC"}, nil
}

type memoryStorage struct {
	kernelports.Client
	mu    sync.Mutex
	files map[string][]byte
}

func (s *memoryStorage) Upload(
	_ context.Context,
	params *kernelports.UploadParams,
) (*kernelports.FileInfo, error) {
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[params.Key] = body

	return &kernelports.FileInfo{Key: params.Key, Size: int64(len(body))}, nil
}

type notices struct {
	sent []*notification.Notification
}

func (n *notices) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	n.sent = append(n.sent, entity)

	return entity, nil
}

type exportRig struct {
	exports       *Exports
	conversations *pagedConversations
	repo          *memoryExports
	storage       *memoryStorage
	notices       *notices
}

func newExportRig(t *testing.T, conversations int) *exportRig {
	t.Helper()

	paged := &pagedConversations{
		threads:  map[pulid.ID]*conversation.Thread{},
		messages: map[pulid.ID][]conversation.Message{},
	}
	ids := make([]pulid.ID, 0, conversations)
	for idx := range conversations {
		thread := &conversation.Thread{
			ID:                pulid.MustNew("athr_"),
			UserID:            testOwner,
			AgentDefinitionID: testAgent,
			Title:             "Conversation " + string(rune('A'+idx%26)),
			CreatedAt:         1_772_323_200,
		}
		paged.threads[thread.ID] = thread
		paged.messages[thread.ID] = []conversation.Message{
			{ThreadID: thread.ID, Role: conversation.RoleUser, Content: "Hello"},
		}
		ids = append(ids, thread.ID)
	}
	repo := &memoryExports{export: &desktranscript.Export{
		ID:                pulid.MustNew(desktranscript.ExportIDPrefix),
		OrganizationID:    testOrg,
		BusinessUnitID:    testBU,
		RequestedByUserID: pulid.MustNew("usr_"),
		Kind:              desktranscript.ExportKindSelection,
		ThreadIDs:         ids,
		ConversationCount: len(ids),
		Status:            desktranscript.ExportStatusRunning,
	}}
	storage := &memoryStorage{files: map[string][]byte{}}
	sent := &notices{}
	assembler := NewAssembler(&AssemblerParams{
		Conversations: paged,
		Users:         namedUsers{},
		Definitions:   namedAgents{},
		Renderer:      plainRenderer{},
	})

	return &exportRig{
		exports: NewExports(&ExportsParams{
			Exports:       repo,
			Conversations: paged,
			Assembler:     assembler,
			Organizations: utcOrganization{},
			Storage:       storage,
			Notifier:      sent,
			Now:           func() time.Time { return time.Unix(1_772_400_000, 0) },
			Logger:        zap.NewNop(),
		}),
		conversations: paged,
		repo:          repo,
		storage:       storage,
		notices:       sent,
	}
}

func (r *exportRig) run(t *testing.T) *desktranscript.Export {
	t.Helper()
	export, err := r.exports.Run(
		t.Context(),
		pagination.TenantInfo{OrgID: testOrg, BuID: testBU},
		r.repo.export.ID,
		nil,
	)
	require.NoError(t, err)

	return export
}

func TestRunPagesThroughTheConversationsAndWritesOneZip(t *testing.T) {
	t.Parallel()

	r := newExportRig(t, conversationsRead+5)
	export := r.run(t)

	assert.Equal(t, desktranscript.ExportStatusSucceeded, export.Status)
	assert.Equal(t, conversationsRead+5, export.ConversationCount)
	require.Len(t, r.conversations.reads, 2, "conversations are read a page at a time")
	assert.Len(t, r.conversations.reads[0], conversationsRead)
	assert.Len(t, r.conversations.reads[1], 5)
	require.NotNil(t, export.ArtifactExpiresAt)
	assert.Equal(t, int64(1_772_400_000)+int64(ExportTTL.Seconds()), *export.ArtifactExpiresAt)

	file := r.storage.files[export.ArtifactKey]
	require.NotEmpty(t, file)
	assert.Equal(t, int64(len(file)), export.ByteSize)
	archive, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	require.NoError(t, err)

	names := map[string]bool{}
	var records *zip.File
	for _, entry := range archive.File {
		names[entry.Name] = true
		if entry.Name == recordsFileName {
			records = entry
		}
		assert.NotContains(t, entry.Name, "athr_", "no record id in a file name")
	}
	assert.True(t, names["Dana Reyes/2026-03-01 - Conversation A.md"])
	assert.True(t, names["Dana Reyes/2026-03-01 - Conversation A (2).md"],
		"a second conversation with the same name is numbered, not overwritten")
	require.NotNil(t, records)

	reader, err := records.Open()
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	require.Len(t, lines, conversationsRead+5)
	var first desktranscriptports.ConversationRecord
	require.NoError(t, sonic.UnmarshalString(lines[0], &first))
	assert.Equal(t, "Dana Reyes", first.Owner.Name)
	assert.Equal(t, "dana@example.com", first.Owner.Email)

	require.Len(t, r.notices.sent, 1)
	assert.Equal(t, desktranscriptports.ExportReadyEvent, r.notices.sent[0].EventType)
}

func TestAFileOverTheSizeBoundFailsWithWhatToDo(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"The file passed the 1024 MB an export may hold. Export a shorter timeframe, one person or one agent.",
		failureMessage(t.Context(), fmt.Errorf("write: %w", uploadpipe.ErrTooLarge)),
	)
	assert.NotContains(
		t,
		failureMessage(t.Context(), errors.New("connection reset")),
		"connection reset",
		"a failure's own wording never reaches the requester",
	)
}

func TestAFailedRunNotifiesTheRequester(t *testing.T) {
	t.Parallel()

	r := newExportRig(t, 1)
	failed, err := r.exports.Fail(t.Context(), r.repo.export, "The export could not be written.")
	require.NoError(t, err)

	assert.Equal(t, desktranscript.ExportStatusFailed, failed.Status)
	require.Len(t, r.notices.sent, 1)
	assert.Equal(t, desktranscriptports.ExportFailedEvent, r.notices.sent[0].EventType)
	assert.Equal(t, desktranscriptports.ExportsPath, r.notices.sent[0].Data["link"])
	assert.Equal(t, r.repo.export.RequestedByUserID, *r.notices.sent[0].TargetUserID)
}

func TestRunLeavesAFinishedExportAsItIs(t *testing.T) {
	t.Parallel()

	r := newExportRig(t, 1)
	r.repo.export.Status = desktranscript.ExportStatusSucceeded

	export := r.run(t)

	assert.Equal(t, desktranscript.ExportStatusSucceeded, export.Status)
	assert.Empty(t, r.conversations.reads, "a retried activity writes nothing twice")
	assert.Empty(t, r.notices.sent)
}

func TestReadGroupsBoundEachMessageRead(t *testing.T) {
	t.Parallel()

	threads := make([]*conversation.Thread, 4)
	counts := map[pulid.ID]int{}
	for idx := range threads {
		threads[idx] = &conversation.Thread{ID: pulid.MustNew("athr_")}
		counts[threads[idx].ID] = 900
	}

	groups := readGroups(threads, counts)

	require.Len(t, groups, 2)
	assert.Len(t, groups[0], 2)
	assert.Len(t, groups[1], 2)
}
