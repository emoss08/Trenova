package recordshareresolver

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
)

// shareResource is the resource whose read permission a share of this kind
// of record needs: the record's own. A kind the registry does not name maps
// to no resource the reader can hold, so the permission check refuses it and
// the service never sees it.
func shareResource(kind recordkind.Kind) permission.Resource {
	if !kind.Known() {
		return permission.Resource("unknown_record_kind")
	}

	return kind.Resource()
}
