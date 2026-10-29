package ediselectoptions

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestEDISelectOptionMappers(t *testing.T) {
	t.Parallel()

	tenderSender := pulid.MustNew("org_")
	tenderReceiver := pulid.MustNew("org_")
	transferEntity := &editransfer.EDITransfer{
		ID:                   pulid.MustNew("edilt_"),
		SourceOrganizationID: tenderSender,
		TargetOrganizationID: tenderReceiver,
		Status:               editransfer.TransferStatusSubmitted,
		CreatedAt:            1780415500,
		TenderPayload: editransfer.LoadTenderPayload{
			BOL:           "BOL-3003",
			CustomerLabel: "ACME Freight",
		},
		SourcePartner: &edi.EDIPartner{Name: "Partner A"},
		TargetPartner: &edi.EDIPartner{Name: "Partner B"},
	}
	transferOption := ediTransferSelectOption(transferEntity, tenderReceiver)
	assert.Equal(t, "BOL-3003", transferOption.Label)
	assert.Equal(t, "ACME Freight", *transferOption.Description)
	assert.Equal(t, string(editransfer.TransferStatusSubmitted), transferOption.Meta["status"])
	assert.Equal(t, "Partner B", transferOption.Meta["partner"])
	assert.NotContains(t, transferOption.Meta, "sourcePartner")
	assert.NotContains(t, transferOption.Meta, "targetPartner")
	assert.Equal(
		t,
		"Partner A",
		ediTransferSelectOption(transferEntity, tenderSender).Meta["partner"],
	)
	assert.Equal(
		t,
		transferEntity.CreatedAt,
		ediTransferSelectOptionItemFor(tenderReceiver)(transferEntity).Cursor.At,
	)

	fallbackTransfer := &editransfer.EDITransfer{
		ID:                   pulid.MustNew("edilt_"),
		TargetOrganizationID: tenderReceiver,
		Status:               editransfer.TransferStatusSubmitted,
		CreatedAt:            1780415600,
	}
	unnamedTransfer := ediTransferSelectOption(fallbackTransfer, tenderReceiver)
	assert.Equal(t, "Load tender "+fallbackTransfer.ID.String(), unnamedTransfer.Label)
	assert.NotContains(t, unnamedTransfer.Meta, "partner")

	connectionEntity := &edi.EDIConnection{
		ID:                   pulid.MustNew("edic_"),
		SourceOrganizationID: pulid.MustNew("org_"),
		TargetOrganizationID: pulid.MustNew("org_"),
		Method:               edi.ConnectionMethodInternal,
		Status:               edi.ConnectionStatusActive,
		CreatedAt:            1780415700,
		SourceOrganization:   &tenant.Organization{Name: "Acme Corp"},
		TargetOrganization:   &tenant.Organization{Name: "Beta LLC"},
	}
	connectionOption := ediConnectionSelectOption(connectionEntity)
	assert.Equal(t, "Acme Corp → Beta LLC", connectionOption.Label)
	assert.Equal(t, "Internal · Active", *connectionOption.Description)
	assert.Equal(t, string(edi.ConnectionMethodInternal), connectionOption.Meta["method"])
	assert.Equal(t, string(edi.ConnectionStatusActive), connectionOption.Meta["status"])
	assert.Equal(t, "Acme Corp", connectionOption.Meta["sourceOrganizationName"])
	assert.Equal(t, "Beta LLC", connectionOption.Meta["targetOrganizationName"])
	assert.Equal(
		t,
		connectionEntity.CreatedAt,
		ediConnectionSelectOptionItem(connectionEntity).Cursor.At,
	)

	fallbackConnection := &edi.EDIConnection{
		ID:                   pulid.MustNew("edic_"),
		SourceOrganizationID: pulid.MustNew("org_"),
		TargetOrganizationID: pulid.MustNew("org_"),
		Method:               edi.ConnectionMethodInternal,
		Status:               edi.ConnectionStatusActive,
		CreatedAt:            1780415800,
	}
	fallbackOption := ediConnectionSelectOption(fallbackConnection)
	assert.Equal(
		t,
		fallbackConnection.SourceOrganizationID.String()+" → "+fallbackConnection.TargetOrganizationID.String(),
		fallbackOption.Label,
	)
}
