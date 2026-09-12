import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { StatementGroup, StatementShipment } from "@trenova/shared/types/statement";
import { afterEach, describe, expect, it, vi } from "vitest";
import { StatementGroupCard } from "../statement-group-card";

afterEach(cleanup);

function shipment(overrides: Partial<StatementShipment> = {}): StatementShipment {
  return {
    billingQueueItemId: "bqi_1",
    shipmentId: "shp_1",
    orderId: null,
    proNumber: "PRO-1001",
    bol: "BOL-1",
    poNumber: "PO-9",
    orderNumber: null,
    serviceDate: 1_773_000_000,
    amount: 100,
    ...overrides,
  };
}

function group(overrides: Partial<StatementGroup> = {}): StatementGroup {
  return {
    key: "cus_1",
    label: "Acme Freight",
    shipmentCount: 3,
    // Deliberately disagrees with the members, so a card that renders the
    // server's total instead of summing what is actually included fails. The
    // member amounts are chosen so no subtotal equals any single row, which is
    // what lets the assertions below tell the header total apart from a line.
    totalAmount: 999999,
    belowMinimum: false,
    shipments: [
      shipment({ billingQueueItemId: "bqi_1", proNumber: "PRO-1001", amount: 100 }),
      shipment({ billingQueueItemId: "bqi_2", proNumber: "PRO-1002", amount: 250 }),
      shipment({ billingQueueItemId: "bqi_3", proNumber: "PRO-1003", amount: 75 }),
    ],
    ...overrides,
  };
}

function renderCard(props: Partial<Parameters<typeof StatementGroupCard>[0]> = {}) {
  const handlers = {
    onToggleExpanded: vi.fn(),
    onToggleShipment: vi.fn(),
    onToggleGroup: vi.fn(),
  };
  render(
    <StatementGroupCard
      group={group()}
      currencyCode="USD"
      expanded
      heldIds={new Set()}
      {...handlers}
      {...props}
    />,
  );
  return handlers;
}

describe("StatementGroupCard", () => {
  // The number a biller bills against has to be the number they are looking at,
  // so the total is summed from the included members and never read off the group.
  it("sums the shipments that are actually included", () => {
    renderCard();

    expect(screen.getByText("$425.00")).toBeInTheDocument();
    expect(screen.queryByText("$999,999.00")).not.toBeInTheDocument();
  });

  it("drops a held shipment out of the total and says how many remain", () => {
    renderCard({ heldIds: new Set(["bqi_2"]) });

    expect(screen.getByText("$175.00")).toBeInTheDocument();
    expect(screen.getByText("2 of 3 shipments")).toBeInTheDocument();
  });

  it("reports every shipment when nothing is held", () => {
    renderCard();

    expect(screen.getByText("3 shipments")).toBeInTheDocument();
  });

  it("hands back the shipment whose checkbox was clicked", async () => {
    const user = userEvent.setup();
    const handlers = renderCard();

    await user.click(screen.getByRole("checkbox", { name: /include PRO-1002/i }));

    expect(handlers.onToggleShipment).toHaveBeenCalledWith(
      expect.objectContaining({ billingQueueItemId: "bqi_2" }),
    );
  });

  // Partly-held is neither checked nor unchecked: clicking the header must be
  // able to mean "put them all back", and an unchecked box would say the opposite.
  it("shows the group checkbox as indeterminate when only some are held", () => {
    renderCard({ heldIds: new Set(["bqi_2"]) });

    const groupCheckbox = screen.getByRole("checkbox", { name: /include every shipment/i });
    expect(groupCheckbox).toHaveAttribute("data-indeterminate");
  });

  it("collapses its shipments away", () => {
    renderCard({ expanded: false });

    expect(screen.queryByText("PRO-1001")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { expanded: false })).toBeInTheDocument();
  });

  // A group under the minimum still shows its total — it is not zero, it is just
  // not going out — so the flag has to be a separate marker.
  it("marks a group held under the customer minimum without hiding its value", () => {
    renderCard({ group: group({ belowMinimum: true }) });

    expect(screen.getByLabelText(/under the customer's invoice minimum/i)).toBeInTheDocument();
    expect(screen.getByText("$425.00")).toBeInTheDocument();
  });
});
