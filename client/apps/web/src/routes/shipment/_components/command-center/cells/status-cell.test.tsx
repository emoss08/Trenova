import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it } from "vitest";
import { StatusCell } from "./status-cell";

afterEach(cleanup);

function makeShipment(overrides: Partial<Shipment>): Shipment {
  return { id: "shp_1", status: "ReadyToInvoice", ...overrides } as Shipment;
}

describe("StatusCell", () => {
  it("shows where the shipment sits in the billing queue under its status", () => {
    render(<StatusCell shipment={makeShipment({ billingTransferStatus: "InReview" })} />);

    const billing = screen.getByText("In Review");
    expect(billing.closest("[title]")).toHaveAttribute("title", "Billing queue: In Review");
  });

  it("shows a posted invoice as Posted", () => {
    render(
      <StatusCell
        shipment={makeShipment({ status: "Invoiced", billingTransferStatus: "Posted" })}
      />,
    );

    expect(screen.getByTitle("Billing queue: Posted")).toBeInTheDocument();
  });

  it("shows nothing extra for a shipment billing has never received", () => {
    const { container } = render(
      <StatusCell shipment={makeShipment({ billingTransferStatus: null })} />,
    );

    expect(container.querySelector("[title^='Billing queue']")).toBeNull();
  });
});
