import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { OpenStatement } from "@trenova/shared/types/statement";
import { afterEach, describe, expect, it, vi } from "vitest";
import { filterStatements, sortStatements, StatementSidebar } from "../statement-sidebar";

afterEach(cleanup);

const NOW = 1_774_000_000;
const DAY = 86_400;

function statement(overrides: Partial<OpenStatement> = {}): OpenStatement {
  return {
    customerId: "cus_1",
    customerName: "Acme Freight",
    customerCode: "ACME",
    customerStatus: "Active",
    cycle: "Monthly",
    billingCycleAnchorDay: 1,
    billingCycleTimezone: "America/Denver",
    periodStart: NOW - 10 * DAY,
    periodEnd: NOW + 10 * DAY,
    lastBilledPeriodEnd: null,
    shipmentCount: 4,
    invoiceCount: 1,
    totalAmount: 4200,
    currencyCode: "USD",
    splitBy: "Customer",
    sectionBy: "Shipment",
    detail: "Detailed",
    minimumAmount: null,
    autoBill: false,
    belowMinimum: false,
    groups: [],
    ...overrides,
  };
}

describe("sortStatements", () => {
  // The work is deadline-shaped: the statement closing tonight is the one that
  // matters, not the largest one.
  it("puts the soonest boundary first by default", () => {
    const soon = statement({ customerId: "soon", periodEnd: NOW + DAY, totalAmount: 10 });
    const later = statement({
      customerId: "later",
      periodEnd: NOW + 20 * DAY,
      totalAmount: 99999,
    });

    expect(sortStatements([later, soon], "soonest").map((s) => s.customerId)).toEqual([
      "soon",
      "later",
    ]);
  });

  it("sorts by value when asked", () => {
    const small = statement({ customerId: "small", totalAmount: 10, periodEnd: NOW + DAY });
    const large = statement({
      customerId: "large",
      totalAmount: 500,
      periodEnd: NOW + 20 * DAY,
    });

    expect(sortStatements([small, large], "value").map((s) => s.customerId)).toEqual([
      "large",
      "small",
    ]);
  });

  // An empty statement is a fact, not a task. It sinks under every sort, or the
  // customer with nothing accrued and a boundary tonight would head the list.
  it("sinks statements with no freight under every sort", () => {
    const empty = statement({
      customerId: "empty",
      shipmentCount: 0,
      totalAmount: 0,
      periodEnd: NOW + DAY,
    });
    const busy = statement({ customerId: "busy", periodEnd: NOW + 20 * DAY });

    for (const sort of ["soonest", "value", "name"] as const) {
      expect(sortStatements([empty, busy], sort).map((s) => s.customerId)).toEqual([
        "busy",
        "empty",
      ]);
    }
  });

  it("does not mutate the list it was given", () => {
    const input = [
      statement({ customerId: "b", periodEnd: NOW + 20 * DAY }),
      statement({ customerId: "a", periodEnd: NOW + DAY }),
    ];

    sortStatements(input, "soonest");

    expect(input.map((s) => s.customerId)).toEqual(["b", "a"]);
  });
});

describe("filterStatements", () => {
  it("matches on name and on code, case-insensitively", () => {
    const acme = statement({ customerId: "acme", customerName: "Acme Freight", customerCode: "ACM" });
    const other = statement({ customerId: "other", customerName: "Globex", customerCode: "GLX" });

    expect(filterStatements([acme, other], "acme").map((s) => s.customerId)).toEqual(["acme"]);
    expect(filterStatements([acme, other], "glx").map((s) => s.customerId)).toEqual(["other"]);
    expect(filterStatements([acme, other], "  ").map((s) => s.customerId)).toEqual([
      "acme",
      "other",
    ]);
  });

  // A customer with no code is a real row, not a crash.
  it("survives a missing code", () => {
    const noCode = statement({ customerCode: null });

    expect(filterStatements([noCode], "acme")).toHaveLength(1);
    expect(filterStatements([noCode], "zzz")).toHaveLength(0);
  });
});

describe("StatementSidebar", () => {
  it("says nothing matches, and offers to clear, when the search empties the list", async () => {
    const user = userEvent.setup();
    render(
      <StatementSidebar
        statements={[statement()]}
        loading={false}
        nowSeconds={NOW}
        selectedCustomerId={null}
        onSelect={vi.fn()}
      />,
    );

    await user.type(screen.getByPlaceholderText("Search customer..."), "zzzz");

    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /clear filters/i })).toBeInTheDocument();
  });

  // The no-data voice must not read as a filter problem: nobody is on a statement
  // schedule, and the fix is in the customer's billing profile, not the search box.
  it("explains the absence rather than offering a filter reset when there are none", () => {
    render(
      <StatementSidebar
        statements={[]}
        loading={false}
        nowSeconds={NOW}
        selectedCustomerId={null}
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByText("No statement customers")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /clear filters/i })).not.toBeInTheDocument();
  });

  it("selects the customer whose card was clicked", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <StatementSidebar
        statements={[statement({ customerId: "cus_42", customerName: "Globex" })]}
        loading={false}
        nowSeconds={NOW}
        selectedCustomerId={null}
        onSelect={onSelect}
      />,
    );

    await user.click(screen.getByText("Globex"));

    expect(onSelect).toHaveBeenCalledWith("cus_42");
  });
});
