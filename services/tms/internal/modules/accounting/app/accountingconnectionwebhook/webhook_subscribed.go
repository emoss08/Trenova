package accountingconnectionwebhook

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type subscribedConnection struct {
	tenant pagination.TenantInfo
	id     pulid.ID
}
