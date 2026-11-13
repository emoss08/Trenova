import type { AttentionSummary } from "@/config/attention-rows";
import type { ShipmentAnalyticsData } from "@/lib/shipment-analytics";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { briefQueues, HomeBrief, type HomeBriefProps } from "../home-brief";

const attention = {
  billingQueue: 446,
  pendingApprovals: 0,
  reconciliationExceptions: 22,
  serviceFailures: 0,
  ediAttention: 5,
  agentDecisions: 0,
} as unknown as AttentionSummary;

const analytics = {
  unassigned: { count: 1213 },
  atRisk: { count: 0 },
  activeShipments: { count: 12 },
} as unknown as ShipmentAnalyticsData;

const same = (message: string) => message;

function renderBrief(overrides: Partial<HomeBriefProps> = {}) {
  return render(
    <MemoryRouter>
      <HomeBrief
        attention={attention}
        analytics={analytics}
        analyticsReady
        loading={false}
        {...overrides}
      />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  // 19:19 UTC is 2:19 in the afternoon in New York.
  vi.useFakeTimers({ now: new Date("2026-11-13T19:19:00Z"), toFake: ["Date"] });
  useAuthStore.setState({
    user: { name: "Dana Whitfield", timezone: "America/New_York", timeFormat: "12-hour" },
  } as never);
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  useAuthStore.setState({ user: null } as never);
});

describe("briefQueues", () => {
  it("lists freight first, then the queues in their order, and leaves out the empty ones", () => {
    expect(
      briefQueues(same, attention, analytics, true).map((queue) => [
        queue.label,
        queue.count,
        queue.href,
        queue.tone,
      ]),
    ).toEqual([
      ["Unassigned", 1213, "/shipment-management/shipments", "warning"],
      ["Billing queue", 446, "/billing/queue", "info"],
      ["Reconciliation exceptions", 22, "/billing/reconciliation-exceptions", "danger"],
      ["EDI needs attention", 5, "/edi/overview", "warning"],
    ]);
  });

  it("says nothing about freight until the shipment figures are real", () => {
    expect(briefQueues(same, attention, analytics, false).map((queue) => queue.key)).toEqual([
      "billingQueue",
      "reconciliationExceptions",
      "ediAttention",
    ]);
  });

  it("is empty when nothing is waiting or nothing has been read", () => {
    expect(briefQueues(same, undefined, analytics, false)).toEqual([]);
  });
});

describe("HomeBrief", () => {
  it("greets the person by first name for the time of day in their timezone", () => {
    renderBrief();

    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Good afternoon, Dana");
    expect(screen.getByText(/Friday, November 13 · 2:19 PM/)).toBeInTheDocument();
  });

  it("greets without a name rather than inventing one", () => {
    useAuthStore.setState({ user: { timezone: "America/New_York" } } as never);
    renderBrief();

    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent(/^Good afternoon$/);
  });

  it("sums what needs the person in one sentence over a strip where every figure is a link", () => {
    renderBrief();

    expect(screen.getByText("1,686 items need you across 4 queues.")).toBeInTheDocument();
    const strip = screen.getByRole("group", { name: "What needs you" });
    expect(
      within(strip)
        .getAllByRole("link")
        .map((link) => [link.textContent, link.getAttribute("href")]),
    ).toEqual([
      ["Unassigned1,213", "/shipment-management/shipments"],
      ["Billing queue446", "/billing/queue"],
      ["Reconciliation exceptions22", "/billing/reconciliation-exceptions"],
      ["EDI needs attention5", "/edi/overview"],
    ]);
  });

  it("speaks of one item in one queue in the singular", () => {
    renderBrief({
      attention: undefined,
      analytics: { ...analytics, unassigned: { count: 1 } } as never,
    });

    expect(screen.getByText("1 item needs you across 1 queue.")).toBeInTheDocument();
  });

  it("says the day is clear, with what is moving, and draws no strip", () => {
    renderBrief({
      attention: undefined,
      analytics: { ...analytics, unassigned: { count: 0 } } as never,
    });

    expect(
      screen.getByText("You're clear — 12 loads moving, nothing flagged."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("group", { name: "What needs you" })).not.toBeInTheDocument();
  });

  it("claims neither work nor a clear day while the counts are still loading", () => {
    renderBrief({ loading: true });

    expect(screen.queryByText(/need you|needs you|You're clear/)).not.toBeInTheDocument();
    expect(screen.queryAllByRole("link")).toEqual([]);
  });
});
