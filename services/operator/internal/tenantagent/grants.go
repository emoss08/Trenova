package tenantagent

import (
	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/supportkeys"
	"github.com/emoss08/trenova/operator/internal/tms"
)

func (a *Agent) grantsOf(id uuid.UUID, held []tms.SupportGrant) *[]instanceapi.SupportGrant {
	grants, dropped := supportkeys.Grants(held)
	for _, grant := range dropped {
		a.p.Logger.Warn(
			"the TMS listed a support grant the contract cannot carry; it is left out",
			"instanceId", id, "grant", grant,
		)
	}
	for i := 1; i < len(grants); i++ {
		if grants[i].OrganizationId != grants[0].OrganizationId {
			a.p.Logger.Error(
				"the TMS listed support grants of more than one organization for one tenant; none is reported",
				"instanceId",
				id,
			)
			return nil
		}
	}
	return &grants
}
