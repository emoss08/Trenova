package retrievaltest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/emoss08/trenova/internal/core/services/retrievalservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	FixturePath    = "internal/core/services/agentevalgate/" + agentevalgate.RetrievalEmbeddingFixturePath
	ModelKey       = "eval.local/" + agentevalgate.EmbeddingFixtureModel + "@768"
	DocumentSuite  = "internal/core/services/agentevalgate/evals/documentretrieval.yaml"
	InboxSuite     = "internal/core/services/agentevalgate/evals/inboxretrieval.yaml"
	MemorySuite    = "internal/core/services/agentevalgate/evals/memoryretrieval.yaml"
	DocumentFloors = "internal/core/services/agentevalgate/evals/documentretrieval.floors.json"
	InboxFloors    = "internal/core/services/agentevalgate/evals/inboxretrieval.floors.json"
	MemoryFloors   = "internal/core/services/agentevalgate/evals/memoryretrieval.floors.json"

	RecordCommand = "cd services/tms && ollama pull nomic-embed-text && " +
		agentevalgate.OllamaURLEnv + "=" + agentevalgate.DefaultOllamaURL +
		" go test -tags nofitz -count=1 -run 'TestRecordRetrievalEmbeddingFixture' " +
		"./internal/core/services/retrievalservice/ -record"
	FloorsCommand = "cd services/tms && " +
		"TRENOVA_TEST_POSTGRES_IMAGE=trenova-postgres:local go test -tags 'integration nofitz' " +
		"-count=1 -run 'TestRetrievalAgainstFloors' ./internal/core/services/retrievalservice/ -update"
)

func EmbeddingInputs(
	documents, inbox, memories *agentevalgate.RetrievalSuite,
) agentevalgate.EmbeddingInputs {
	inputs := agentevalgate.EmbeddingInputs{
		Documents: make([]serviceports.EmbeddingCatalogItem, 0, 64),
		Queries:   make([]string, 0, 64),
	}

	add := func(key string, chunks []retrievalservice.Chunk) {
		for _, chunk := range chunks {
			inputs.Documents = append(inputs.Documents, serviceports.EmbeddingCatalogItem{
				Key:         key + "#" + strconv.Itoa(chunk.Index),
				Text:        chunk.Text,
				ContentHash: chunk.Hash,
			})
		}
	}

	for _, item := range documents.Corpus {
		add(item.Key, retrievalservice.ChunkDocument(DocumentFor(item, "")))
	}
	for _, item := range inbox.Corpus {
		add(item.Key, retrievalservice.ChunkEmail(MessageFor(item)))
	}
	for _, item := range memories.Corpus {
		add(item.Key, retrievalservice.ChunkMemory(MemoryFor(item)))
	}

	for _, suite := range []*agentevalgate.RetrievalSuite{documents, inbox, memories} {
		inputs.Queries = append(inputs.Queries, suite.Queries()...)
	}

	return inputs
}

type FixtureVectorizer struct {
	Fixture  *agentevalgate.EmbeddingFixture
	ModelKey string
}

var _ serviceports.QueryVectorizer = FixtureVectorizer{}

func (v FixtureVectorizer) Vectorize(
	_ context.Context,
	req serviceports.QueryVectorRequest,
) (serviceports.QueryVector, error) {
	vector, ok := v.Fixture.Query(req.Text)
	if !ok {
		return serviceports.QueryVector{}, fmt.Errorf(
			"no recorded vector for query %q; re-record with:\n  %s", req.Text, RecordCommand)
	}

	return serviceports.QueryVector{
		Available:  true,
		Vector:     vector,
		ModelKey:   v.ModelKey,
		Dimensions: v.Fixture.Dimensions,
	}, nil
}

func (v FixtureVectorizer) Availability(
	context.Context,
	pagination.TenantInfo,
) (airetrieval.Availability, error) {
	return airetrieval.Availability{Available: true, ExtensionInstalled: true}, nil
}

type Access struct {
	Unreadable map[permission.Resource]bool
	Ceiling    permission.FieldSensitivity
	Registry   *permission.Registry
	Tenant     pagination.TenantInfo
	UserID     pulid.ID
	Threads    repositories.ThreadOwnerRepository
}

var _ serviceports.RetrievalAccess = Access{}

func (a Access) MayReadResource(_ context.Context, resource permission.Resource) bool {
	return !a.Unreadable[resource]
}

func (a Access) MayReadRecord(ctx context.Context, resource permission.Resource, _ string) bool {
	return a.MayReadResource(ctx, resource)
}

func (a Access) ReadableDocuments(
	ctx context.Context,
	docs []*document.Document,
) (map[pulid.ID]bool, error) {
	var owners map[pulid.ID]pulid.ID
	if ids := document.ConversationIDs(docs); len(ids) > 0 && a.Threads != nil &&
		a.UserID.IsNotNil() {
		found, err := a.Threads.ThreadOwners(ctx, repositories.ThreadOwnersRequest{
			TenantInfo: a.Tenant,
			ThreadIDs:  ids,
		})
		if err != nil {
			return nil, err
		}
		owners = found
	}

	readable := make(map[pulid.ID]bool, len(docs))
	for _, doc := range docs {
		if doc.VisibleToPerson(owners, a.UserID) &&
			a.MayReadRecord(ctx, doc.OwnerResource(), doc.ResourceID) {
			readable[doc.ID] = true
		}
	}

	return readable, nil
}

func (a Access) ceiling() permission.FieldSensitivity {
	if a.Ceiling == "" {
		return permission.SensitivityRestricted
	}

	return a.Ceiling
}

func (a Access) ShowsField(_ context.Context, resource permission.Resource, field string) bool {
	if a.Registry == nil {
		return true
	}

	return a.ceiling().CanAccess(a.Registry.GetFieldSensitivity(resource.String(), field))
}

func (a Access) ShowsRecordText(_ context.Context, resource permission.Resource) bool {
	if a.Registry == nil {
		return true
	}
	definition, ok := a.Registry.Get(resource.String())

	return ok && a.ceiling().CanAccess(definition.DefaultSensitivity)
}
