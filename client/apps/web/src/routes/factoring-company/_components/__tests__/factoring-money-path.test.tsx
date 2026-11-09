import type { FactoringCompanyOverview } from "@/lib/graphql/factoring-company-page";
import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringMoneyPath } from "../factoring-money-path";

afterEach(() => cleanup());

const totals = (count: number, amountMinor: number) => ({ count, amountMinor });

function overview(amountsIncluded: boolean): FactoringCompanyOverview {
  return {
    factoringCompanyId: "fac_lonestar",
    currencyCode: "USD",
    amountsIncluded,
    movedWindowDays: 30,
    ready: totals(9, amountsIncluded ? 2_148_000 : 0),
    readyLate: totals(3, 0),
    submitted: totals(12, amountsIncluded ? 3_490_000 : 0),
    funded: totals(31, amountsIncluded ? 68_870_000 : 0),
    reserveHeld: totals(31, amountsIncluded ? 4_821_000 : 0),
    released: totals(44, amountsIncluded ? 6_239_940 : 0),
    chargebacksDue: totals(4, amountsIncluded ? 1_240_000 : 0),
    chargedBack: totals(1, amountsIncluded ? 185_000 : 0),
    oldestFundedOn: 1_790_000_000,
    chargebackDeadline: 1_794_500_000,
    awaiting: null,
    awaitingCount: 0,
    pace: [],
    paceMedianDays: null,
    paceLate: 0,
    agreement: null,
    delivery: { configured: true, method: "Sftp", lastTestedAt: null, lastTestError: null },
    notices: { assigned: 14, needsNotice: 2, awaiting: 3, acknowledged: 9 },
    attention: [],
    setup: { agreement: true, delivery: true, customers: true, notices: false, done: 3 },
    summary: [],
  };
}

function renderPath(data: FactoringCompanyOverview, onTab = vi.fn()) {
  render(
    <MemoryRouter>
      <FactoringMoneyPath overview={data} factoringCompanyId="fac_lonestar" onTab={onTab} />
    </MemoryRouter>,
  );
  return onTab;
}

describe("FactoringMoneyPath", () => {
  it("opens each stage in the factoring workspace narrowed to this factor", () => {
    renderPath(overview(true));
    const path = screen.getByRole("region", { name: "Where the invoices are" });
    const hrefs = within(path)
      .getAllByRole("link")
      .map((link) => link.getAttribute("href"));
    expect(hrefs).toEqual([
      "/billing/factoring?view=ready&factor=fac_lonestar",
      "/billing/factoring?view=submitted&factor=fac_lonestar",
      "/billing/factoring?view=funded&factor=fac_lonestar",
      "/billing/factoring?view=funded&factor=fac_lonestar",
    ]);
  });

  it("leads each stage with its amount when the reader can see money", () => {
    renderPath(overview(true));
    expect(screen.getByText("$21,480.00")).toBeTruthy();
    expect(screen.getByText("$688,700.00")).toBeTruthy();
    expect(screen.getByText("3 have been ready for over a day")).toBeTruthy();
  });

  it("leads with invoice counts and shows no amount without reserve access", () => {
    const { container } = render(
      <MemoryRouter>
        <FactoringMoneyPath
          overview={overview(false)}
          factoringCompanyId="fac_lonestar"
          onTab={vi.fn()}
        />
      </MemoryRouter>,
    );
    expect(screen.getByText("9 invoices")).toBeTruthy();
    expect(screen.getByText("31 invoices")).toBeTruthy();
    expect(container.textContent).not.toContain("$");
  });

  it("opens the ledger for what has come back", () => {
    const onTab = renderPath(overview(true));
    screen.getByRole("button", { name: /Released, last 30 days/ }).click();
    expect(onTab).toHaveBeenCalledWith("ledger");
  });
});
