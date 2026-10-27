package retrievaltest

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent/documentcontentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/pkg/dbscope"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/retrievalservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type Corpus struct {
	Tenant pagination.TenantInfo
	Keys   map[pulid.ID]string
	IDs    map[string]pulid.ID
}

func newCorpus(tenant pagination.TenantInfo, size int) *Corpus {
	return &Corpus{
		Tenant: tenant,
		Keys:   make(map[pulid.ID]string, size),
		IDs:    make(map[string]pulid.ID, size),
	}
}

func (c *Corpus) add(key string, id pulid.ID) {
	c.Keys[id] = key
	c.IDs[key] = id
}

func (c *Corpus) KeysOf(ids []pulid.ID) []string {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		if key, ok := c.Keys[id]; ok {
			keys = append(keys, key)
			continue
		}
		keys = append(keys, "foreign:"+id.String())
	}

	return keys
}

type DocumentSeed struct {
	Tenant       pagination.TenantInfo
	UploadedByID pulid.ID
	OwnerType    string
	OwnerID      pulid.ID
	Items        []agentevalgate.RetrievalItem
}

func DocumentFor(
	item agentevalgate.RetrievalItem,
	owner string,
) *agentports.RetrievalDocumentSource {
	if owner == "" {
		owner = item.AttachedTo
	}

	doc := &document.Document{
		FileName:         item.FileName,
		OriginalName:     item.FileName,
		FileSize:         int64(len(strings.Join(item.Pages, "\n"))),
		FileType:         "application/pdf",
		StoragePath:      "eval/" + item.Key + ".pdf",
		Status:           document.StatusActive,
		IsCurrentVersion: true,
		ResourceType:     owner,
		DetectedKind:     item.Kind,
		ContentStatus:    document.ContentStatusIndexed,
	}
	content := &documentcontent.Content{
		Status:               documentcontent.StatusIndexed,
		ContentText:          strings.Join(item.Pages, "\n\n"),
		PageCount:            len(item.Pages),
		DetectedDocumentKind: item.Kind,
		StructuredData:       structuredFields(item.Fields),
	}
	pages := make([]*documentcontent.Page, 0, len(item.Pages))
	for idx, text := range item.Pages {
		pages = append(pages, &documentcontent.Page{
			PageNumber:    idx + 1,
			SourceKind:    documentcontent.SourceKindNative,
			ExtractedText: text,
			Metadata:      map[string]any{},
		})
	}

	return &agentports.RetrievalDocumentSource{Document: doc, Content: content, Pages: pages}
}

func SeedDocuments(t *testing.T, ctx context.Context, db *keel.DB, seed DocumentSeed) *Corpus {
	t.Helper()

	corpus := newCorpus(seed.Tenant, len(seed.Items))
	for _, item := range seed.Items {
		source := DocumentFor(item, seed.OwnerType)
		doc := source.Document
		doc.OrganizationID = seed.Tenant.OrgID
		doc.BusinessUnitID = seed.Tenant.BuID
		doc.ResourceID = pulid.MustNew("shp_").String()
		if seed.OwnerID.IsNotNil() {
			doc.ResourceID = seed.OwnerID.String()
		}
		doc.UploadedByID = seed.UploadedByID
		err := documentdb.Documents.Create(tenantContext(ctx, seed.Tenant), db, doc)
		require.NoError(t, err, item.Key)

		content := source.Content
		content.DocumentID = doc.ID
		content.OrganizationID = seed.Tenant.OrgID
		content.BusinessUnitID = seed.Tenant.BuID
		err = documentcontentdb.Contents.Create(tenantContext(ctx, seed.Tenant), db, content)
		require.NoError(t, err, item.Key)

		for _, page := range source.Pages {
			page.DocumentContentID = content.ID
			page.DocumentID = doc.ID
			page.OrganizationID = seed.Tenant.OrgID
			page.BusinessUnitID = seed.Tenant.BuID
		}
		if len(source.Pages) > 0 {
			err = documentcontentdb.Pages.CreateMany(tenantContext(ctx, seed.Tenant), db, source.Pages)
			require.NoError(t, err, item.Key)
		}

		corpus.add(item.Key, doc.ID)
	}

	return corpus
}

func SeedColleague(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
) pulid.ID {
	t.Helper()

	id := strings.ToLower(pulid.MustNew("usr_").String())
	marker := "c" + id[len(id)-12:]
	return seedtest.NewUser(tenant.OrgID, tenant.BuID).
		WithName("Colleague").
		WithUsername(marker).
		WithEmail(marker+"@example.com").
		Build(t, ctx, db).ID
}

func SeedConversation(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) pulid.ID {
	t.Helper()

	thread := &conversation.Thread{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		UserID:            userID,
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Title:             "Private conversation",
		Status:            conversation.ThreadStatusActive,
		Origin:            conversation.ThreadOriginPanel,
	}
	err := conversationdb.Threads.Create(tenantContext(ctx, tenant), db, thread)
	require.NoError(t, err)

	return thread.ID
}

func structuredFields(fields []string) map[string]any {
	structured := make(map[string]any, len(fields))
	for _, field := range fields {
		key, value, ok := strings.Cut(field, ":")
		if ok {
			structured[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}

	return structured
}

func SeedMailbox(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
) pulid.ID {
	t.Helper()

	mailbox := &inboundmessage.Mailbox{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Name:           "Eval inbox",
		Address:        "inbox-" + strings.ToLower(tenant.OrgID.String()) + "@eval.test",
		Provider:       inboundmessage.ProviderPostmark,
		TokenHash:      pulid.MustNew("tok_").String(),
		ReviewPolicy:   inboundmessage.ReviewAlways,
	}
	err := inboundmessagedb.Mailboxes.Create(tenantContext(ctx, tenant), db, mailbox)
	require.NoError(t, err)

	return mailbox.ID
}

type MessageSeed struct {
	Tenant    pagination.TenantInfo
	MailboxID pulid.ID
	Items     []agentevalgate.RetrievalItem
}

func MessageFor(item agentevalgate.RetrievalItem) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ProviderMessageID: item.Key,
		FromAddress:       item.From,
		FromName:          item.FromName,
		Subject:           item.Subject,
		TextBody:          item.Body,
		Status:            inboundmessage.StatusInReview,
	}
}

func SeedMessages(t *testing.T, ctx context.Context, db *keel.DB, seed MessageSeed) *Corpus {
	t.Helper()

	corpus := newCorpus(seed.Tenant, len(seed.Items))
	for idx, item := range seed.Items {
		message := MessageFor(item)
		message.OrganizationID = seed.Tenant.OrgID
		message.BusinessUnitID = seed.Tenant.BuID
		message.MailboxID = seed.MailboxID
		message.ReceivedAt = int64(1_790_000_000 + idx*60)
		err := inboundmessagedb.InboundMessages.Create(tenantContext(ctx, seed.Tenant), db, message)
		require.NoError(t, err, item.Key)

		corpus.add(item.Key, message.ID)
	}

	return corpus
}

type MemorySeed struct {
	Tenant            pagination.TenantInfo
	Items             []agentevalgate.RetrievalItem
	Status            agent.MemoryStatus
	Scope             agent.MemoryScope
	AgentDefinitionID pulid.ID
	ExpiresAt         *int64
}

func MemoryFor(item agentevalgate.RetrievalItem) *agent.Memory {
	return &agent.Memory{
		Kind:     agent.MemoryKind(item.Kind),
		Source:   agent.MemorySourceUser,
		Status:   agent.MemoryStatusActive,
		Scope:    agent.MemoryScopeOrganization,
		ToolName: item.Tool,
		Content:  item.Content,
	}
}

func SeedMemories(t *testing.T, ctx context.Context, db *keel.DB, seed MemorySeed) *Corpus {
	t.Helper()

	corpus := newCorpus(seed.Tenant, len(seed.Items))
	for _, item := range seed.Items {
		memory := MemoryFor(item)
		memory.OrganizationID = seed.Tenant.OrgID
		memory.BusinessUnitID = seed.Tenant.BuID
		memory.ExpiresAt = seed.ExpiresAt
		if seed.Status != "" {
			memory.Status = seed.Status
		}
		if seed.Scope != "" {
			memory.Scope = seed.Scope
		}
		if seed.AgentDefinitionID.IsNotNil() {
			definitionID := seed.AgentDefinitionID
			memory.AgentDefinitionID = &definitionID
		}
		err := agentdb.Memories.Create(tenantContext(ctx, seed.Tenant), db, memory)
		require.NoError(t, err, item.Key)

		corpus.add(item.Key, memory.ID)
	}

	return corpus
}

type EmbedSeed struct {
	Repo       agentports.AIRetrievalRepository
	Sources    agentports.RetrievalSourceRepository
	Fixture    *agentevalgate.EmbeddingFixture
	ModelKey   string
	SourceType airetrieval.SourceType
	Corpus     *Corpus
}

func SeedEmbeddings(t *testing.T, ctx context.Context, seed EmbedSeed) {
	t.Helper()

	ctx = dbscope.WithTenant(ctx, seed.Corpus.Tenant.DBTenant())

	ids := make([]pulid.ID, 0, len(seed.Corpus.IDs))
	for _, id := range seed.Corpus.IDs {
		ids = append(ids, id)
	}

	chunksByID := chunkSources(t, ctx, seed, ids)
	for id, chunks := range chunksByID {
		embedded := make([]agentports.EmbeddingChunk, 0, len(chunks))
		for _, chunk := range chunks {
			vector, ok := seed.Fixture.Document(chunk.Hash)
			require.Truef(t, ok, "no recorded vector for %s chunk %d; re-record with:\n  %s",
				seed.Corpus.Keys[id], chunk.Index, RecordCommand)
			embedded = append(embedded, agentports.EmbeddingChunk{
				ChunkIndex:  chunk.Index,
				ContentHash: chunk.Hash,
				Vector:      vector,
			})
		}

		_, err := seed.Repo.ReplaceChunks(ctx, &agentports.ReplaceEmbeddingChunksRequest{
			Source: agentports.AIRetrievalSourceRef{
				TenantInfo: seed.Corpus.Tenant,
				SourceType: seed.SourceType,
				SourceID:   id,
			},
			ModelKey:   seed.ModelKey,
			Dimensions: seed.Fixture.Dimensions,
			Chunks:     embedded,
		})
		require.NoError(t, err)
	}
}

func chunkSources(
	t *testing.T,
	ctx context.Context,
	seed EmbedSeed,
	ids []pulid.ID,
) map[pulid.ID][]retrievalservice.Chunk {
	t.Helper()

	tenant := seed.Corpus.Tenant
	chunks := make(map[pulid.ID][]retrievalservice.Chunk, len(ids))

	switch seed.SourceType {
	case airetrieval.SourceTypeDocument:
		sources, err := seed.Sources.GetDocuments(ctx, &agentports.RetrievalDocumentsRequest{
			TenantInfo:   tenant,
			IDs:          ids,
			IncludePages: true,
		})
		require.NoError(t, err)
		for _, source := range sources {
			chunks[source.Document.ID] = retrievalservice.ChunkDocument(source)
		}
	case airetrieval.SourceTypeInboundMessage:
		messages, err := seed.Sources.GetInboundMessages(ctx, agentports.RetrievalSourcesRequest{
			TenantInfo: tenant,
			IDs:        ids,
		})
		require.NoError(t, err)
		for _, message := range messages {
			chunks[message.ID] = retrievalservice.ChunkEmail(message)
		}
	case airetrieval.SourceTypeMemory:
		memories, err := seed.Sources.GetMemories(ctx, agentports.RetrievalSourcesRequest{
			TenantInfo: tenant,
			IDs:        ids,
		})
		require.NoError(t, err)
		for _, memory := range memories {
			chunks[memory.ID] = retrievalservice.ChunkMemory(memory)
		}
	}
	require.Len(t, chunks, len(ids), "every seeded source reads back")

	return chunks
}

func tenantContext(ctx context.Context, tenant pagination.TenantInfo) context.Context {
	return dbscope.WithTenant(ctx, tenant.DBTenant())
}
