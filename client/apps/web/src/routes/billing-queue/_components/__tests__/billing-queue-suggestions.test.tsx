import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { BillingQueueRecommendation } from "@trenova/shared/types/billing-queue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueSuggestions } from "../billing-queue-suggestions";

const mocks = vi.hoisted(() => ({ recommendations: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { getRecommendations: mocks.recommendations } },
}));
vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (select: (state: { user: { id: string } }) => unknown) =>
    select({ user: { id: "usr_me" } }),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function recommendation(
  kind: BillingQueueRecommendation["kind"],
  itemIds: string[],
  source: BillingQueueRecommendation["source"] = "Rules",
): BillingQueueRecommendation {
  return { kind, source, count: itemIds.length, total: itemIds.length * 1000, itemIds };
}

function renderSuggestions(recommendations: BillingQueueRecommendation[]) {
  mocks.recommendations.mockResolvedValue(recommendations);
  const run = vi.fn().mockResolvedValue({});
  const onOpenItem = vi.fn();
  const onHoldItems = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <BillingQueueSuggestions
        run={run}
        onOpenItem={onOpenItem}
        onHoldItems={onHoldItems}
        isStarting={false}
      />
    </QueryClientProvider>,
  );
  return { run, onOpenItem, onHoldItems };
}

describe("BillingQueueSuggestions", () => {
  it("leads with the batch Nova agreed with and drops the plain approve-every-ready batch", async () => {
    const { run } = renderSuggestions([
      recommendation("ApproveAgentAgreed", ["bqi_1", "bqi_2"], "Agent"),
      recommendation("ApproveReady", ["bqi_1", "bqi_2", "bqi_3"]),
    ]);

    expect(await screen.findByText("Approve the 2 Nova checked")).toBeInTheDocument();
    expect(screen.queryByText("Approve 3 ready items")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Approve 2" }));
    expect(run).toHaveBeenCalledWith("Approve", ["bqi_1", "bqi_2"], { assignApprover: true });
  });

  it("offers approving every ready item when Nova did not review", async () => {
    renderSuggestions([recommendation("ApproveReady", ["bqi_3"])]);
    expect(await screen.findByText("Approve 1 ready item")).toBeInTheDocument();
  });

  it("assigns the unassigned batch to the person looking at it", async () => {
    const { run } = renderSuggestions([recommendation("AssignToMe", ["bqi_4", "bqi_5"])]);
    await userEvent.click(await screen.findByRole("button", { name: "Assign 2" }));
    expect(run).toHaveBeenCalledWith("Assign", ["bqi_4", "bqi_5"], { billerId: "usr_me" });
  });

  it("asks for a hold reason before holding possible duplicates", async () => {
    const { run, onHoldItems } = renderSuggestions([recommendation("HoldDuplicates", ["bqi_6"])]);
    await userEvent.click(await screen.findByRole("button", { name: "Hold 1" }));
    expect(onHoldItems).toHaveBeenCalledWith(["bqi_6"], "Possible duplicate invoice");
    expect(run).not.toHaveBeenCalled();
  });

  it("opens the first item of a batch that needs a look rather than acting on it", async () => {
    const { run, onOpenItem } = renderSuggestions([
      recommendation("ReviewAgentConcerns", ["bqi_7", "bqi_8"], "Agent"),
    ]);
    await userEvent.click(await screen.findByRole("button", { name: "Review 2" }));
    expect(onOpenItem).toHaveBeenCalledWith("bqi_7");
    expect(run).not.toHaveBeenCalled();
  });

  it("shows nothing when there is nothing to batch", () => {
    renderSuggestions([]);
    expect(screen.queryByRole("region", { name: "Suggested" })).not.toBeInTheDocument();
  });
});
