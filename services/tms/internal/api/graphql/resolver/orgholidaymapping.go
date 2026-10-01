package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func orgHolidayFromInput(
	input *gqlmodel.OrgHolidayInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *worker.OrgHoliday {
	return &worker.OrgHoliday{
		ID:             id,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           input.Name,
		HolidayDate:    int64(input.HolidayDate),
		Kind:           input.Kind,
		RecursAnnually: input.RecursAnnually,
		Description:    stringValue(input.Description),
		Version:        int64(intValue(input.Version)),
	}
}
