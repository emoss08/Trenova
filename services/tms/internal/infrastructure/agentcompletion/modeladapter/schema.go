package modeladapter

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// DiagnosisSchemaName labels the diagnosis schema for protocols that require one.
const DiagnosisSchemaName = "agent_diagnosis"

// DiagnosisPayload is the decoded shape of a diagnosis reply.
type DiagnosisPayload struct {
	Proposals  []ProposalPayload  `json:"proposals"`
	Exceptions []ExceptionPayload `json:"exceptions"`
}

type ProposalPayload struct {
	ToolName   string              `json:"toolName"`
	ToolParams map[string]any      `json:"toolParams"`
	Confidence float64             `json:"confidence"`
	Rationale  string              `json:"rationale"`
	Evidence   []agent.EvidenceRef `json:"evidence"`
}

type ExceptionPayload struct {
	Category       string              `json:"category"`
	Severity       string              `json:"severity"`
	AttemptSummary string              `json:"attemptSummary"`
	Evidence       []agent.EvidenceRef `json:"evidence"`
	BlastRadius    int                 `json:"blastRadius"`
}

// ToDiagnoseResult converts a decoded payload into the port-level result.
func (p DiagnosisPayload) ToDiagnoseResult(model string) *serviceports.DiagnoseResult {
	result := &serviceports.DiagnoseResult{
		ModelIdentifier: model,
		Proposals:       make([]serviceports.ProposedAction, 0, len(p.Proposals)),
		Exceptions:      make([]serviceports.RaisedException, 0, len(p.Exceptions)),
	}

	for _, proposal := range p.Proposals {
		result.Proposals = append(result.Proposals, serviceports.ProposedAction{
			ToolName:   proposal.ToolName,
			ToolParams: proposal.ToolParams,
			Confidence: proposal.Confidence,
			Rationale:  proposal.Rationale,
			Evidence:   proposal.Evidence,
		})
	}

	for _, exception := range p.Exceptions {
		result.Exceptions = append(result.Exceptions, serviceports.RaisedException{
			Category:       exception.Category,
			Severity:       exception.Severity,
			AttemptSummary: exception.AttemptSummary,
			Evidence:       exception.Evidence,
			BlastRadius:    exception.BlastRadius,
		})
	}

	return result
}

// BuildDiagnosisSchema is the JSON Schema a diagnosis reply must satisfy.
func BuildDiagnosisSchema() map[string]any {
	evidenceItem := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type": map[string]any{"type": "string"},
			"id":   map[string]any{"type": "string"},
			"note": map[string]any{"type": "string"},
		},
		"required":             []string{"type", "id"},
		"additionalProperties": false,
	}

	evidenceArray := map[string]any{
		"type":     "array",
		"minItems": 1,
		"items":    evidenceItem,
	}

	proposal := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"toolName":   map[string]any{"type": "string"},
			"toolParams": map[string]any{"type": "object"},
			"confidence": map[string]any{"type": "number"},
			"rationale":  map[string]any{"type": "string"},
			"evidence":   evidenceArray,
		},
		"required": []string{
			"toolName",
			"toolParams",
			"confidence",
			"rationale",
			"evidence",
		},
		"additionalProperties": false,
	}

	exception := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category":       map[string]any{"type": "string"},
			"severity":       map[string]any{"type": "string"},
			"attemptSummary": map[string]any{"type": "string"},
			"blastRadius":    map[string]any{"type": "integer"},
			"evidence":       evidenceArray,
		},
		"required": []string{
			"category",
			"severity",
			"attemptSummary",
			"evidence",
		},
		"additionalProperties": false,
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"proposals":  map[string]any{"type": "array", "items": proposal},
			"exceptions": map[string]any{"type": "array", "items": exception},
		},
		"required":             []string{"proposals", "exceptions"},
		"additionalProperties": false,
	}
}
