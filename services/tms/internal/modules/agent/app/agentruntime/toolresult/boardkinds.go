package toolresult

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
)

const shipmentNumberKey = "shipmentProNumber"

func returnedRef(record map[string]any) (agentpagecontext.EntityRef, bool) {
	if ref, ok := recordRef(record); ok {
		return ref, true
	}
	id, ok := record["id"].(string)
	if !ok {
		return agentpagecontext.EntityRef{}, false
	}
	separator := strings.IndexByte(id, '_')
	if separator <= 0 {
		return agentpagecontext.EntityRef{}, false
	}
	kind, ok := recordkind.OfIDPrefix(id[:separator+1])
	if !ok || !boardregistry.Catalog().Supports(kind) {
		return agentpagecontext.EntityRef{}, false
	}
	label := citedRecordLabel(kind.Resource(), record)
	if label == "" {
		label = strings.TrimSpace(assistantartifact.ReadableString(record[shipmentNumberKey]))
	}
	if label == "" {
		label = wholeLabel(kind, record)
	}

	return agentpagecontext.EntityRef{Type: string(kind), ID: id, Label: label}, true
}

func wholeLabel(kind recordkind.Kind, record map[string]any) string {
	spec, ok := boardregistry.Catalog().KindOf(kind)
	if !ok || spec.PartOf == "" {
		return ""
	}
	for _, key := range slices.Sorted(maps.Keys(record)) {
		whole, isRecord := record[key].(map[string]any)
		if !isRecord {
			continue
		}
		id, _ := whole["id"].(string)
		separator := strings.IndexByte(id, '_')
		if separator <= 0 {
			continue
		}
		if of, known := recordkind.OfIDPrefix(id[:separator+1]); known && of == spec.PartOf {
			return citedRecordLabel(of.Resource(), whole)
		}
	}

	return ""
}

const howToShowKey = "howToShow"

func HowToShow(content string) string {
	decoded, ok := decodeResult(content)
	if !ok {
		return ""
	}
	record, isRecord := decoded.(map[string]any)
	if !isRecord {
		return ""
	}
	hint, _ := record[howToShowKey].(string)

	return strings.TrimSpace(hint)
}
