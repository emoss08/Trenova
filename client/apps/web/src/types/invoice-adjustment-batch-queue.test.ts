import { describe, expect, it } from "vitest";
import { invoiceAdjustmentBatchQueueItemSchema } from "@/types/invoice-adjustment";

/**
 * GET /billing/invoice-adjustments/batches/ lists invoiceadjustment.InvoiceAdjustmentBatch
 * rows: the batch's own `id` plus the scan-only queue projections (pendingCount,
 * submittedByName, lastFailure, lastFailureCount). The fixture follows those
 * json tags, including the nullzero submitter a system-submitted batch has.
 */
const batchRow = {
  id: "iab_01K99ZRDA85S8ZXERVM2F2W3B8",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  idempotencyKey: "bulk-credit-2026-10-14",
  status: "PartialSuccess",
  totalCount: 12,
  processedCount: 12,
  succeededCount: 9,
  failedCount: 3,
  submittedById: null,
  submittedAt: 1_760_400_000,
  submittedByName: "",
  pendingCount: 0,
  lastFailure: "Invoice has applied payments",
  lastFailureCount: 2,
  metadata: {},
  version: 4,
  createdAt: 1_760_400_000,
  updatedAt: 1_760_400_900,
};

describe("invoiceAdjustmentBatchQueueItemSchema", () => {
  it("parses a batch row as the batch list endpoint sends it", () => {
    const parsed = invoiceAdjustmentBatchQueueItemSchema.parse(batchRow);

    expect(parsed.id).toBe("iab_01K99ZRDA85S8ZXERVM2F2W3B8");
    expect(parsed.status).toBe("PartialSuccess");
    expect(parsed.pendingCount).toBe(0);
    expect(parsed.lastFailure).toBe("Invoice has applied payments");
    expect(parsed.submittedById).toBeNull();
  });
});
