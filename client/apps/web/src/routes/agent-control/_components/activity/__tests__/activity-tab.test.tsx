import type { AgentActivitySummary } from "@/lib/graphql/agent-activity";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ActivityView } from "../../rail-items";

const fetchAgentActivitySummary = vi.fn<(since: number) => Promise<AgentActivitySummary>>();

vi.mock("@/lib/graphql/agent-activity", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-activity")>()),
  fetchAgentActivitySummary: (since: number) => fetchAgentActivitySummary(since),
}));

// The tables have their own tests; here each only has to be mounted.
vi.mock("../agent-run-table", () => ({ default: () => <p>Runs table</p> }));
vi.mock("../agent-proposal-table", () => ({ default: () => <p>Proposals table</p> }));
vi.mock("../agent-plan-table", () => ({ default: () => <p>Plans table</p> }));
vi.mock("../agent-evaluation-table", () => ({ default: () => <p>Evaluations table</p> }));
vi.mock("../agent-exception-table", () => ({ default: () => <p>Exceptions table</p> }));

const { default: ActivityTab } = await import("../activity-tab");

const NOW = 1_800_007_200;

const summary: AgentActivitySummary = {
  since: 1_800_000_000,
  runs: 14,
  runsFailed: 2,
  runsWorking: 1,
  runsAwaiting: 3,
  pendingProposals: 4,
  oldestPendingAt: NOW - 3600,
  openExceptions: 2,
  decisionWindowDays: 7,
  decided: 8,
  approvedAsProposed: 0.75,
};

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  vi.spyOn(Date, "now").mockReturnValue(NOW * 1000);
  fetchAgentActivitySummary.mockReset();
});

afterEach(() => {
  restoreLayout();
  cleanup();
  vi.restoreAllMocks();
});

function renderTab(view: ActivityView, onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter hasMemory onUrlUpdate={onUrlUpdate}>
        {children}
      </NuqsTestingAdapter>
    </QueryClientProvider>
  );
  return render(<ActivityTab view={view} />, { wrapper });
}

describe("ActivityTab", () => {
  it("says what ran today and what waits on a person", async () => {
    fetchAgentActivitySummary.mockResolvedValue(summary);
    renderTab("runs");

    expect(await screen.findByRole("link", { name: "2 failed" })).toBeTruthy();
    expect(screen.getByText("14 runs")).toBeTruthy();
    expect(screen.getByRole("link", { name: "4 proposals wait" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "2 exceptions are" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Decide 4 proposals" })).toBeTruthy();
    expect(screen.getByText(/^Oldest waiting/)).toBeTruthy();
    expect(screen.getByText("Runs table")).toBeTruthy();
    expect(fetchAgentActivitySummary.mock.calls[0]?.[0]).toBeLessThanOrEqual(NOW);
  });

  it("shows today's figures and how often people approve as proposed", async () => {
    fetchAgentActivitySummary.mockResolvedValue(summary);
    renderTab("runs");

    const figures = await screen.findByRole("region", { name: "Activity figures" });
    expect(within(figures).getByText("14")).toBeTruthy();
    expect(within(figures).getByText("1 running now")).toBeTruthy();
    expect(within(figures).getByText("2")).toBeTruthy();
    expect(within(figures).getByText("4")).toBeTruthy();
    expect(within(figures).getByText("75%")).toBeTruthy();
    expect(within(figures).getByText("last 7 days")).toBeTruthy();
  });

  it("opens today's failed runs, the waiting proposals and the open exceptions", async () => {
    fetchAgentActivitySummary.mockResolvedValue(summary);
    const updates: URLSearchParams[] = [];
    renderTab("runs", (event) => updates.push(event.searchParams));

    await userEvent.click(await screen.findByRole("link", { name: "2 failed" }));
    await waitFor(() => expect(updates.at(-1)?.get("fieldFilters")).toContain("Failed"));
    expect(updates.at(-1)?.get("activity") ?? "runs").toBe("runs");
    expect(updates.at(-1)?.get("fieldFilters")).toContain("createdAt");

    await userEvent.click(screen.getByRole("button", { name: "Decide 4 proposals" }));
    await waitFor(() => expect(updates.at(-1)?.get("activity")).toBe("proposals"));
    expect(updates.at(-1)?.get("fieldFilters")).toContain("Pending");

    await userEvent.click(screen.getByRole("link", { name: "2 exceptions are" }));
    await waitFor(() => expect(updates.at(-1)?.get("activity")).toBe("exceptions"));
    expect(updates.at(-1)?.get("fieldFilters")).toContain("InReview");
  });

  it("says nothing of proposals or exceptions the reader may not read", async () => {
    fetchAgentActivitySummary.mockResolvedValue({
      ...summary,
      runsFailed: 0,
      pendingProposals: null,
      oldestPendingAt: null,
      openExceptions: null,
      decided: null,
      approvedAsProposed: null,
    });
    renderTab("runs");

    expect(await screen.findByText("14 runs")).toBeTruthy();
    expect(screen.queryByText(/wait/)).toBeNull();
    expect(screen.queryByText(/exception/)).toBeNull();
    expect(screen.queryByRole("button", { name: /^Decide/ })).toBeNull();
    const figures = screen.getByRole("region", { name: "Activity figures" });
    expect(within(figures).queryByText("Awaiting a decision")).toBeNull();
  });

  it("says when nothing waits on a person", async () => {
    fetchAgentActivitySummary.mockResolvedValue({
      ...summary,
      pendingProposals: 0,
      oldestPendingAt: null,
      openExceptions: 0,
    });
    renderTab("proposals");

    expect(await screen.findByText(/Nothing waits on a person\./)).toBeTruthy();
    expect(screen.queryByRole("button", { name: /^Decide/ })).toBeNull();
    expect(screen.getByText("Proposals table")).toBeTruthy();
  });

  it("says when the day could not be loaded, and still shows the table", async () => {
    fetchAgentActivitySummary.mockRejectedValue(new Error("down"));
    renderTab("exceptions");

    expect(
      await screen.findByText("What agents did today could not be loaded. Try again shortly."),
    ).toBeTruthy();
    expect(screen.getByText("Exceptions table")).toBeTruthy();
  });
});
