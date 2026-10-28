package toolparams

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// tenantFrom builds the tenant envelope every write travels in.
//
// The organization, the business unit and the acting user all come from the
// authenticated actor, never from the model's arguments. guardExecute has
// already asserted that the actor and the params agree, so there is one place
// a tool could get this wrong and it is not here.
func TenantFrom(params agenttoolports.ToolExecuteParams) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	}
}

// targetOf names the record a call acts on, from the argument that carries its
// id. A missing or malformed id yields no target, and the call is still made:
// the tool's own Execute reports the bad argument in its own words, and a
// proposal without a target simply skips the staleness check.
func TargetOf(
	params map[string]any,
	key string,
	resource permission.Resource,
) (agenttoolports.ToolTarget, bool) {
	raw, _ := params[key].(string)
	id, err := pulid.Parse(raw)
	if err != nil || id.IsNil() {
		return agenttoolports.ToolTarget{}, false
	}

	return agenttoolports.ToolTarget{Resource: resource, ID: id}, true
}

func TargetInvoice(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return TargetOf(params, ParamInvoiceID, permission.ResourceInvoice)
}
