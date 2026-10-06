import { cleanup, render as rtlRender, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { BoardCapabilitiesProvider } from "@/lib/shipment-board/capabilities";
import { StatusCell } from "./status-cell";

function render(ui: ReactElement, operationType: "asset" | "brokerage" | "both" = "asset") {
  return rtlRender(
    <BoardCapabilitiesProvider value={{ ai: false, operationType, hos: false, maps: false }}>
      {ui}
    </BoardCapabilitiesProvider>,
  );
}

afterEach(cleanup);

describe("StatusCell", () => {
  // Billing has its own column; the status cell holds the shipment's status alone.
  it("shows the shipment status without the billing queue state", () => {
    render(
      <StatusCell
        shipment={
          { id: "shp_1", status: "ReadyToInvoice", billingTransferStatus: "InReview" } as Shipment
        }
      />,
    );

    expect(screen.getByText("Ready to invoice")).toBeInTheDocument();
    expect(screen.queryByText("In review")).not.toBeInTheDocument();
  });

  it("reads Booked rather than Assigned for a brokerage", () => {
    render(<StatusCell shipment={{ id: "shp_2", status: "Assigned" } as Shipment} />, "brokerage");
    expect(screen.getByText("Booked")).toBeInTheDocument();
  });

  it("keeps Assigned for an organization with its own drivers", () => {
    render(<StatusCell shipment={{ id: "shp_3", status: "Assigned" } as Shipment} />, "both");
    expect(screen.getByText("Assigned")).toBeInTheDocument();
  });
});
