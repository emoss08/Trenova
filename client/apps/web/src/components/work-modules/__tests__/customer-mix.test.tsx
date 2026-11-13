import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ShipmentAnalyticsData } from "@/lib/shipment-analytics";
import { CustomerMix } from "../customer-mix";

type CustomerMixCard = ShipmentAnalyticsData["customerMix"];

function renderMix(customerMix: CustomerMixCard) {
  render(<CustomerMix customerMix={customerMix} />);
}

// The figures about each customer need customer read, and the order of the names alone ranks
// them by revenue, so a shipment reader who cannot read customers gets no customer mix at all
// (null) and the card says which access it needs instead of drawing an empty list.
describe("customer mix", () => {
  it("says a reader who cannot read customers needs that access", () => {
    renderMix(null);

    expect(
      screen.getByText("You need access to customers to see the customer mix"),
    ).toBeInTheDocument();
    expect(screen.queryByText("No customer revenue in this window")).toBeNull();
    expect(screen.queryByText(/\$/)).toBeNull();
  });

  it("draws a customer reader's figures", () => {
    renderMix({
      windowDays: 30,
      entries: [
        {
          customerId: "cus_1",
          name: "Acme Foods",
          revenue: 40000,
          share: 80,
          loads: 10,
          trend: 12.5,
        },
      ],
    });

    const row = screen.getByText("Acme Foods").closest("li");
    expect(row).not.toBeNull();
    expect(within(row as HTMLElement).getByTitle("$40,000.00 · 10 loads")).toBeInTheDocument();
    expect(within(row as HTMLElement).getByText("▲12.5")).toBeInTheDocument();
    expect(screen.queryByText("You need access to customers to see the customer mix")).toBeNull();
  });

  // The widget that shows the mix draws the one frame and the one title, so the mix
  // brings neither: a second card inside the tile was a frame within a frame.
  it("draws no card or heading of its own", () => {
    const { container } = render(
      <CustomerMix
        customerMix={{
          windowDays: 30,
          entries: [
            { customerId: "cus_1", name: "Acme Foods", revenue: 1, share: 1, loads: 1, trend: 0 },
          ],
        }}
      />,
    );

    expect(container.querySelector("section, header, [role=tablist]")).toBeNull();
    expect(screen.queryByRole("heading")).toBeNull();
  });

  it("tells an empty window apart from a withheld card", () => {
    renderMix({ windowDays: 30, entries: [] });

    expect(screen.getByText("No customer revenue in this window")).toBeInTheDocument();
    expect(screen.queryByText("You need access to customers to see the customer mix")).toBeNull();
  });
});
