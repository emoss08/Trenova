package equipmentcontinuityports

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/keelkinds"
)

type LocationColumns struct {
	ID        keelkinds.PulidIDCol
	Name      keel.StringCol
	Latitude  keel.Float64Col
	Longitude keel.Float64Col
}
