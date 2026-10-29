package memoryquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
)

var (
	memoryKinds = agenttoolschema.Source("agent.memoryKind", []agentmemory.MemoryKind{
		agentmemory.MemoryKindInstruction,
		agentmemory.MemoryKindFact,
		agentmemory.MemoryKindCorrection,
		agentmemory.MemoryKindProcedure,
	})
	memoryFilterSubjectTypes = agenttoolschema.Source(
		"agent.memoryFilterSubjectType",
		[]agentmemory.MemorySubjectType{
			agentmemory.MemorySubjectCustomer,
			agentmemory.MemorySubjectLocation,
			agentmemory.MemorySubjectWorker,
			agentmemory.MemorySubjectCarrier,
		},
	)
)
