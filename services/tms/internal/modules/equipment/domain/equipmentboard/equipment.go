package equipmentboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

var Kinds = []boardcatalog.Kind{
	{Kind: recordkind.Kind(permission.ResourceTractor)},
	{Kind: recordkind.Kind(permission.ResourceTrailer)},
}
