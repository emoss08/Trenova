//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/core/services/retrievalservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/airetrievalrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/retrievalsourcerepository"
	"github.com/emoss08/trenova/internal/testutil/retrievaltest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/reflectutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSearchToolsNeverLeakAndMarkEveryRecord(t *testing.T) {
	documents, err := agentevalgate.LoadRetrievalSuite(
		"../agentevalgate/evals/documentretrieval.yaml",
	)
	require.NoError(t, err)
	inbox, err := agentevalgate.LoadRetrievalSuite("../agentevalgate/evals/inboxretrieval.yaml")
	require.NoError(t, err)

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "LK")
	otherTenant := pagination.TenantInfo{OrgID: other.Organization.ID, BuID: other.BusinessUnit.ID}

	own := retrievaltest.SeedDocuments(t, ctx, db, retrievaltest.DocumentSeed{
		Tenant: tenant, UploadedByID: data.User.ID, Items: documents.Corpus,
	})
	sensitive := retrievaltest.SeedDocuments(t, ctx, db, retrievaltest.DocumentSeed{
		Tenant: tenant, UploadedByID: data.User.ID, OwnerType: "worker", Items: documents.Corpus,
	})
	foreignDocs := retrievaltest.SeedDocuments(t, ctx, db, retrievaltest.DocumentSeed{
		Tenant: otherTenant, UploadedByID: other.User.ID, Items: documents.Corpus,
	})
	ownMail := retrievaltest.SeedMessages(t, ctx, db, retrievaltest.MessageSeed{
		Tenant:    tenant,
		MailboxID: retrievaltest.SeedMailbox(t, ctx, db, tenant),
		Items:     inbox.Corpus,
	})
	foreignMail := retrievaltest.SeedMessages(t, ctx, db, retrievaltest.MessageSeed{
		Tenant:    otherTenant,
		MailboxID: retrievaltest.SeedMailbox(t, ctx, db, otherTenant),
		Items:     inbox.Corpus,
	})

	conn := postgres.NewTestConnection(db)
	repo := airetrievalrepository.New(airetrievalrepository.Params{
		DB:     conn,
		Probe:  postgres.NewTestCapabilityProbe(conn),
		Logger: zap.NewNop(),
	})
	sources := retrievalsourcerepository.New(retrievalsourcerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	var vectorizer serviceports.QueryVectorizer
	fixture, fixtureErr := agentevalgate.LoadEmbeddingFixture(retrievaltest.FixturePath)
	availability, err := repo.VectorAvailability(ctx)
	require.NoError(t, err)
	if fixtureErr == nil && availability.Available {
		for _, seeded := range []pagination.TenantInfo{tenant, otherTenant} {
			settings := airetrieval.DefaultSettings(seeded.OrgID, seeded.BuID)
			settings.ActiveModelKey = retrievaltest.ModelKey
			settings.Dimensions = airetrieval.Dimensions768
			_, err = repo.UpdateSettings(ctx, settings)
			require.NoError(t, err)
		}
		for _, corpus := range []*retrievaltest.Corpus{own, foreignDocs} {
			retrievaltest.SeedEmbeddings(t, ctx, retrievaltest.EmbedSeed{
				Repo: repo, Sources: sources, Fixture: fixture, ModelKey: retrievaltest.ModelKey,
				SourceType: airetrieval.SourceTypeDocument, Corpus: corpus,
			})
		}
		for _, corpus := range []*retrievaltest.Corpus{ownMail, foreignMail} {
			retrievaltest.SeedEmbeddings(t, ctx, retrievaltest.EmbedSeed{
				Repo: repo, Sources: sources, Fixture: fixture, ModelKey: retrievaltest.ModelKey,
				SourceType: airetrieval.SourceTypeInboundMessage, Corpus: corpus,
			})
		}
		vectorizer = retrievaltest.FixtureVectorizer{
			Fixture:  fixture,
			ModelKey: retrievaltest.ModelKey,
		}
	} else if !errors.Is(fixtureErr, agentevalgate.ErrEmbeddingFixtureMissing) && fixtureErr != nil {
		require.NoError(t, fixtureErr)
	}

	searcher := retrievalservice.NewSearcher(retrievalservice.SearcherParams{
		Logger:     zap.NewNop(),
		Repo:       repo,
		Sources:    sources,
		Registry:   permission.NewRegistry(),
		Vectorizer: vectorizer,
	})
	permissions := &readPermissions{readable: map[permission.Resource]bool{}}
	for _, resource := range []permission.Resource{
		permission.ResourceDocument,
		permission.ResourceInboundMessage,
		permission.ResourceShipment,
		permission.ResourceCustomer,
		permission.ResourceCarrier,
	} {
		permissions.readable[resource] = true
	}

	params := func(query string) serviceports.QueryToolParams {
		return serviceports.QueryToolParams{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Actor: &serviceports.RequestActor{
				PrincipalType:  serviceports.PrincipalTypeUser,
				PrincipalID:    data.User.ID,
				UserID:         data.User.ID,
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
			},
			Params: map[string]any{"query": query, "limit": 10},
		}
	}

	forbidden := make(map[string]struct{}, 64)
	for _, corpus := range []*retrievaltest.Corpus{sensitive, foreignDocs, foreignMail} {
		for id := range corpus.Keys {
			forbidden[id.String()] = struct{}{}
		}
	}

	tools := searchTools(t, searcher, permissions)
	queries := append(documents.Queries(), inbox.Queries()...)
	for _, query := range queries {
		assertNoLeak(t, leakCheck{
			tool:      tools["search_documents"],
			params:    params(query),
			idField:   "documentId",
			entity:    agent.TaintEntityDocument,
			own:       own,
			forbidden: forbidden,
		})
		assertNoLeak(t, leakCheck{
			tool:      tools["search_inbound_messages"],
			params:    params(query),
			idField:   "id",
			entity:    agent.TaintEntityInboundMessage,
			own:       ownMail,
			forbidden: forbidden,
		})
	}
}

func searchTools(
	t *testing.T,
	searcher serviceports.RetrievalSearcher,
	permissions serviceports.PermissionEngine,
) map[string]serviceports.AgentQueryTool {
	t.Helper()

	supplied := map[reflect.Type]reflect.Value{
		reflect.TypeFor[serviceports.RetrievalSearcher](): reflect.ValueOf(searcher),
		reflect.TypeFor[serviceports.PermissionEngine]():  reflect.ValueOf(permissions),
	}
	tools := make(map[string]serviceports.AgentQueryTool, 2)
	for _, provider := range agentquerytoolservice.ToolProviders() {
		built, err := reflectutils.Construct(provider, supplied)
		require.NoError(t, err)
		tool, ok := built.(serviceports.AgentQueryTool)
		if !ok {
			continue
		}
		if name := tool.Name(); name == "search_documents" || name == "search_inbound_messages" {
			tools[name] = tool
		}
	}
	require.Len(t, tools, 2)

	return tools
}

type leakCheck struct {
	tool      serviceports.AgentQueryTool
	params    serviceports.QueryToolParams
	idField   string
	entity    string
	own       *retrievaltest.Corpus
	forbidden map[string]struct{}
}

func assertNoLeak(t *testing.T, check leakCheck) {
	t.Helper()

	result, err := check.tool.Query(t.Context(), check.params)
	require.NoError(t, err)

	carrier, ok := result.(agent.TaintCarrier)
	require.True(t, ok, "%s names the records it returns", check.tool.Name())
	marked := taintedIDs(carrier.TaintedRecords(), check.entity)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	var decoded struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))

	query := check.params.Params["query"]
	for _, row := range decoded.Items {
		id, _ := row[check.idField].(string)
		require.NotEmpty(t, id)
		assert.NotContains(t, check.forbidden, id, "%s for %q", check.tool.Name(), query)
		assert.Contains(t, check.own.Keys, pulid.ID(id), "%s for %q", check.tool.Name(), query)
		assert.Contains(t, marked, id, "every record %s returns taints the turn", check.tool.Name())
	}
}

func taintedIDs(refs []agent.RecordRef, entity string) map[string]struct{} {
	ids := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.EntityType == entity {
			ids[ref.ID] = struct{}{}
		}
	}

	return ids
}

type readPermissions struct {
	serviceports.PermissionEngine

	readable map[permission.Resource]bool
}

func (p *readPermissions) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	return &serviceports.PermissionCheckResult{
		Allowed: req.Operation == permission.OpRead &&
			p.readable[permission.Resource(req.Resource)],
	}, nil
}

func (p *readPermissions) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*serviceports.ResourcePermissionDetail, error) {
	detail := &serviceports.ResourcePermissionDetail{Resource: resource}
	if p.readable[permission.Resource(resource)] {
		detail.Operations = []permission.Operation{permission.OpRead}
		detail.MaxSensitivity = permission.SensitivityRestricted
	}

	return detail, nil
}
