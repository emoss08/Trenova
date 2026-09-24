package retrievaltest

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/emoss08/trenova/internal/core/services/retrievalservice"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
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
) *repositories.RetrievalDocumentSource {
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

	return &repositories.RetrievalDocumentSource{Document: doc, Content: content, Pages: pages}
}

func SeedDocuments(t *testing.T, ctx context.Context, db bun.IDB, seed DocumentSeed) *Corpus {
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
		_, err := db.NewInsert().Model(doc).Exec(ctx)
		require.NoError(t, err, item.Key)

		content := source.Content
		content.DocumentID = doc.ID
		content.OrganizationID = seed.Tenant.OrgID
		content.BusinessUnitID = seed.Tenant.BuID
		_, err = db.NewInsert().Model(content).Exec(ctx)
		require.NoError(t, err, item.Key)

		for _, page := range source.Pages {
			page.DocumentContentID = content.ID
			page.DocumentID = doc.ID
			page.OrganizationID = seed.Tenant.OrgID
			page.BusinessUnitID = seed.Tenant.BuID
		}
		if len(source.Pages) > 0 {
			_, err = db.NewInsert().Model(&source.Pages).Exec(ctx)
			require.NoError(t, err, item.Key)
		}

		corpus.add(item.Key, doc.ID)
	}

	return corpus
}

func SeedColleague(
	t *testing.T,
	ctx context.Context,
	db bun.IDB,
	tenant pagination.TenantInfo,
) pulid.ID {
	t.Helper()

	id := strings.ToLower(pulid.MustNew("usr_").String())
	marker := "c" + id[len(id)-12:]
	var colleague pulid.ID
	require.NoError(t, db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		colleague = seedtest.NewUser(tenant.OrgID, tenant.BuID).
			WithName("Colleague").
			WithUsername(marker).
			WithEmail(marker + "@example.com").
			Build(t, ctx, tx).ID

		return nil
	}))

	return colleague
}

func SeedConversation(
	t *testing.T,
	ctx context.Context,
	db bun.IDB,
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
	_, err := db.NewInsert().Model(thread).Exec(ctx)
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
	db bun.IDB,
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
	_, err := db.NewInsert().Model(mailbox).Exec(ctx)
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

func SeedMessages(t *testing.T, ctx context.Context, db bun.IDB, seed MessageSeed) *Corpus {
	t.Helper()

	corpus := newCorpus(seed.Tenant, len(seed.Items))
	for idx, item := range seed.Items {
		message := MessageFor(item)
		message.OrganizationID = seed.Tenant.OrgID
		message.BusinessUnitID = seed.Tenant.BuID
		message.MailboxID = seed.MailboxID
		message.ReceivedAt = int64(1_790_000_000 + idx*60)
		_, err := db.NewInsert().Model(message).Exec(ctx)
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

func SeedMemories(t *testing.T, ctx context.Context, db bun.IDB, seed MemorySeed) *Corpus {
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
		_, err := db.NewInsert().Model(memory).Exec(ctx)
		require.NoError(t, err, item.Key)

		corpus.add(item.Key, memory.ID)
	}

	return corpus
}

type EmbedSeed struct {
	Repo       repositories.AIRetrievalRepository
	Sources    repositories.RetrievalSourceRepository
	Fixture    *agentevalgate.EmbeddingFixture
	ModelKey   string
	SourceType airetrieval.SourceType
	Corpus     *Corpus
}

func SeedEmbeddings(t *testing.T, ctx context.Context, seed EmbedSeed) {
	t.Helper()

	ids := make([]pulid.ID, 0, len(seed.Corpus.IDs))
	for _, id := range seed.Corpus.IDs {
		ids = append(ids, id)
	}

	chunksByID := chunkSources(t, ctx, seed, ids)
	for id, chunks := range chunksByID {
		embedded := make([]repositories.EmbeddingChunk, 0, len(chunks))
		for _, chunk := range chunks {
			vector, ok := seed.Fixture.Document(chunk.Hash)
			require.Truef(t, ok, "no recorded vector for %s chunk %d; re-record with:\n  %s",
				seed.Corpus.Keys[id], chunk.Index, RecordCommand)
			embedded = append(embedded, repositories.EmbeddingChunk{
				ChunkIndex:  chunk.Index,
				ContentHash: chunk.Hash,
				Vector:      vector,
			})
		}

		_, err := seed.Repo.ReplaceChunks(ctx, repositories.ReplaceEmbeddingChunksRequest{
			Source: repositories.AIRetrievalSourceRef{
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
		sources, err := seed.Sources.GetDocuments(ctx, repositories.RetrievalDocumentsRequest{
			TenantInfo:   tenant,
			IDs:          ids,
			IncludePages: true,
		})
		require.NoError(t, err)
		for _, source := range sources {
			chunks[source.Document.ID] = retrievalservice.ChunkDocument(source)
		}
	case airetrieval.SourceTypeInboundMessage:
		messages, err := seed.Sources.GetInboundMessages(ctx, repositories.RetrievalSourcesRequest{
			TenantInfo: tenant,
			IDs:        ids,
		})
		require.NoError(t, err)
		for _, message := range messages {
			chunks[message.ID] = retrievalservice.ChunkEmail(message)
		}
	case airetrieval.SourceTypeMemory:
		memories, err := seed.Sources.GetMemories(ctx, repositories.RetrievalSourcesRequest{
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
