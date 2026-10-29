package agentextensiontest

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentextension/exaconnector"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const (
	Now    = int64(1_790_150_400)
	APIKey = "exa-live-key"
)

type MemoryRepo struct {
	mu      sync.Mutex
	records map[string]*agentextension.Extension
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{records: map[string]*agentextension.Extension{}}
}

func repoKey(tenantInfo pagination.TenantInfo, typ agentextension.Type) string {
	return tenantInfo.OrgID.String() + "|" + tenantInfo.BuID.String() + "|" + typ.String()
}

func (r *MemoryRepo) ListByTenant(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*agentextension.Extension, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*agentextension.Extension, 0, len(r.records))
	for _, record := range r.records {
		if record.OrganizationID == tenantInfo.OrgID && record.BusinessUnitID == tenantInfo.BuID {
			clone := *record
			out = append(out, &clone)
		}
	}

	return out, nil
}

func (r *MemoryRepo) GetByType(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*agentextension.Extension, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[repoKey(tenantInfo, typ)]
	if !ok {
		return nil, errortypes.NewNotFoundError(
			"Agent extension not found within your organization",
		)
	}
	clone := *record

	return &clone, nil
}

func (r *MemoryRepo) Create(
	_ context.Context,
	entity *agentextension.Extension,
) (*agentextension.Extension, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := repoKey(
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.Type,
	)
	if _, exists := r.records[key]; exists {
		return nil, errortypes.NewConflictError("exists")
	}
	entity.ID = pulid.MustNew("aext_")
	entity.UpdatedAt = Now
	clone := *entity
	r.records[key] = &clone

	return entity, nil
}

func (r *MemoryRepo) Update(
	_ context.Context,
	entity *agentextension.Extension,
) (*agentextension.Extension, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := repoKey(
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.Type,
	)
	current, ok := r.records[key]
	if !ok || current.Version != entity.Version {
		return nil, errortypes.NewConflictError("version mismatch")
	}
	entity.Version++
	entity.UpdatedAt = Now
	clone := *entity
	r.records[key] = &clone

	return entity, nil
}

type MemoryUsage struct {
	mu       sync.Mutex
	Requests int
	Failures int
	Cost     decimal.Decimal
}

func (u *MemoryUsage) Reserve(
	_ context.Context,
	params agentextensionports.ReserveExtensionRequestParams,
) (bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.Requests >= params.Limit {
		return false, nil
	}
	u.Requests++

	return true, nil
}

func (u *MemoryUsage) RecordOutcome(
	_ context.Context,
	params agentextensionports.RecordExtensionOutcomeParams,
) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if params.Failed {
		u.Failures++
	}
	u.Cost = u.Cost.Add(params.CostUSD)

	return nil
}

func (u *MemoryUsage) Summarize(
	context.Context,
	agentextensionports.SummarizeExtensionUsageParams,
) (map[agentextension.Type]agentextension.UsageSummary, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	return map[agentextension.Type]agentextension.UsageSummary{
		agentextension.TypeExa: {
			RequestsToday:     u.Requests,
			RequestsThisMonth: u.Requests,
			FailuresThisMonth: u.Failures,
			CostThisMonthUSD:  u.Cost,
		},
	}, nil
}

type FakeWeb struct {
	Kind        agentextension.Type
	mu          sync.Mutex
	Searches    []*agentextensionports.WebSearchProviderRequest
	Reads       []agentextensionports.WebPageProviderRequest
	Results     []agentextensionports.WebSearchProviderResult
	PageText    string
	SearchErr   error
	ReadErr     error
	SearchCost  decimal.Decimal
	ReadCost    decimal.Decimal
	MissingPage bool
}

func (f *FakeWeb) Type() agentextension.Type { return f.Kind }

func (f *FakeWeb) Classify(err error) agentextensionports.ProviderFailure {
	return exaconnector.New().Classify(err)
}

func (f *FakeWeb) Test(
	ctx context.Context,
	settings map[string]string,
) (*agentextensionports.ConnectorTestResult, error) {
	resp, err := f.Search(ctx, &agentextensionports.WebSearchProviderRequest{
		Settings:   settings,
		Query:      "test",
		NumResults: 1,
	})
	if err != nil {
		return nil, err
	}

	return &agentextensionports.ConnectorTestResult{CostUSD: resp.CostUSD}, nil
}

func (f *FakeWeb) Search(
	_ context.Context,
	req *agentextensionports.WebSearchProviderRequest,
) (*agentextensionports.WebSearchProviderResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Searches = append(f.Searches, req)
	if f.SearchErr != nil {
		return nil, f.SearchErr
	}

	return &agentextensionports.WebSearchProviderResponse{
		Results: f.Results,
		CostUSD: f.SearchCost,
	}, nil
}

func (f *FakeWeb) Read(
	_ context.Context,
	req agentextensionports.WebPageProviderRequest,
) (*agentextensionports.WebPageProviderResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Reads = append(f.Reads, req)
	if f.ReadErr != nil {
		return nil, f.ReadErr
	}
	if f.MissingPage {
		return &agentextensionports.WebPageProviderResponse{
			URL:     req.URL,
			CostUSD: f.ReadCost,
		}, exa.ErrContentNotFound
	}

	text := f.PageText
	if runes := []rune(text); len(runes) > req.MaxCharacters {
		text = string(runes[:req.MaxCharacters])
	}

	return &agentextensionports.WebPageProviderResponse{
		Title:         "Hours of Service",
		URL:           req.URL,
		PublishedDate: "2024-03-01T00:00:00.000Z",
		Text:          text,
		CostUSD:       f.ReadCost,
	}, nil
}

type RecordingAudit struct {
	auditports.AuditService

	mu      sync.Mutex
	Entries []*auditports.LogActionParams
}

func (a *RecordingAudit) LogAction(
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.Entries = append(a.Entries, params)

	return nil
}

func (a *RecordingAudit) List(
	context.Context,
	*auditports.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (a *RecordingAudit) RegisterSensitiveFields(
	permission.Resource,
	[]auditports.SensitiveField,
) error {
	return nil
}

func EncryptionService() *encryptionservice.Service {
	return encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("unit-test-encryption-key-with-at-least-32-bytes"),
	)
}

type extensionConfigs interface {
	GetConfig(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ agentextension.Type,
	) (*agentextensionports.AgentExtensionConfigResponse, error)
	UpdateConfig(
		ctx context.Context,
		typ agentextension.Type,
		req *agentextensionports.UpdateAgentExtensionRequest,
	) (*agentextensionports.AgentExtensionConfigResponse, error)
}

func EnableType(
	t *testing.T,
	extensions extensionConfigs,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
	configuration map[string]string,
) *agentextensionports.AgentExtensionConfigResponse {
	t.Helper()

	current, err := extensions.GetConfig(t.Context(), tenantInfo, typ)
	require.NoError(t, err)

	resp, err := extensions.UpdateConfig(
		t.Context(),
		typ,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo:    tenantInfo,
			Enabled:       true,
			Availability:  agentextension.AvailabilityAllAgents,
			Configuration: configuration,
			Version:       current.Version,
		},
	)
	require.NoError(t, err)

	return resp
}
