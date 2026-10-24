package driverpayrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const nowEpochSQL = "extract(epoch from current_timestamp)::bigint"

func when(include bool, p keel.Pred) keel.Pred {
	if !include {
		return nil
	}

	return p
}

func searchPred(query string, cols ...keel.StringCol) keel.Pred {
	if query == "" {
		return nil
	}

	preds := make([]keel.Pred, 0, len(cols))
	for _, col := range cols {
		preds = append(preds, col.ContainsFold(query))
	}

	return keel.Or(preds...)
}

func openAfterNow(col keel.Int64Col) keel.Pred {
	return keel.Or(col.IsNull(), keel.Raw("? > "+nowEpochSQL, col))
}

func openAfter(col keel.Int64Col, at int64) keel.Pred {
	return keel.Or(col.IsNull(), col.Gt(at))
}

func tenantOf(orgID, buID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: orgID, BuID: buID}
}
