package agentredteam

import (
	"context"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type inboundDesk struct {
	rec *recorder
}

func (d *inboundDesk) GetByID(
	_ context.Context,
	req repositories.GetInboundMessageByIDRequest,
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
	repositories.InboundMessageRepository

	desk *inboundDesk
}

func (r *inboundMessageRepository) GetByID(
	ctx context.Context,
	req repositories.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	return r.desk.GetByID(ctx, req)
}

func (d *inboundDesk) CheckLink(_ context.Context, req inboundmessageservice.LinkRequest) error {
	d.rec.read(ReadInboundBox, "CheckLink", req.TenantInfo)

	return nil
}

func (d *inboundDesk) Link(
	ctx context.Context,
	req inboundmessageservice.LinkRequest,
) (*inboundmessage.InboundMessage, error) {
	return d.GetByID(ctx, repositories.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
}

func (d *inboundDesk) Review(
	ctx context.Context,
	req inboundmessageservice.ReviewRequest,
) (*inboundmessage.InboundMessage, error) {
	return d.GetByID(ctx, repositories.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
}

type memoryRepository struct {
	repositories.AgentMemoryRepository

	rec      *recorder
	mu       sync.Mutex
	memories []*agent.Memory
}

func (r *memoryRepository) Create(_ context.Context, entity *agent.Memory) (*agent.Memory, error) {
	r.rec.read(ReadMemoryRepo, "Create", pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})

	created := *entity
	created.ID = pulid.MustNew("amem_")
	if created.Scope == "" {
		created.Scope = agent.MemoryScopeOrganization
	}

	r.mu.Lock()
	r.memories = append(r.memories, &created)
	r.mu.Unlock()
	r.rec.remembered(&created)

	return &created, nil
}

func (r *memoryRepository) FindActive(
	_ context.Context,
	req repositories.FindActiveAgentMemoryRequest,
) (*agent.Memory, error) {
	r.rec.read(ReadMemoryRepo, "FindActive", req.TenantInfo)

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, memory := range r.memories {
		if sameTenant(memory, req.TenantInfo) &&
			memory.Status == agent.MemoryStatusActive &&
			memory.Content == strings.TrimSpace(req.Content) &&
			memory.ToolName == req.ToolName {
			return memory, nil
		}
	}

	return nil, nil //nolint:nilnil // an absent memory is nil, nil by the repository contract
}

func (r *memoryRepository) GetByID(
	_ context.Context,
	req repositories.GetAgentMemoryByIDRequest,
) (*agent.Memory, error) {
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

func (r *memoryRepository) Search(
	_ context.Context,
	req repositories.SearchAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	r.rec.read(ReadMemoryRepo, "Search", req.TenantInfo)

	return nil, nil
}

func (r *memoryRepository) ListActive(
	_ context.Context,
	req repositories.ListActiveAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	r.rec.read(ReadMemoryRepo, "ListActive", req.TenantInfo)

	return nil, nil
}

func sameTenant(memory *agent.Memory, tenant pagination.TenantInfo) bool {
	return memory.OrganizationID == tenant.OrgID && memory.BusinessUnitID == tenant.BuID
}

type runRepository struct {
	repositories.AgentRunRepository

	rec *recorder
}

func (r *runRepository) GetByID(
	_ context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	tenant := pagination.TenantInfo{}
	if req.TenantInfo != nil {
		tenant = *req.TenantInfo
	}
	r.rec.read(ReadRunRepo, "GetByID", tenant)

	return nil, errortypes.NewNotFoundError("Agent run not found")
}

type extensionGate struct{}

func (extensionGate) ActiveExtensions(
	context.Context,
	pagination.TenantInfo,
) (map[agentextension.Type]agentextension.Availability, error) {
	types := agentextension.AllTypes()
	active := make(map[agentextension.Type]agentextension.Availability, len(types))
	for _, typ := range types {
		active[typ] = agentextension.AvailabilityAllAgents
	}

	return active, nil
}
