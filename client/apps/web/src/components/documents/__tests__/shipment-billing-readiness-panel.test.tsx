import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment, ShipmentBillingReadiness } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ShipmentBillingReadinessPanel } from "../shipment-billing-readiness-panel";

afterEach(cleanup);

const readiness = {
  shipmentId: "shp_1",
  shipmentStatus: "ReadyToInvoice",
  policy: {},
  requirements: [],
  missingRequirements: [],
  validationFailures: [],
  warnings: [],
  serviceFailureContext: { hasUnresolved: false, unresolvedCount: 0, serviceFailureIds: [] },
  canMarkReadyToInvoice: true,
  shouldAutoMarkReadyToInvoice: false,
  shouldAutoTransferToBilling: false,
} as unknown as ShipmentBillingReadiness;

function renderPanel(shipment: Partial<Shipment>) {
  return render(
    <ShipmentBillingReadinessPanel
      readiness={readiness}
      shipment={{ id: "shp_1", ...shipment } as Shipment}
      onUploadRequired={vi.fn()}
      onMarkReadyToInvoice={vi.fn()}
      isMarkingReady={false}
    />,
  );
}

describe("ShipmentBillingReadinessPanel status hint", () => {
  it("reports the billing queue state once billing holds the shipment", () => {
    renderPanel({ status: "ReadyToInvoice", billingTransferStatus: "SentBackToOps" });

    expect(screen.getByTitle("Billing queue: Sent back to ops")).toBeInTheDocument();
    expect(screen.queryByText("Ready to invoice")).not.toBeInTheDocument();
  });

  it("still says ready to invoice before the shipment is transferred", () => {
    renderPanel({ status: "ReadyToInvoice", billingTransferStatus: null });

    expect(screen.getByText("Ready to invoice")).toBeInTheDocument();
  });
});

describe("ShipmentBillingReadinessPanel requirements", () => {
  it("says an attached proof of delivery was rejected rather than missing", () => {
    const pod = {
      documentTypeId: "dt_pod",
      documentTypeCode: "POD",
      documentTypeName: "Proof of Delivery",
      satisfied: false,
      documentCount: 0,
      documentIds: [],
      ineligibleDocuments: [{ documentId: "doc_1", standing: "rejected" as const }],
    };

    render(
      <ShipmentBillingReadinessPanel
        readiness={{
          ...readiness,
          shipmentStatus: "Completed",
          canMarkReadyToInvoice: false,
          requirements: [pod],
          missingRequirements: [pod],
        }}
        shipment={{ id: "shp_1", status: "Completed" } as Shipment}
        onUploadRequired={vi.fn()}
        onMarkReadyToInvoice={vi.fn()}
        isMarkingReady={false}
      />,
    );

    expect(screen.getByText("Proof of Delivery")).toBeInTheDocument();
    expect(screen.getByText("Uploaded copy was rejected")).toBeInTheDocument();
  });
});
