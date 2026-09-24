package shipmentimportassistantservice

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingAILogRepo struct {
	repoports.AILogRepository
	created []*ailog.Log
}

func (r *recordingAILogRepo) Create(_ context.Context, entry *ailog.Log) (*ailog.Log, error) {
	r.created = append(r.created, entry)
	return entry, nil
}

func TestAddLocation_CutsTheCodeOnACharacterBoundary(t *testing.T) {
	t.Parallel()

	tenant := importTenant()
	engine := mocks.NewMockPermissionEngine(t)
	expectCheck(engine, tenant, permission.ResourceLocation, permission.OpCreate, true)

	states := mocks.NewMockUsStateRepository(t)
	states.EXPECT().
		GetByAbbreviation(mock.Anything, "QC").
		Return(&usstate.UsState{ID: pulid.MustNew("us_")}, nil).
		Once()

	categories := mocks.NewMockLocationCategoryRepository(t)
	categories.EXPECT().
		SelectOptions(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*locationcategory.LocationCategory]{
			Items: []*locationcategory.LocationCategory{{ID: pulid.MustNew("lc_")}},
			Total: 1,
		}, nil).
		Once()

	var created *location.Location
	transformer := mocks.NewMockDataTransformer(t)
	transformer.EXPECT().
		TransformLocation(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *location.Location) error {
			created = entity
			return errors.New("stop before saving")
		}).
		Once()

	service := &Service{
		logger:               zap.NewNop(),
		permissions:          engine,
		usStateRepo:          states,
		locationCategoryRepo: categories,
		locationService: locationservice.New(locationservice.Params{
			Logger:      zap.NewNop(),
			Transformer: transformer,
		}),
	}

	service.RunTool(t.Context(), tenant, &serviceports.ToolCall{
		ID:   "call_1",
		Name: "add_location",
		Arguments: map[string]any{
			"name":          "Entrepôt Montréal Nord",
			"address_line1": "1 rue Principale",
			"city":          "Montréal",
			"state_abbrev":  "QC",
			"postal_code":   "H1A 0A1",
		},
	})

	require.NotNil(t, created)
	assert.True(t, utf8.ValidString(created.Code))
	assert.Equal(t, "Entrepôt M", created.Code)
}

func TestLogAICall_KeepsPreviewsOnCharacterBoundaries(t *testing.T) {
	t.Parallel()

	logs := &recordingAILogRepo{}
	service := &Service{logger: zap.NewNop(), aiLogRepo: logs}

	message := strings.Repeat("a", 511) + "é" + strings.Repeat("b", 10)
	reply := strings.Repeat("c", 1023) + "日本" + strings.Repeat("d", 10)

	service.logAICall(t.Context(), &serviceports.ShipmentImportChatRequest{
		TenantInfo:  importTenant(),
		UserMessage: message,
		DocumentID:  pulid.MustNew("doc_").String(),
	}, &TurnRecord{Message: reply, Model: "test-model"})

	require.Len(t, logs.created, 1)
	entry := logs.created[0]
	assert.True(t, utf8.ValidString(entry.Prompt))
	assert.True(t, utf8.ValidString(entry.Response))
	assert.True(t, strings.HasSuffix(entry.Prompt, strings.Repeat("a", 511)+"é"))
	assert.True(t, strings.HasSuffix(entry.Response, strings.Repeat("c", 1023)+"日"))
}

func TestLogAICall_RecordsTheModelAndTokensTheTurnUsed(t *testing.T) {
	t.Parallel()

	logs := &recordingAILogRepo{}
	service := &Service{logger: zap.NewNop(), aiLogRepo: logs}
	providerID := pulid.MustNew("aip_")

	service.logAICall(t.Context(), &serviceports.ShipmentImportChatRequest{
		TenantInfo:  importTenant(),
		UserMessage: "Set the customer to Acme",
		DocumentID:  pulid.MustNew("doc_").String(),
	}, &TurnRecord{
		Message:         "Found Acme Freight.",
		Model:           "qwen2.5:32b",
		ProviderID:      providerID,
		ProviderKind:    aiprovider.KindOllama,
		InputTokens:     1200,
		OutputTokens:    80,
		ReasoningTokens: 20,
	})

	require.Len(t, logs.created, 1)
	entry := logs.created[0]
	assert.Equal(t, ailog.Model("qwen2.5:32b"), entry.Model)
	assert.Equal(t, providerID, entry.ProviderID)
	assert.Equal(t, string(aiprovider.KindOllama), entry.ProviderKind)
	assert.Equal(t, 1200, entry.PromptTokens)
	assert.Equal(t, 80, entry.CompletionTokens)
	assert.Equal(t, 1280, entry.TotalTokens)
	assert.Equal(t, 20, entry.ReasoningTokens)
}
