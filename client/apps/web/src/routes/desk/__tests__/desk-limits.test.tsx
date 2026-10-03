import { DeskLimitNote } from "@/routes/desk/_components/desk-limits";
import type { ThreadBudget } from "@/types/assistant";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

const nov1 = Date.UTC(2026, 10, 1) / 1000;

function budget(overrides: Partial<ThreadBudget> = {}): ThreadBudget {
  return {
    agentName: "Dispatch",
    spentUsd: "10.00",
    limitUsd: "50.00",
    share: 0.2,
    near: false,
    monthStart: Date.UTC(2026, 9, 1) / 1000,
    resetsAt: nov1,
    runsToday: 3,
    dailyRunLimit: 0,
    person: null,
    ...overrides,
  };
}

describe("DeskLimitNote", () => {
  it("stays out of the way while every cap has room", () => {
    const { container } = render(<DeskLimitNote budget={budget()} timezone="UTC" />);

    expect(container).toBeEmptyDOMElement();
  });

  it("warns when the agent's budget is nearly spent", () => {
    render(<DeskLimitNote budget={budget({ share: 0.93, near: true })} timezone="UTC" />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "Dispatch has used 93% of its monthly budget",
    );
  });

  it("names the person's own allowance first, with what is left", () => {
    render(
      <DeskLimitNote
        budget={budget({
          share: 0.95,
          near: true,
          person: { used: 230, limit: 250, resetsAt: nov1 },
        })}
        timezone="America/Los_Angeles"
      />,
    );

    const note = screen.getByRole("status");
    expect(note).toHaveTextContent("20 questions left in your allowance this month");
    // The month rolls over in UTC, so the date is not pulled back a day.
    expect(note).toHaveTextContent(/Nov(ember)? 1/);
  });

  it("leaves a spent cap to the refusal card rather than warning about it", () => {
    const { container } = render(
      <DeskLimitNote budget={budget({ share: 1.02, near: true })} timezone="UTC" />,
    );

    expect(container).toBeEmptyDOMElement();
  });
});
