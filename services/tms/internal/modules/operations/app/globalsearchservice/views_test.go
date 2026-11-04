package globalsearchservice

import (
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

var searchViewRule = sync.OnceValue(func() *refview.Views {
	return refview.New(permissioncatalog.NewRegistry())
})

// searchViews is the reference view rule the service is built with, over the
// real registry. With no permission engine a person's ceiling on a record
// they reach through another is Internal, as it is when their permissions
// cannot be read.
func searchViews() permissionports.ReferenceViews {
	return refview.NewPort(searchViewRule(), nil)
}
