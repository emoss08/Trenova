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

    expect(screen.getByTitle("Billing queue: Sent Back to Ops")).toBeInTheDocument();
    expect(screen.queryByText("Ready to invoice")).not.toBeInTheDocument();
  });

  it("still says ready to invoice before the shipment is transferred", () => {
    renderPanel({ status: "ReadyToInvoice", billingTransferStatus: null });

    expect(screen.getByText("Ready to invoice")).toBeInTheDocument();
  });
});
