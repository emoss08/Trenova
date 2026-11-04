package toolresult

import (
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
	if content == "" || len(content) > maxCitedContentBytes {
		return nil
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') {
		return nil
	}

	var decoded any
	if err := sonic.UnmarshalString(trimmed, &decoded); err != nil {
		return nil
	}

	collector := &citedRecordCollector{seen: make(map[string]struct{})}
	collector.walk(decoded, 0)

	return collector.refs
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
	id, ok := record["id"].(string)
	if !ok {
		return
	}
	separator := strings.IndexByte(id, '_')
	if separator <= 0 {
		return
	}
	resource, ok := recordkind.ResourceOfIDPrefix(id[:separator+1])
	if !ok {
		return
	}
	kind := resource.String()
	if _, routable := productguide.RecordPath(kind, id); !routable &&
		resource != permission.ResourceDocument {
		return
	}

	label := citedRecordLabel(resource, record)
	if label == "" {
		return
	}

	key := kind + ":" + id
	if _, dup := c.seen[key]; dup {
		return
	}
	c.seen[key] = struct{}{}
	c.refs = append(c.refs, agentpagecontext.EntityRef{Type: kind, ID: id, Label: label})
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
