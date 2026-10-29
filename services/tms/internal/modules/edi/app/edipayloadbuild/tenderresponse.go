package edipayloadbuild

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
)

func TenderResponseFor(transfer *editransfer.EDITransfer) *edipayload.TenderResponsePayload {
	return BuildTenderResponsePayload(transfer).TenderResponse
}
