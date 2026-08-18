package ratematrixrepository

import (
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/shared/pulid"
)

func stampDimensions(entity *ratematrix.RateMatrix, resetIDs bool) {
	for _, dimension := range entity.Dimensions {
		if dimension == nil {
			continue
		}

		if resetIDs {
			dimension.ID = pulid.Nil
		}

		dimension.RateMatrixID = entity.ID
		dimension.OrganizationID = entity.OrganizationID
		dimension.BusinessUnitID = entity.BusinessUnitID
	}
}
