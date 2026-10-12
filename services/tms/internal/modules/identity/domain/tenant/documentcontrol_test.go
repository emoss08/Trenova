package tenant

import "testing"

func TestDocumentControlAllowsShipmentImportWhenShipmentAllowed(t *testing.T) {
	t.Parallel()

	control := &DocumentControl{
		ShipmentDraftAllowedResources: []string{"shipment"},
	}

	if !control.AllowsShipmentDraftResource("shipment_import") {
		t.Fatal("expected shipment_import to be allowed when shipment is allowed")
	}
}

func TestIsAllowedDocumentControlResourceNormalizesShipmentImport(t *testing.T) {
	t.Parallel()

	if !isAllowedDocumentControlResource("shipment_import") {
		t.Fatal("expected shipment_import to normalize to a supported resource")
	}
}
