import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueService } from "./billing-queue";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const run = {
  id: "bqar_1",
  action: "Hold",
  status: "Scheduled",
  totalCount: 1,
  succeededCount: 0,
  failedCount: 0,
  skippedCount: 0,
  commitAt: 1_789_000_030,
  cancelRequestedAt: null,
  failureMessage: "",
  items: [],
};

function lastRequest() {
  return vi.mocked(requestGraphQL).mock.calls.at(-1)?.[0];
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("BillingQueueService.updateCharges", () => {
  it("sends each accessorial's amount as a decimal string", async () => {
    vi.mocked(requestGraphQL).mockRejectedValue(new Error("stop after the request"));

    await expect(
      new BillingQueueService().updateCharges("bqi_1", {
        formulaTemplateId: "ft_1",
        additionalCharges: [{ accessorialChargeId: "acc_1", method: "Flat", amount: 75, unit: 2 }],
      }),
    ).rejects.toThrow("stop after the request");

    expect(lastRequest()).toMatchObject({
      operationName: "UpdateBillingQueueCharges",
      variables: {
        id: "bqi_1",
        input: {
          formulaTemplateId: "ft_1",
          additionalCharges: [
            { id: undefined, accessorialChargeId: "acc_1", method: "Flat", amount: "75", unit: 2 },
          ],
        },
      },
    });
  });
});

describe("BillingQueueService.startBulk", () => {
  it("starts the run with its idempotency key and parses it", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ startBillingQueueBulkRun: run });

    const started = await new BillingQueueService().startBulk("Hold", ["bqi_1"], "key-1", {
      holdReasonCode: "CustomerDispute",
      reason: "Disputed rate",
    });

    expect(lastRequest()).toMatchObject({
      operationName: "StartBillingQueueBulkRun",
      variables: {
        input: {
          action: "Hold",
          itemIds: ["bqi_1"],
          idempotencyKey: "key-1",
          params: { holdReasonCode: "CustomerDispute", reason: "Disputed rate" },
        },
      },
    });
    expect(started.status).toBe("Scheduled");
    expect(started.items).toEqual([]);
  });
});

describe("BillingQueueService.getNovaGeneration", () => {
  it("resolves to null when nothing is generating", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ billingQueueNovaGeneration: null });

    await expect(new BillingQueueService().getNovaGeneration()).resolves.toBeNull();
    expect(lastRequest()?.operationName).toBe("BillingQueueNovaGeneration");
  });
});

describe("BillingQueueService.listFilterPresets", () => {
  it("resolves to the caller's presets", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      billingQueueFilterPresets: [
        {
          id: "bqfp_1",
          organizationId: "org_1",
          businessUnitId: "bu_1",
          userId: "usr_1",
          name: "Disputes",
          filters: { status: "OnHold" },
          isDefault: true,
          version: 1,
          createdAt: 1_789_000_000,
          updatedAt: 1_789_000_000,
        },
      ],
    });

    const presets = await new BillingQueueService().listFilterPresets();

    expect(presets).toHaveLength(1);
    expect(presets[0].filters).toEqual({ status: "OnHold" });
  });
});

describe("BillingQueueService.getActivity", () => {
  it("continues before the given entry", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      billingQueueItemActivity: { items: [], hasMore: false, total: 0 },
    });

    await new BillingQueueService().getActivity("bqi_1", { at: 1_789_000_000, id: "bqe_9" }, 10);

    expect(lastRequest()).toMatchObject({
      operationName: "BillingQueueItemActivity",
      variables: { id: "bqi_1", limit: 10, before: { at: 1_789_000_000, id: "bqe_9" } },
    });
  });
});
