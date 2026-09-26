package extractionevalservice

import (
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/pkg/pagination"
)

func tenantOf(run *extractioneval.ExtractionRun) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID}
}
