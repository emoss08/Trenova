import type { AgentControl } from "@/lib/graphql/agent-control";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { OrganizationWide } from "../organization-wide";

afterEach(cleanup);

/* AgentControl as the agentControl query serves it: limits are Decimals in the functional currency. */
function control(overrides: Partial<AgentControl> = {}): AgentControl {
  return {
    earnedAutonomy: false,
    promotionThreshold: 10,
    learningOff: false,
    personMonthlyMessages: 0,
    agentMoneyCallCeiling: "1000.00",
    agentMoneyRunCalls: 10,
    agentMoneyRunAmount: "5000.00",
    agentMoneyDayCalls: 50,
    agentMoneyDayAmount: "25000.00",
    agentMoneyCurrency: "CAD",
    aiTrainingConsent: false,
    shadowMode: false,
    version: 1,
    ...overrides,
  } as AgentControl;
}

function renderRows(value: AgentControl) {
  render(
    <OrganizationWide
      control={value}
      routing={null}
      canEdit
      onEdit={vi.fn()}
      onOpenRouting={vi.fn()}
    />,
  );
}

describe("OrganizationWide money row", () => {
  it("states the per-change limit in the organization's functional currency", () => {
    renderRows(control());

    expect(screen.getByText("Up to CA$1,000.00 a change")).toBeInTheDocument();
  });

  it("says a person approves every change when the per-change limit or the run count is 0", () => {
    renderRows(control({ agentMoneyCallCeiling: "0" }));
    expect(screen.getByText("A person approves every change")).toBeInTheDocument();
    cleanup();

    renderRows(control({ agentMoneyRunCalls: 0 }));
    expect(screen.getByText("A person approves every change")).toBeInTheDocument();
  });
});
