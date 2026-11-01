import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceRunService } from "./invoice-run";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe("InvoiceRunService.billStatementNow", () => {
  it("bills the statement with each held item excluded and parses the outcome", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      billStatementNow: {
        runId: "ivr_1",
        runNumber: "RUN-7",
        results: [
          {
            groupId: "g_1",
            groupLabel: "Acme",
            success: true,
            skipped: false,
            invoiceId: "inv_1",
            invoiceNumber: "INV-1",
            error: "",
          },
        ],
        totalCount: 1,
        successCount: 1,
        skippedCount: 0,
        errorCount: 0,
      },
    });

    const result = await new InvoiceRunService().billStatementNow("cus_1", "Early close", [
      "bqi_held",
    ]);

    expect(vi.mocked(requestGraphQL).mock.calls[0][0]).toMatchObject({
      operationName: "BillStatementNow",
      variables: {
        customerId: "cus_1",
        input: {
          reason: "Early close",
          exclude: [
            {
              billingQueueItemId: "bqi_held",
              reason: "Held back by the biller before this statement billed",
            },
          ],
        },
      },
    });
    expect(result.successCount).toBe(1);
    expect(result.results[0].error).toBeNull();
  });
});
