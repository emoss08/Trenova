package toolresult

import (
	"maps"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/productguide"
)

const (
	maxCitedRecords      = 24
	maxReturnedRecords   = 200
	maxCitedRecordDepth  = 6
	maxCitedContentBytes = 512 << 10
)

var documentLabelKeys = []string{"originalName", "fileName"}

// shipmentLabelKey is a shipment's PRO number. It names a shipment; on any
// other record (an invoice, a carrier invoice) it is the shipment that record
// belongs to, so it names that record only when nothing of its own does: an
// unnamed record would not be cited at all.
const shipmentLabelKey = "proNumber"

// CitedRecords names the records a tool's result describes: every object in
// it whose id carries a record kind's prefix and that opens on a page of its
// own, or in the document viewer.
// A reader shows them as the records the answer was built from, so they come
// from what the tools returned and never from the reply's wording.
func CitedRecords(content string) []agentpagecontext.EntityRef {
	decoded, ok := decodeResult(content)
	if !ok {
		return nil
	}

	collector := &citedRecordCollector{seen: make(map[string]struct{})}
	collector.walk(decoded, 0)

	return collector.refs
}

func ReturnedRecords(content string) []agentpagecontext.EntityRef {
	decoded, ok := decodeResult(content)
	if !ok {
		return nil
	}

	collector := &returnedRecordCollector{seen: make(map[string]struct{})}
	collector.walk(decoded, 0)

	return collector.refs
}

func ObjectsHolding(content, value string) []map[string]any {
	decoded, ok := decodeResult(content)
	if !ok {
		return nil
	}
	found := make([]map[string]any, 0, 1)
	var walk func(node any, depth int)
	walk = func(node any, depth int) {
		if depth > maxCitedRecordDepth {
			return
		}
		switch typed := node.(type) {
		case map[string]any:
			for _, key := range slices.Sorted(maps.Keys(typed)) {
				if text, isText := typed[key].(string); isText && text == value {
					found = append(found, typed)
					break
				}
			}
			for _, key := range slices.Sorted(maps.Keys(typed)) {
				walk(typed[key], depth+1)
			}
		case []any:
			for _, child := range typed {
				walk(child, depth+1)
			}
		}
	}
	walk(decoded, 0)

	return found
}

func Payload(content string) (any, bool) {
	return decodeResult(content)
}

func decodeResult(content string) (any, bool) {
	if content == "" || len(content) > maxCitedContentBytes {
		return nil, false
	}
	if _, payload, fenced := Unfence(content); fenced {
		content = payload
	}
	if end := strings.Index(content, "\n\n"); end >= 0 {
		content = content[:end]
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') {
		return nil, false
	}

	var decoded any
	if err := sonic.UnmarshalString(trimmed, &decoded); err != nil {
		return nil, false
	}

	return decoded, true
}

type citedRecordCollector struct {
	refs []agentpagecontext.EntityRef
	seen map[string]struct{}
}

func (c *citedRecordCollector) full() bool {
	return len(c.refs) >= maxCitedRecords
}

func (c *citedRecordCollector) walk(value any, depth int) {
	if c.full() || depth > maxCitedRecordDepth {
		return
	}

	switch typed := value.(type) {
	case map[string]any:
		c.consider(typed)
		for _, child := range typed {
			if c.full() {
				return
			}
			switch child.(type) {
			case map[string]any, []any:
				c.walk(child, depth+1)
			}
		}
	case []any:
		for _, child := range typed {
			if c.full() {
				return
			}
			c.walk(child, depth+1)
		}
	}
}

func (c *citedRecordCollector) consider(record map[string]any) {
	ref, ok := recordRef(record)
	if !ok || ref.Label == "" {
		return
	}

	key := ref.Type + ":" + ref.ID
	if _, dup := c.seen[key]; dup {
		return
	}
	c.seen[key] = struct{}{}
	c.refs = append(c.refs, ref)
}

type returnedRecordCollector struct {
	refs []agentpagecontext.EntityRef
	seen map[string]struct{}
}

func (c *returnedRecordCollector) walk(value any, depth int) {
	if len(c.refs) >= maxReturnedRecords || depth > maxCitedRecordDepth {
		return
	}

	switch typed := value.(type) {
	case map[string]any:
		if ref, ok := recordRef(typed); ok {
			c.keep(ref)
			return
		}
		for _, key := range slices.Sorted(maps.Keys(typed)) {
			switch child := typed[key].(type) {
			case map[string]any, []any:
				c.walk(child, depth+1)
			}
		}
	case []any:
		for _, child := range typed {
			c.walk(child, depth+1)
		}
	}
}

func (c *returnedRecordCollector) keep(ref agentpagecontext.EntityRef) {
	key := ref.Type + ":" + ref.ID
	if _, dup := c.seen[key]; dup {
		return
	}
	c.seen[key] = struct{}{}
	c.refs = append(c.refs, ref)
}

func recordRef(record map[string]any) (agentpagecontext.EntityRef, bool) {
	id, ok := record["id"].(string)
	if !ok {
		return agentpagecontext.EntityRef{}, false
	}
	separator := strings.IndexByte(id, '_')
	if separator <= 0 {
		return agentpagecontext.EntityRef{}, false
	}
	resource, ok := recordkind.ResourceOfIDPrefix(id[:separator+1])
	if !ok {
		return agentpagecontext.EntityRef{}, false
	}
	kind := resource.String()
	if _, routable := productguide.RecordPath(kind, id); !routable &&
		resource != permission.ResourceDocument {
		return agentpagecontext.EntityRef{}, false
	}

	return agentpagecontext.EntityRef{
		Type:  kind,
		ID:    id,
		Label: citedRecordLabel(resource, record),
	}, true
}

func citedRecordLabel(resource permission.Resource, record map[string]any) string {
	if resource == permission.ResourceDocument {
		for _, key := range documentLabelKeys {
			if value := assistantartifact.ReadableString(record[key]); value != "" {
				return value
			}
		}
	}

	if resource != permission.ResourceShipment {
		own := strings.TrimSpace(assistantartifact.RecordLabelSkipping(record, shipmentLabelKey))
		if own != "" {
			return own
		}
	}

	return strings.TrimSpace(assistantartifact.RecordLabel(record))
}
