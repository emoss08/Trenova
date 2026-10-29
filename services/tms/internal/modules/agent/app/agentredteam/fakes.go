package agentredteam

import (
	"context"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type inboundDesk struct {
	rec *recorder
}

func (d *inboundDesk) GetByID(
	_ context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	d.rec.read(ReadInboundBox, "GetByID", req.TenantInfo)

	return &inboundmessage.InboundMessage{
		ID:             req.ID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		FromAddress:    "billing@remit-partner.example",
		Subject:        "Remittance advice",
		Status:         inboundmessage.StatusClassified,
		Confidence:     0.99,
		Mailbox: &inboundmessage.Mailbox{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			ReviewPolicy:   inboundmessage.ReviewAutoHandle,
		},
	}, nil
}

type inboundMessageRepository struct {
	inboundmessageports.InboundMessageRepository

	desk *inboundDesk
}

func (r *inboundMessageRepository) GetByID(
	ctx context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	return r.desk.GetByID(ctx, req)
}

func (d *inboundDesk) CheckLink(_ context.Context, req *inboundqueue.LinkRequest) error {
	d.rec.read(ReadInboundBox, "CheckLink", req.TenantInfo)

	return nil
}

func (d *inboundDesk) Link(
	ctx context.Context,
	req *inboundqueue.LinkRequest,
) (*inboundmessage.InboundMessage, error) {
	return d.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
}

func (d *inboundDesk) Review(
	ctx context.Context,
	req *inboundqueue.ReviewRequest,
) (*inboundmessage.InboundMessage, error) {
	return d.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
}

type memoryRepository struct {
	agentmemoryports.AgentMemoryRepository

	rec      *recorder
	mu       sync.Mutex
	memories []*agentmemory.Memory
}

func (r *memoryRepository) Create(
	_ context.Context,
	entity *agentmemory.Memory,
) (*agentmemory.Memory, error) {
	r.rec.read(ReadMemoryRepo, "Create", pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})

	created := *entity
	created.ID = pulid.MustNew("amem_")
	if created.Scope == "" {
		created.Scope = agentmemory.MemoryScopeOrganization
	}

	r.mu.Lock()
	r.memories = append(r.memories, &created)
	r.mu.Unlock()
	r.rec.remembered(&created)

	return &created, nil
}

//nolint:gocritic // agentmemoryports.AgentMemoryRepository takes the request by value
func (r *memoryRepository) FindActive(
	_ context.Context,
	req *agentmemoryports.FindActiveAgentMemoryRequest,
) (*agentmemory.Memory, error) {
	r.rec.read(ReadMemoryRepo, "FindActive", req.TenantInfo)

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, memory := range r.memories {
		if sameTenant(memory, req.TenantInfo) &&
			memory.Status == agentmemory.MemoryStatusActive &&
			memory.Content == strings.TrimSpace(req.Content) &&
			memory.ToolName == req.ToolName {
			return memory, nil
		}
	}

	return nil, nil //nolint:nilnil // an absent memory is nil, nil by the repository contract
}

// MarkUsed is reached when a remember restates a memory already kept and the
// service refreshes it. The service counts the use on the row FindActive
// handed it, which here is the stored row itself, so the fake only records
// which tenant was touched.
//
//nolint:gocritic // agentmemoryports.AgentMemoryRepository takes the request by value
func (r *memoryRepository) MarkUsed(
	_ context.Context,
	req *agentmemoryports.MarkAgentMemoriesUsedRequest,
) error {
	r.rec.read(ReadMemoryRepo, "MarkUsed", req.TenantInfo)

	return nil
}

// GetPreference is reached before every remember, to learn whether the person
// asked to approve saves first. No one in the harness has chosen, so saves run
// as they do by default.
func (r *memoryRepository) GetPreference(
	_ context.Context,
	req agentmemoryports.GetAgentMemoryPreferenceRequest,
) (*agentmemory.MemoryPreference, error) {
	r.rec.read(ReadMemoryRepo, "GetPreference", req.TenantInfo)

	return nil, nil //nolint:nilnil // a person who never chose has no preference row
}

func (r *memoryRepository) GetByID(
	_ context.Context,
	req agentmemoryports.GetAgentMemoryByIDRequest,
) (*agentmemory.Memory, error) {
	r.rec.read(ReadMemoryRepo, "GetByID", req.TenantInfo)

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, memory := range r.memories {
		if memory.ID == req.ID && sameTenant(memory, req.TenantInfo) {
			return memory, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Agent memory not found")
}

//nolint:gocritic // agentmemoryports.AgentMemoryRepository takes the request by value
func (r *memoryRepository) Search(
	_ context.Context,
	req *agentmemoryports.SearchAgentMemoriesRequest,
) ([]*agentmemory.Memory, error) {
	r.rec.read(ReadMemoryRepo, "Search", req.TenantInfo)

	return nil, nil
}

//nolint:gocritic // agentmemoryports.AgentMemoryRepository takes the request by value
func (r *memoryRepository) ListActive(
	_ context.Context,
	req *agentmemoryports.ListActiveAgentMemoriesRequest,
) ([]*agentmemory.Memory, error) {
	r.rec.read(ReadMemoryRepo, "ListActive", req.TenantInfo)

	return nil, nil
}

func sameTenant(memory *agentmemory.Memory, tenant pagination.TenantInfo) bool {
	return memory.OrganizationID == tenant.OrgID && memory.BusinessUnitID == tenant.BuID
}

type runRepository struct {
	agentruntimeports.AgentRunRepository

	rec *recorder
}

func (r *runRepository) GetByID(
	_ context.Context,
	req agentruntimeports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	tenant := pagination.TenantInfo{}
	if req.TenantInfo != nil {
		tenant = *req.TenantInfo
	}
	r.rec.read(ReadRunRepo, "GetByID", tenant)

	return nil, errortypes.NewNotFoundError("Agent run not found")
}

type extensionGate struct{}

func (extensionGate) ActiveCapabilities(
	context.Context,
	pagination.TenantInfo,
) (agentextension.Active, error) {
	capabilities := agentextension.AllCapabilities()
	active := make(agentextension.Active, len(capabilities))
	for _, capability := range capabilities {
		types := agentextension.TypesFor(capability)
		if len(types) == 0 {
			continue
		}
		active[capability] = agentextension.ActiveExtension{
			Type:         types[0],
			Availability: agentextension.AvailabilityAllAgents,
		}
	}

	return active, nil
}
