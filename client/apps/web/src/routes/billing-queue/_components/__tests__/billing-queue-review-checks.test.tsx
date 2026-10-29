import { billingQueue } from "@/lib/queries/billing-queue";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type {
  BillingCheck,
  BillingIssue,
  BillingQueueItem,
} from "@trenova/shared/types/billing-queue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueReviewChecks } from "../billing-queue-review-checks";

const mocks = vi.hoisted(() => ({
  resolveIssue: vi.fn(),
  undoIssue: vi.fn(),
  canUpdate: true,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canUpdate, isLoading: false }),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    billingQueueService: { resolveIssue: mocks.resolveIssue, undoIssue: mocks.undoIssue },
  },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.canUpdate = true;
});

const passing = (key: BillingCheck["key"], code: string): BillingCheck => ({
  key,
  state: "ok",
  code,
  detail: "",
  issueId: null,
  facts: null,
});

const fuelIssue: BillingIssue = {
  id: "bqis_fuel",
  itemId: "bqi_1",
  checkKey: "charges",
  code: "AccessorialNotOnRateCon",
  subjectKey: "acg_fuel",
  summary: "$1,037.45 fuel surcharge isn't on the rate con",
  reasoning:
    "Miller & Rose Industries's rate con doesn't price a fuel surcharge. It was added to the shipment by hand, so bill it only if the contract passes it through or there's a receipt for it.",
  source: "Deterministic",
  flaggedChargeId: "acg_fuel",
  options: [
    {
      key: "keep",
      label: "Keep it · bill $1,037.45",
      amount: "1037.45",
      done: "Fuel surcharge kept",
      effect: { kind: "keep", chargeId: "acg_fuel" },
    },
    {
      key: "drop",
      label: "Remove it",
      amount: null,
      done: "Fuel surcharge removed",
      effect: { kind: "drop", chargeId: "acg_fuel" },
    },
  ],
  resolutionKey: null,
  resolutionText: null,
  resolvedById: null,
  resolvedAt: null,
  requestedAt: null,
  undoable: false,
  createdAt: 1_800_000_000,
};

const settledFuel: BillingIssue = {
  ...fuelIssue,
  resolutionKey: "keep",
  resolutionText: "Fuel surcharge kept",
  resolvedById: "usr_1",
  resolvedAt: 1_800_000_100,
  undoable: true,
};

function item({
  status = "InReview",
  checks,
  issues,
  blocker = "",
}: {
  status?: BillingQueueItem["status"];
  checks: BillingCheck[];
  issues: BillingIssue[];
  blocker?: string;
}): BillingQueueItem {
  return {
    id: "bqi_1",
    shipmentId: "shp_1",
    status,
    detentionHolds: [],
    ratingReviewHold: null,
    review: {
      checks,
      issues,
      needsCount: checks.filter((check) => check.state !== "ok").length,
      ready: checks.every((check) => check.state === "ok"),
      blocker,
      documents: [],
      billerName: "Dana Whitfield",
    },
  } as unknown as BillingQueueItem;
}

const flagged = item({
  checks: [
    passing("biller", "assigned"),
    { key: "charges", state: "fail", code: "issue", detail: "", issueId: "bqis_fuel", facts: null },
    passing("pod", "not_required"),
    passing("terms", "terms"),
    passing("duplicate", "unique"),
  ],
  issues: [fuelIssue],
  blocker: "issue",
});

const cleared = item({
  checks: [
    passing("biller", "assigned"),
    passing("charges", "resolved"),
    passing("pod", "not_required"),
    passing("terms", "terms"),
    passing("duplicate", "unique"),
  ],
  issues: [settledFuel],
});

function renderChecks(value: BillingQueueItem) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  const invalidate = vi.spyOn(client, "invalidateQueries");
  render(
    <QueryClientProvider client={client}>
      <BillingQueueReviewChecks item={value} />
    </QueryClientProvider>,
  );
  return { client, invalidate };
}

describe("BillingQueueReviewChecks", () => {
  it("shows nothing for an item that is not in review, or was read without its review", () => {
    renderChecks({ ...flagged, status: "ReadyForReview" });
    expect(screen.queryByTestId("billing-queue-review-checks")).not.toBeInTheDocument();

    renderChecks({ ...flagged, review: null } as BillingQueueItem);
    expect(screen.queryByTestId("billing-queue-review-checks")).not.toBeInTheDocument();
  });

  it("names each flagged check with what was found, why, and the choices that settle it", () => {
    renderChecks(flagged);

    expect(screen.getByText("1 check needs you before this can be approved")).toBeInTheDocument();
    const check = screen.getByTestId("billing-queue-check-charges");
    expect(check).toHaveTextContent(
      "Charges match the rate con · $1,037.45 fuel surcharge isn't on the rate con",
    );
    expect(check).toHaveTextContent("rate con doesn't price a fuel surcharge");
    expect(within(check).getByRole("button", { name: "Keep it · bill $1,037.45" })).toBeEnabled();
    expect(within(check).getByRole("button", { name: "Remove it" })).toBeEnabled();
    expect(screen.queryByTestId("billing-queue-check-pod")).not.toBeInTheDocument();
  });

  it("settles the check with the chosen option and puts the answered item in the cache", async () => {
    mocks.resolveIssue.mockResolvedValue(cleared);
    const { client, invalidate } = renderChecks(flagged);

    await userEvent.click(screen.getByRole("button", { name: "Keep it · bill $1,037.45" }));

    await waitFor(() =>
      expect(client.getQueryData(billingQueue.get("bqi_1").queryKey)).toEqual(cleared),
    );
    expect(mocks.resolveIssue).toHaveBeenCalledWith("bqi_1", "bqis_fuel", "keep");
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billingQueue"] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billing-queue-list"] });
  });

  it("says every check is clear, and offers to undo what was settled", async () => {
    mocks.undoIssue.mockResolvedValue(flagged);
    renderChecks(cleared);

    expect(screen.getByText("All five checks clear")).toBeInTheDocument();
    expect(screen.getByText("Settled · Fuel surcharge kept")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Undo" }));

    await waitFor(() => expect(mocks.undoIssue).toHaveBeenCalledWith("bqi_1", "bqis_fuel"));
  });

  it("shows the checks without the choices to someone who can't update the queue", () => {
    mocks.canUpdate = false;
    renderChecks(flagged);

    expect(screen.getByTestId("billing-queue-check-charges")).toHaveTextContent(
      "$1,037.45 fuel surcharge isn't on the rate con",
    );
    expect(screen.queryByRole("button", { name: "Remove it" })).not.toBeInTheDocument();
  });
});
