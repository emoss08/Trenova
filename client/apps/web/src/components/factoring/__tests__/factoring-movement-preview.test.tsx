import type { FactoringPlannedMovement } from "@/lib/graphql/factoring";
import { cleanup, render, screen } from "@testing-library/react";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { afterEach, describe, expect, it } from "vitest";
import { FactoringMovementPreview } from "../factoring-movement-preview";

afterEach(cleanup);

const REQUESTED = 1_801_000_000;
const OPEN_DAY = 1_801_600_000;

function releaseMovement(
  overrides: Partial<FactoringPlannedMovement> = {},
): FactoringPlannedMovement {
  return {
    itemId: "facsubi_1",
    invoiceId: "inv_1",
    invoiceNumber: "INV-100",
    customerId: "cus_1",
    requestedDate: REQUESTED,
    accountingDate: REQUESTED,
    fiscalPeriodId: null,
    fiscalPeriodName: null,
    requiresApproval: false,
    movement: {
      kind: "ReserveRelease",
      treatment: "Sale",
      cashMinor: 7_000,
      feeMinor: 300,
      reserveMinor: -7_300,
      liabilityMinor: 0,
      arMinor: 0,
      invoiceEffect: "None",
      invoiceMinor: 0,
    },
    lines: [
      {
        role: "Cash",
        accountId: "gla_cash",
        debit: 7_000,
        credit: 0,
        account: { id: "gla_cash", accountCode: "1000", name: "Operating cash" },
      },
      {
        role: "FeeExpense",
        accountId: "gla_fee",
        debit: 300,
        credit: 0,
        account: { id: "gla_fee", accountCode: "6100", name: "Factoring fees" },
      },
      {
        role: "DueFromFactor",
        accountId: "gla_due",
        debit: 0,
        credit: 7_300,
        account: null,
      },
    ],
    ...overrides,
  };
}

function notes(): string[] {
  return screen.getAllByRole("listitem").map((item) => item.textContent ?? "");
}

describe("FactoringMovementPreview", () => {
  it("names the fiscal period as a sentence of its own", () => {
    render(
      <FactoringMovementPreview
        movement={releaseMovement({ fiscalPeriodId: "fp_1", fiscalPeriodName: "2027-02" })}
        currencyCode="USD"
      />,
    );

    expect(notes()).toEqual([
      `Books on ${formatUnixDateMedium(REQUESTED)}.`,
      "Period 2027-02.",
      "Leaves the invoice's balance as it is.",
    ]);
  });

  it("names the first open day and leaves the period out when the plan has none", () => {
    render(
      <FactoringMovementPreview
        movement={releaseMovement({ accountingDate: OPEN_DAY, requiresApproval: true })}
        currencyCode="USD"
      />,
    );

    expect(notes()).toEqual([
      `${formatUnixDateMedium(REQUESTED)} is in a closed period, so it books on ${formatUnixDateMedium(OPEN_DAY)}, the first open day.`,
      "The entry waits for approval before it reaches the ledger.",
      "Leaves the invoice's balance as it is.",
    ]);
  });

  it("shows each line and the totals in the schedule's currency", () => {
    render(<FactoringMovementPreview movement={releaseMovement()} currencyCode="EUR" />);

    expect(screen.getByText("1000 · Operating cash")).toBeTruthy();
    expect(screen.getByText("gla_due")).toBeTruthy();
    expect(screen.getAllByText("€73.00")).toHaveLength(3);
    expect(screen.getByText("€70.00")).toBeTruthy();
    expect(screen.getByText("€3.00")).toBeTruthy();
  });
});
