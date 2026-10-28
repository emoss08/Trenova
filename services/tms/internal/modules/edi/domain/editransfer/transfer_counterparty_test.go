package editransfer_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestEDITransferCounterparty_IsThePartnerTheReadingOrganizationKeeps(t *testing.T) {
	t.Parallel()

	sender := pulid.MustNew("org_")
	receiver := pulid.MustNew("org_")
	senderRecordOfReceiver := &edi.EDIPartner{Name: "Acme Foods"}
	receiverRecordOfSender := &edi.EDIPartner{Name: "Lone Star Logistics"}

	internal := &editransfer.EDITransfer{
		SourceOrganizationID: sender,
		TargetOrganizationID: receiver,
		SourcePartner:        senderRecordOfReceiver,
		TargetPartner:        receiverRecordOfSender,
	}
	assert.True(t, internal.ReceivedBy(receiver))
	assert.False(t, internal.ReceivedBy(sender))
	assert.Same(t, receiverRecordOfSender, internal.Counterparty(receiver))
	assert.Same(t, senderRecordOfReceiver, internal.Counterparty(sender))

	readByReceiver := &editransfer.EDITransfer{
		SourceOrganizationID: sender,
		TargetOrganizationID: receiver,
		TargetPartner:        receiverRecordOfSender,
	}
	assert.Same(t, receiverRecordOfSender, readByReceiver.Counterparty(receiver))

	readBySender := &editransfer.EDITransfer{
		SourceOrganizationID: sender,
		TargetOrganizationID: receiver,
		SourcePartner:        senderRecordOfReceiver,
	}
	assert.Same(t, senderRecordOfReceiver, readBySender.Counterparty(sender))

	walmart := &edi.EDIPartner{Name: "Walmart"}
	fromFile := &editransfer.EDITransfer{
		SourceOrganizationID: receiver,
		TargetOrganizationID: receiver,
		SourcePartner:        walmart,
		TargetPartner:        walmart,
	}
	assert.True(t, fromFile.ReceivedBy(receiver))
	assert.Same(t, walmart, fromFile.Counterparty(receiver))

	assert.Nil(t, (&editransfer.EDITransfer{TargetOrganizationID: receiver}).Counterparty(receiver))
}
