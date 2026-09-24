package document

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/stringutils"
)

var ownerResourceAliases = map[string]permission.Resource{
	"assistant_thread":   permission.ResourceAssistant,
	"invoice_adjustment": permission.ResourceInvoice,
}

func OwnerResource(resourceType string) permission.Resource {
	normalized := strings.ToLower(stringutils.ConvertCamelToSnake(strings.TrimSpace(resourceType)))
	if alias, ok := ownerResourceAliases[normalized]; ok {
		return alias
	}

	return permission.Resource(normalized)
}

func (d *Document) OwnerResource() permission.Resource {
	return OwnerResource(d.ResourceType)
}
