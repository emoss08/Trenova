import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { BillingQueueStats } from "@trenova/shared/types/billing-queue";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueCaughtUp } from "../billing-queue-caught-up";

const mocks = vi.hoisted(() => ({
  assist: vi.fn(),
  recommendations: vi.fn(),
  start: vi.fn(),
  canPost: true,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canPost, isLoading: false }),
}));

vi.mock("../use-bulk-run", () => ({
  useBulkRun: () => ({ start: mocks.start, isStarting: false }),
}));

vi.mock("@/services/billing-queue", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/billing-queue")>()),
  billingQueueService: {
    getAssist: mocks.assist,
    getRecommendations: mocks.recommendations,
  },
}));

function statsWith(lanes: BillingQueueStats["lanes"]): BillingQueueStats {
  return {
    lanes,
    openCount: lanes.reduce((sum, lane) => sum + lane.count, 0),
    openTotal: 0,
    pulse: null,
  };
}

function renderCaughtUp(stats: BillingQueueStats, onShowApproved = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <BillingQueueCaughtUp stats={stats} onShowApproved={onShowApproved} />
    </QueryClientProvider>,
  );
  return onShowApproved;
}

beforeEach(() => {
  mocks.canPost = true;
  mocks.assist.mockResolvedValue({ review: false, brief: false, search: false });
  mocks.recommendations.mockResolvedValue([]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("BillingQueueCaughtUp", () => {
  it("is the queue's empty sheet saying everything was approved and posted, with nothing to do", async () => {
    renderCaughtUp(statsWith([]));

    expect(screen.getByRole("region", { name: "You're all caught up" })).toHaveTextContent(
      "Every shipment that reached billing has been approved and posted.",
    );
    expect(screen.queryByRole("img", { name: "Done" })).not.toBeInTheDocument();
    await vi.waitFor(() => expect(mocks.recommendations).toHaveBeenCalled());
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("says how many approved drafts wait to post and offers to see and post them", async () => {
    mocks.recommendations.mockResolvedValue([
      {
        kind: "PostApproved",
        source: "Rules",
        count: 2,
        total: 900,
        itemIds: ["bqi_1", "bqi_2"],
      },
    ]);
    const onShowApproved = renderCaughtUp(statsWith([{ lane: "Approved", count: 2, total: 900 }]));

    expect(
      screen.getByText(
        "Every shipment that reached billing has been approved. 2 approved drafts are still waiting to be posted.",
      ),
    ).toBeInTheDocument();

    await userEvent.click(await screen.findByRole("button", { name: "Post 2 drafts" }));
    expect(mocks.start).toHaveBeenCalledWith("Post", ["bqi_1", "bqi_2"]);

    await userEvent.click(screen.getByRole("button", { name: "See what was approved" }));
    expect(onShowApproved).toHaveBeenCalledOnce();
  });

  it("does not offer to post drafts to someone who may not post invoices", async () => {
    mocks.canPost = false;
    mocks.recommendations.mockResolvedValue([
      { kind: "PostApproved", source: "Rules", count: 1, total: 450, itemIds: ["bqi_1"] },
    ]);
    renderCaughtUp(statsWith([{ lane: "Approved", count: 1, total: 450 }]));

    expect(
      await screen.findByRole("button", { name: "See what was approved" }),
    ).toBeInTheDocument();
    await vi.waitFor(() => expect(mocks.recommendations).toHaveBeenCalled());
    expect(screen.queryByRole("button", { name: /Post/u })).not.toBeInTheDocument();
  });

  it("notes that Nova reviews what arrives next where Nova reviews the queue", async () => {
    mocks.assist.mockResolvedValue({ review: true, brief: true, search: false });
    renderCaughtUp(statsWith([]));

    expect(
      await screen.findByText(/reviews each shipment as it reaches billing/u),
    ).toBeInTheDocument();
  });
});
