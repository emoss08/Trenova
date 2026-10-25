package carriersettlementrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func when(include bool, p keel.Pred) keel.Pred {
	if !include {
		return nil
	}

	return p
}

func coalesceZero(e keel.Expr) keel.Expr {
	return keel.Raw("coalesce(?, 0)", e)
}

func tenantOf(orgID, buID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: orgID, BuID: buID}
}
