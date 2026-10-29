package editenderservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

type LoadTenderPlan struct {
	SourceShipment *shipment.Shipment
	SourcePartner  *edi.EDIPartner
	TargetPartner  *edi.EDIPartner
	Payload        editransfer.LoadTenderPayload
	Mapping        *editendermapping.MappingPreview
	Status         editransfer.TransferStatus
}
