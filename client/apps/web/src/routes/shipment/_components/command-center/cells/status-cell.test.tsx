import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it } from "vitest";
import { StatusCell } from "./status-cell";

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

    expect(screen.getByText("Ready to Invoice")).toBeInTheDocument();
    expect(screen.queryByText("In Review")).not.toBeInTheDocument();
  });
});
