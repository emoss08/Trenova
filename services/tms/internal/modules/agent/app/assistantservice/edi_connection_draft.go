package assistantservice

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	toolDraftEDIConnection = "draft_edi_connection"
	draftKeyTitle          = "title"
	draftKeyNote           = "note"
	draftKeySummary        = "summary"
)

var (
	connectionDraftOmitted = []string{draftKeyTitle, draftKeyNote}
	sampleOverviewKept     = []string{"transactionSet", "x12Version", draftKeySummary}
	connectionDraftTrimmed = []string{"mappings", "samples", "gaps"}
)

func connectionDraftArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	if typeutils.StringOfTrimmed(result["documentDigest"]) == "" {
		return nil
	}

	payload := maps.Clone(result)
	for _, key := range connectionDraftOmitted {
		delete(payload, key)
	}
	payload["samples"] = sampleSummaries(result["samples"])
	mappings, _ := result["mappings"].([]any)
	payload["mappingCount"] = len(mappings)
	payload["truncated"] = false
	for _, key := range connectionDraftTrimmed {
		if payloadSize(payload) <= assistantartifact.MaxPayloadBytes {
			break
		}
		fitRows(payload, key)
	}

	title := typeutils.StringOfTrimmed(result[draftKeyTitle])
	if title == "" {
		title = "EDI connection · " + typeutils.StringOfTrimmed(result["fileName"])
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindEDIConnectionDraft,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(title),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

func sampleSummaries(value any) []any {
	samples, _ := value.([]any)
	out := make([]any, 0, len(samples))
	for _, item := range samples {
		sample, ok := item.(map[string]any)
		if !ok {
			continue
		}
		summary := maps.Clone(sample)
		if overview, isObject := sample["overview"].(map[string]any); isObject {
			kept := make(map[string]any, len(sampleOverviewKept))
			for key, field := range overview {
				if slices.Contains(sampleOverviewKept, key) {
					kept[key] = field
				}
			}
			summary["overview"] = kept
		}
		out = append(out, summary)
	}
	return out
}
