import type { CollectionWorklistRow } from "@/types/collection";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CollectionsTable } from "../collections-table";

afterEach(cleanup);

function row(overrides: Partial<CollectionWorklistRow>): CollectionWorklistRow {
  return {
    customerId: "cus_1",
    customerName: "Acme Logistics",
    customerCode: "ACME",
    creditStatus: "Active",
    currencyCode: "USD",
    totalOpenMinor: 500_000,
    overdueMinor: 250_000,
    oldestDaysPastDue: 42,
    openInvoiceCount: 3,
    disputedInvoiceCount: 0,
    collectorId: null,
    collectorName: "",
    nextFollowUpOn: null,
    followUpDue: false,
    lastActivityAt: null,
    lastActivityKind: "",
    lastActivitySummary: "",
    promise: null,
    priority: "High",
    ...overrides,
  };
}

describe("CollectionsTable", () => {
  it("shows what is overdue, the promise standing against it and who is collecting", () => {
    render(
      <CollectionsTable
        selectedCustomerId={null}
        onSelect={vi.fn()}
        rows={[
          row({
            priority: "Critical",
            collectorName: "Jordan Lee",
            promise: {
              activityId: "cact_1",
              amountMinor: 100_000,
              currencyCode: "USD",
              promisedOn: 1_700_000_000,
              madeAt: 1_699_000_000,
              madeByName: "Jordan Lee",
              paidSinceMinor: 0,
              status: "Broken",
            },
          }),
        ]}
      />,
    );

    const [, body] = screen.getAllByRole("rowgroup");
    const cells = within(body).getAllByRole("cell");
    expect(cells[0]).toHaveTextContent("Critical");
    expect(cells[0]).toHaveTextContent("Acme Logistics");
    expect(cells[1]).toHaveTextContent("$2,500.00");
    expect(cells[1]).toHaveTextContent("$5,000.00 open");
    expect(cells[2]).toHaveTextContent("42d");
    expect(cells[3]).toHaveTextContent("Broken");
    expect(cells[3]).toHaveTextContent("$1,000.00 by");
    expect(cells[4]).toHaveTextContent("Not scheduled");
    expect(cells[5]).toHaveTextContent("Jordan Lee");
    expect(cells[6]).toHaveTextContent("Never contacted");
  });

  it("opens the account when a row is chosen", async () => {
    const onSelect = vi.fn();
    render(
      <CollectionsTable
        selectedCustomerId={null}
        onSelect={onSelect}
        rows={[row({ customerId: "cus_9", customerName: "Peak Freight" })]}
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: "Peak Freight" }));

    expect(onSelect).toHaveBeenCalledWith("cus_9");
  });
});
