package reportaccess

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

// Reason says, in a sentence fragment the compiler joins into its refusal,
// why a field or edge of an entity cannot be used, naming the kind read,
// the tier or the protected group the caller lacks. It is empty for an
// available use.
func (r *Resolver) Reason(entity *reportcatalog.Entity, label string, use Use) string {
	switch use.Decision {
	case Available:
		return ""
	case NeedsKindRead:
		return fmt.Sprintf(
			"you do not have read access to %s, which %q on %s needs; "+
				"without it a report can use only its reference fields",
			r.KindLabel(use.Kind), label, entity.Label,
		)
	case WithheldByCeiling:
		return fmt.Sprintf(
			"field %q on %s requires %s access",
			label, entity.Label, use.Sensitivity,
		)
	case NeedsFieldGroup:
		return fmt.Sprintf(
			"field %q on %s is in the %s protected field group, which your role does not grant",
			label, entity.Label, use.Group,
		)
	default:
		return fmt.Sprintf("field %q on %s is not available to you", label, entity.Label)
	}
}

// Need is what a caller lacks for a use, in a few words for a model: "customer
// read", "the Personal field group", "restricted access". It is empty for an
// available use.
func (r *Resolver) Need(use Use) string {
	switch use.Decision {
	case Available:
		return ""
	case NeedsKindRead:
		return fmt.Sprintf("%s read", use.Kind)
	case WithheldByCeiling:
		return fmt.Sprintf("%s access", use.Sensitivity)
	case NeedsFieldGroup:
		return fmt.Sprintf("the %s field group", use.Group)
	default:
		return "access"
	}
}

// KindLabel is the plural label of the catalog entity that is a resource's
// own record, such as Customers for customer.
func (r *Resolver) KindLabel(resource permission.Resource) string {
	fallback := ""
	for i := range r.catalog.Entities {
		entity := &r.catalog.Entities[i]
		if entity.Resource != resource {
			continue
		}
		if !entity.WithinRecord {
			return entity.PluralLabel
		}
		if fallback == "" {
			fallback = entity.PluralLabel
		}
	}
	if fallback != "" {
		return fallback
	}
	return resource.String()
}

// KindRead is the refusal for a report that reads a kind's own rows, as its
// root or a relationship it picks a row of, without the kind's read.
func (r *Resolver) KindRead(entity *reportcatalog.Entity) string {
	return fmt.Sprintf("you do not have read access to %s", entity.PluralLabel)
}

// KindExport is the refusal for a run that exports a kind's own fields
// without the kind's export.
func (r *Resolver) KindExport(entity *reportcatalog.Entity) string {
	return fmt.Sprintf("you do not have export access to %s", entity.PluralLabel)
}

// Need is Resolver.Need, for a surface that holds only the access.
func (a *Access) Need(use Use) string {
	return a.resolver.Need(use)
}
