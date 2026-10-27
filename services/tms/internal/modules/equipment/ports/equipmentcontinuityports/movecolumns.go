package equipmentcontinuityports

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/keelkinds"
)

type MoveColumns struct {
	AssignmentMoveID     keelkinds.PulidIDCol
	AssignmentTractorID  keelkinds.PulidIDCol
	AssignmentTrailerID  keelkinds.PulidIDCol
	AssignmentArchivedAt keel.Int64Col
	MoveID               keelkinds.PulidIDCol
	MoveShipmentID       keelkinds.PulidIDCol
	MoveInProgress       keel.Pred
	ShipmentID           keelkinds.PulidIDCol
	ShipmentProNumber    keel.StringCol
	StopMoveID           keelkinds.PulidIDCol
	StopSequence         keel.Int64Col
	StopWindowStart      keel.Int64Col
	StopWindowEnd        keel.Int64Col
}
