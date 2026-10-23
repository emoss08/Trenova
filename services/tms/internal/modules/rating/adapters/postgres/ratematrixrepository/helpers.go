package ratematrixrepository

import (
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
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
		dimension.VersionNumber = ratematrix.DraftVersion
		dimension.OrganizationID = entity.OrganizationID
		dimension.BusinessUnitID = entity.BusinessUnitID
	}
}
