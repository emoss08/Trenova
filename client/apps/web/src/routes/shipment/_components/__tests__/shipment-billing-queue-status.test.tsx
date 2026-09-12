import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { ShipmentBillingQueueStatus } from "../shipment-billing-queue-status";

afterEach(cleanup);

function renderStatus(shipment: Partial<Shipment>) {
  return render(
    <MemoryRouter>
      <ShipmentBillingQueueStatus
        shipment={{ id: "shp_1", proNumber: "PRO 1001/A", ...shipment } as Shipment}
      />
    </MemoryRouter>,
  );
}

describe("ShipmentBillingQueueStatus", () => {
  it("shows the billing queue state and links to the shipment in the queue", () => {
    renderStatus({ status: "ReadyToInvoice", billingTransferStatus: "SentBackToOps" });

    expect(screen.getByText("Sent Back to Ops")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /view in billing queue/i })).toHaveAttribute(
      "href",
      "/billing/queue?query=PRO%201001%2FA&includePosted=true",
    );
  });

  it("renders nothing for a shipment that was never transferred", () => {
    const { container } = renderStatus({ status: "Completed", billingTransferStatus: null });

    expect(container).toBeEmptyDOMElement();
  });
});
