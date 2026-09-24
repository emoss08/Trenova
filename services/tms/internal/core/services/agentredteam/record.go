package agentredteam

import (
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	ReadTool         = "tool"
	ReadInboundBox   = "inbound_desk"
	ReadMemoryRepo   = "memory_repository"
	ReadRunRepo      = "run_repository"
	ReadShipmentRepo = "shipment_repository"
	ReadCommentRepo  = "shipment_comment_repository"
	ReadPermissions  = "permissions"
	ReadCompletion   = "completion"
)

type ReadRecord struct {
	Kind   string
	Name   string
	Tenant pagination.TenantInfo
}

type DispatchRecord struct {
	Definition *agentdefinition.Definition
	Unattended bool
	Call       agentruntime.DispatchCall
	Held       []string
	Taint      *agent.RunTaint
	SourceRead bool
	Outcome    agentruntime.ToolOutcome
}

type ExecutionRecord struct {
	Tool       string
	Params     serviceports.ToolExecuteParams
	Egress     agent.EgressClass
	SourceRead bool
}

type DelegationRecord struct {
	Parent     *agentdefinition.Definition
	Call       agentruntime.DelegateCall
	SourceRead bool
	Declined   string
}

type MemoryRecord struct {
	Memory     *agent.Memory
	SourceRead bool
}

type OpenedTurn struct {
	Definition *agentdefinition.Definition
	System     string
	Taint      *agent.RunTaint
	Memories   []*agent.Memory
	Reopened   bool
}

type recorder struct {
	mu          sync.Mutex
	sourceRead  bool
	reads       []ReadRecord
	dispatches  []DispatchRecord
	executions  []ExecutionRecord
	delegations []DelegationRecord
	memories    []MemoryRecord
	opened      []OpenedTurn
}

func (r *recorder) read(kind, name string, tenant pagination.TenantInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reads = append(r.reads, ReadRecord{Kind: kind, Name: name, Tenant: tenant})
}

func (r *recorder) hasReadSource() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.sourceRead
}

func (r *recorder) markSourceRead() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sourceRead = true
}

func (r *recorder) dispatched(record *DispatchRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dispatches = append(r.dispatches, *record)
}

func (r *recorder) executed(record *ExecutionRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record.SourceRead = r.sourceRead
	r.executions = append(r.executions, *record)
}

func (r *recorder) delegated(record *DelegationRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.delegations = append(r.delegations, *record)
}

func (r *recorder) remembered(memory *agent.Memory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.memories = append(r.memories, MemoryRecord{Memory: memory, SourceRead: r.sourceRead})
}

func (r *recorder) openedTurn(turn OpenedTurn) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.opened = append(r.opened, turn)
}

func (r *recorder) outcome() *Outcome {
	r.mu.Lock()
	defer r.mu.Unlock()

	return &Outcome{
		Reads:       slices.Clone(r.reads),
		Dispatches:  slices.Clone(r.dispatches),
		Executions:  slices.Clone(r.executions),
		Delegations: slices.Clone(r.delegations),
		Memories:    slices.Clone(r.memories),
		Opened:      slices.Clone(r.opened),
	}
}
