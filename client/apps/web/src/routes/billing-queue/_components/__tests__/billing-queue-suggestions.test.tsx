import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
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

function renderSuggestions(
  recommendations: BillingQueueRecommendation[],
  run = vi.fn().mockResolvedValue({ id: "run_1" }),
) {
  mocks.recommendations.mockResolvedValue(recommendations);
  const onOpenItem = vi.fn();
  const onHoldItems = vi.fn();
  const client = new QueryClient();
  const view = (revertedRunIds: ReadonlySet<string>) => (
    <QueryClientProvider client={client}>
      <BillingQueueSuggestions
        run={run}
        onOpenItem={onOpenItem}
        onHoldItems={onHoldItems}
        isStarting={false}
        revertedRunIds={revertedRunIds}
      />
    </QueryClientProvider>
  );
  const { rerender } = render(view(new Set()));
  return {
    run,
    onOpenItem,
    onHoldItems,
    revert: (...runIds: string[]) => rerender(view(new Set(runIds))),
  };
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

  // The batch's items now belong to its run; offering them again would start a
  // second run over the same freight.
  it("takes a suggestion away once its batch has started", async () => {
    const { run } = renderSuggestions([
      recommendation("ApproveReady", ["bqi_1"]),
      recommendation("AssignToMe", ["bqi_2"]),
    ]);

    await userEvent.click(await screen.findByRole("button", { name: "Approve 1" }));

    await waitFor(() =>
      expect(screen.queryByRole("button", { name: "Approve 1" })).not.toBeInTheDocument(),
    );
    expect(screen.getByRole("button", { name: "Assign 1" })).toBeEnabled();
    expect(run).toHaveBeenCalledTimes(1);
  });

  it("shows the pressed button working and holds the others while the batch starts", async () => {
    let accept: (run: { id: string }) => void = () => {};
    const run = vi.fn(
      () =>
        new Promise<{ id: string }>((resolve) => {
          accept = resolve;
        }),
    );
    renderSuggestions(
      [recommendation("ApproveReady", ["bqi_1"]), recommendation("AssignToMe", ["bqi_2"])],
      run,
    );

    const approve = await screen.findByRole("button", { name: "Approve 1" });
    await userEvent.click(approve);

    expect(approve).toHaveAttribute("aria-busy", "true");
    expect(approve).toBeDisabled();
    expect(screen.getByRole("button", { name: "Assign 1" })).toBeDisabled();

    accept({ id: "run_1" });
    await waitFor(() => expect(approve).not.toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Assign 1" })).toBeEnabled();
  });

  it("brings a suggestion back when its batch is undone", async () => {
    const { revert } = renderSuggestions([recommendation("ApproveReady", ["bqi_1"])]);

    await userEvent.click(await screen.findByRole("button", { name: "Approve 1" }));
    await waitFor(() =>
      expect(screen.queryByRole("button", { name: "Approve 1" })).not.toBeInTheDocument(),
    );

    revert("run_1");

    expect(await screen.findByRole("button", { name: "Approve 1" })).toBeEnabled();
  });

  it("keeps a suggestion whose batch could not start, so it can be tried again", async () => {
    const run = vi.fn().mockRejectedValue(new Error("Conflict"));
    renderSuggestions([recommendation("PostApproved", ["bqi_1"])], run);

    await userEvent.click(await screen.findByRole("button", { name: "Post 1" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Post 1" })).toBeEnabled());
  });
});
