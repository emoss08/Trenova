import type { WorkerPTOBalanceView } from "@/lib/graphql/pto-policy";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { capBarParts, WorkerPTOBalances } from "../worker-pto-balances";

function balance(overrides: Partial<WorkerPTOBalanceView>): WorkerPTOBalanceView {
  return {
    ptoType: "Vacation",
    tracked: true,
    enforced: true,
    balanceDays: "11.50",
    pendingDays: "2.00",
    availableDays: "9.50",
    accruedYtdDays: "12.00",
    usedYtdDays: "5.50",
    carriedDays: "0.00",
    maxBalanceDays: "15.00",
    nextAccrual: null,
    ...overrides,
  } as WorkerPTOBalanceView;
}

afterEach(cleanup);

describe("capBarParts", () => {
  it("draws what is free and what pending requests hold as shares of the cap", () => {
    const bar = capBarParts(balance({}));
    expect(bar.scale).toBe(15);
    expect(bar.availablePercent).toBeCloseTo((9.5 / 15) * 100, 5);
    expect(bar.pendingPercent).toBeCloseTo((2 / 15) * 100, 5);
    expect(bar.overCap).toBe(false);
    expect(bar.ticks).toBe(15);
  });

  it("stretches past the cap when the balance stands above it", () => {
    const bar = capBarParts(
      balance({
        balanceDays: "6.00",
        pendingDays: "1.00",
        availableDays: "5.00",
        maxBalanceDays: "5.00",
      }),
    );
    expect(bar.scale).toBe(6);
    expect(bar.overCap).toBe(true);
    expect(bar.availablePercent + bar.pendingPercent).toBeCloseTo(100, 5);
  });

  it("measures against the balance when the policy sets no cap, and never draws below zero", () => {
    const uncapped = capBarParts(
      balance({
        balanceDays: "4.00",
        pendingDays: "0.00",
        availableDays: "4.00",
        maxBalanceDays: null,
      }),
    );
    expect(uncapped.scale).toBe(4);
    expect(uncapped.availablePercent).toBe(100);
    expect(uncapped.ticks).toBeNull();

    const overdrawn = capBarParts(
      balance({
        balanceDays: "-1.00",
        pendingDays: "0.00",
        availableDays: "-1.00",
        maxBalanceDays: null,
      }),
    );
    expect(overdrawn.availablePercent).toBe(0);
    expect(overdrawn.pendingPercent).toBe(0);
    expect(overdrawn.scale).toBe(1);
  });

  it("leaves the ticks off a cap too long to tick day by day", () => {
    expect(capBarParts(balance({ maxBalanceDays: "60.00" })).ticks).toBeNull();
  });
});

describe("WorkerPTOBalances", () => {
  it("draws one cell per tracked type with its free days, cap and figures", () => {
    render(
      <WorkerPTOBalances
        balances={[
          balance({
            nextAccrual: {
              entryType: "Accrual",
              periodKey: "M:2026-12",
              effectiveAt: 1_796_083_200,
              nominalDays: "1.00",
              deferred: false,
            },
          } as Partial<WorkerPTOBalanceView>),
          balance({
            ptoType: "Sick",
            enforced: false,
            balanceDays: "-1.00",
            pendingDays: "0.00",
            availableDays: "-1.00",
            maxBalanceDays: null,
          }),
          balance({ ptoType: "Personal", tracked: false }),
        ]}
      />,
    );

    const vacation = screen.getByTestId("pto-balance-Vacation");
    expect(vacation).toHaveTextContent("9.5");
    expect(vacation).toHaveTextContent("cap 15");
    expect(vacation).toHaveTextContent(
      "11.5 balance · 2 pending · 12 accrued · 5.5 used this year",
    );
    expect(vacation).toHaveTextContent("+1 on");
    expect(vacation).not.toHaveTextContent("M:2026-12");
    expect(vacation.querySelector('[data-slot="cap-bar-pending"]')).not.toBeNull();

    const sick = screen.getByTestId("pto-balance-Sick");
    expect(sick).toHaveTextContent("-1.0");
    expect(sick).toHaveTextContent("Not enforced");
    expect(sick).toHaveTextContent("no cap");
    expect(sick.querySelector('[data-slot="cap-bar-pending"]')).toBeNull();

    expect(screen.queryByTestId("pto-balance-Personal")).not.toBeInTheDocument();
  });

  it("says so when the policy tracks nothing", () => {
    render(<WorkerPTOBalances balances={[balance({ tracked: false })]} />);
    expect(
      screen.getByText("The policy does not track any type of time off yet."),
    ).toBeInTheDocument();
  });
});
