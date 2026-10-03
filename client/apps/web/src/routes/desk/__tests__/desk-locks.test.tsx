import { deskComposerLock, DeskUsageMeter } from "@/routes/desk/_components/desk-locks";
import type { ThreadBudget } from "@/types/assistant";
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

const t = ((text: string, ...args: unknown[]) =>
  text
    .replace(/\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu, (_m, i, one, other) =>
      (Number(args[Number(i)]) === 1 ? one : other).replace("#", String(args[Number(i)])),
    )
    .replace(/\{(\d+)\}/gu, (_m, i) => String(args[Number(i)]))) as never;

const nov1 = Date.UTC(2026, 10, 1) / 1000;
const oct1 = Date.UTC(2026, 9, 1) / 1000;

function budget(overrides: Partial<ThreadBudget> = {}): ThreadBudget {
  return {
    agentName: "Billing Specialist",
    spentUsd: "460.00",
    limitUsd: "500.00",
    share: 0.92,
    near: true,
    monthStart: oct1,
    resetsAt: nov1,
    runsToday: 3,
    dailyRunLimit: 200,
    budgetUsed: false,
    dailyUsed: false,
    dayResetsAt: Date.UTC(2026, 9, 4) / 1000,
    disabledBy: "",
    disabledAt: 0,
    person: null,
    ...overrides,
  };
}

function lockFor(overrides: Partial<Parameters<typeof deskComposerLock>[0]> = {}) {
  return deskComposerLock({
    thread: { canContinue: true, cannotContinueReason: undefined },
    agentName: "Billing Specialist",
    budget: budget({ near: false, share: 0.2 }),
    noModel: false,
    timezone: "UTC",
    t,
    requestMore: vi.fn(),
    startElsewhere: vi.fn(),
    openAgentControl: vi.fn(),
    ...overrides,
  });
}

function show(node: ReactNode) {
  return render(<div>{node}</div>);
}

describe("deskComposerLock", () => {
  it("leaves the composer open while nothing stops it", () => {
    expect(lockFor()).toBeNull();
  });

  it("names who turned the agent off and when", () => {
    const lock = lockFor({
      thread: { canContinue: false, cannotContinueReason: "AgentDisabled" },
      budget: budget({ disabledBy: "Jordan Pike", disabledAt: Date.UTC(2026, 8, 30, 12) / 1000 }),
    });
    show(lock?.lock);
    expect(
      screen.getByText(/Jordan Pike turned off Billing Specialist on Sep(tember)? 30\./u),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start with another agent" })).toBeInTheDocument();
  });

  it("offers to request access the person lost", () => {
    const requestMore = vi.fn();
    show(
      lockFor({ thread: { canContinue: false, cannotContinueReason: "NoAccess" }, requestMore })
        ?.lock,
    );
    screen.getByRole("button", { name: "Request access" }).click();
    expect(requestMore).toHaveBeenCalledWith("access");
  });

  it("says no model is set up and where to fix it", () => {
    show(lockFor({ noModel: true })?.lock);
    expect(
      screen.getByText("No AI model is set up for your organization yet."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open Agent Control" })).toBeInTheDocument();
  });

  it("locks on a spent allowance, with the count under the composer", () => {
    const lock = lockFor({
      budget: budget({ near: false, person: { used: 1000, limit: 1000, resetsAt: nov1 } }),
    });
    show(lock?.lock);
    expect(screen.getByText("You've used your AI allowance for this period.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Request more" })).toBeInTheDocument();
    expect(lock?.note as string).toMatch(/1,000 of 1,000 messages · resets Nov(ember)? 1/u);
  });

  it("locks on a spent budget with the month and the reset date", () => {
    show(lockFor({ budget: budget({ budgetUsed: true, share: 1.02 }) })?.lock);
    expect(
      screen.getByText("October's budget for Billing Specialist is used up."),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/It resets Nov(ember)? 1, or an admin can raise it\./u),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ask an admin" })).toBeInTheDocument();
  });

  it("locks on the daily limit, which lifts at midnight", () => {
    show(lockFor({ budget: budget({ dailyUsed: true, near: false }) })?.lock);
    expect(
      screen.getByText("Billing Specialist has reached today's 200-request limit."),
    ).toBeInTheDocument();
    expect(screen.getByText("It resets at midnight.")).toBeInTheDocument();
  });
});

describe("DeskUsageMeter", () => {
  it("shows the share and the dollars when the budget is nearly spent", () => {
    render(<DeskUsageMeter budget={budget()} />);
    expect(screen.getByRole("status")).toHaveTextContent(
      "Billing Specialist has used 92% of October's budget",
    );
    expect(screen.getByRole("status")).toHaveTextContent("$460 / $500");
  });

  it("stays away until nine tenths of the budget is spent", () => {
    const { container } = render(<DeskUsageMeter budget={budget({ near: false, share: 0.5 })} />);
    expect(container).toBeEmptyDOMElement();
  });
});
