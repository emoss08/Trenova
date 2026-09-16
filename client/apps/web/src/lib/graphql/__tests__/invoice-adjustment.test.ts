import {
  ApproveInvoiceAdjustmentDocument,
  InvoiceAdjustmentApprovalDetailDocument,
  InvoiceAdjustmentApprovalsDocument,
  InvoiceAdjustmentOperationsSummaryDocument,
  RejectInvoiceAdjustmentDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  approveInvoiceAdjustment,
  fetchInvoiceAdjustmentOperationsSummary,
  fetchInvoiceApprovalDetail,
  listInvoiceAdjustmentApprovals,
  rejectInvoiceAdjustment,
} from "../invoice-adjustment";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables?: Record<string, unknown>;
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

function queueNode(adjustmentId: string, overrides: Record<string, unknown> = {}) {
  return {
    adjustmentId,
    originalInvoiceId: `inv_${adjustmentId}`,
    originalInvoiceNumber: `INV-${adjustmentId}`,
    originalInvoiceStatus: "Posted",
    customerName: "Acme Freight",
    kind: "CreditOnly",
    reason: "",
    policyReason: "",
    policySource: "Policy-controlled approval",
    creditTotalAmount: "125.5000",
    rebillTotalAmount: "0",
    netDeltaAmount: "-125.5000",
    wouldCreateUnappliedCredit: false,
    requiresReconciliationException: false,
    requiresReplacementInvoiceReview: false,
    submittedByName: "",
    submittedAt: null,
    creditMemoInvoiceId: null,
    replacementInvoiceId: null,
    rebillQueueItemId: null,
    batchId: null,
    ...overrides,
  };
}

describe("invoice adjustment GraphQL", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("pages the approval queue by cursor and flattens every edge in order", async () => {
    const first = queueNode("iadj_1", { kind: "WriteOff", submittedAt: 1_700_000_300 });
    const second = queueNode("iadj_2");
    requestGraphQLMock.mockResolvedValue({
      invoiceAdjustmentApprovals: {
        edges: [
          { cursor: "c1", node: first },
          { cursor: "c2", node: second },
        ],
        pageInfo: { hasNextPage: true, endCursor: "c2" },
      },
    });
    const controller = new AbortController();

    const page = await listInvoiceAdjustmentApprovals(
      { first: 20, after: "c0", query: "  INV-9  ", kind: "WriteOff" },
      { signal: controller.signal },
    );

    const call = lastCall();
    expect(call.document).toBe(InvoiceAdjustmentApprovalsDocument);
    expect(call.operationName).toBe("InvoiceAdjustmentApprovals");
    expect(call.signal).toBe(controller.signal);
    expect(call.variables).toEqual({
      input: { first: 20, after: "c0", query: "INV-9", kind: "WriteOff" },
    });
    expect(call.variables?.input).not.toHaveProperty("offset");
    expect(page.items).toEqual([first, second]);
    expect(page.pageInfo).toEqual({ hasNextPage: true, endCursor: "c2" });
  });

  it("leaves the cursor, search, and kind unset on an unfiltered first page", async () => {
    requestGraphQLMock.mockResolvedValue({
      invoiceAdjustmentApprovals: {
        edges: [],
        pageInfo: { hasNextPage: false, endCursor: null },
      },
    });

    const page = await listInvoiceAdjustmentApprovals({
      first: 20,
      after: null,
      query: "   ",
      kind: null,
    });

    expect(lastCall().variables).toEqual({
      input: { first: 20, after: undefined, query: undefined, kind: undefined },
    });
    expect(page).toEqual({ items: [], pageInfo: { hasNextPage: false, endCursor: null } });
  });

  it("reads an adjustment's lines and passes a missing adjustment through as null", async () => {
    const detail = {
      id: "iadj_1",
      status: "PendingApproval",
      lines: [
        {
          id: "l1",
          lineNumber: 1,
          description: "Linehaul",
          creditAmount: "100",
          rebillAmount: "0",
        },
        { id: "l2", lineNumber: 2, description: "Fuel", creditAmount: "25.5", rebillAmount: "0" },
      ],
    };
    requestGraphQLMock.mockResolvedValueOnce({ invoiceAdjustment: detail });
    const controller = new AbortController();

    await expect(
      fetchInvoiceApprovalDetail("iadj_1", { signal: controller.signal }),
    ).resolves.toEqual(detail);
    expect(lastCall().document).toBe(InvoiceAdjustmentApprovalDetailDocument);
    expect(lastCall().variables).toEqual({ id: "iadj_1" });
    expect(lastCall().signal).toBe(controller.signal);

    requestGraphQLMock.mockResolvedValueOnce({ invoiceAdjustment: null });
    await expect(fetchInvoiceApprovalDetail("iadj_gone")).resolves.toBeNull();
  });

  it("reads the operations summary with the caller's signal", async () => {
    const summary = {
      approvalsPending: 3,
      reconciliationPending: 1,
      writeOffPending: 0,
      failedBatchItems: 2,
    };
    requestGraphQLMock.mockResolvedValue({ invoiceAdjustmentOperationsSummary: summary });
    const controller = new AbortController();

    await expect(
      fetchInvoiceAdjustmentOperationsSummary({ signal: controller.signal }),
    ).resolves.toEqual(summary);
    expect(lastCall().document).toBe(InvoiceAdjustmentOperationsSummaryDocument);
    expect(lastCall().signal).toBe(controller.signal);
  });

  it("approves and rejects through GraphQL mutations", async () => {
    requestGraphQLMock.mockResolvedValueOnce({
      approveInvoiceAdjustment: { id: "iadj_1", status: "Executed", approvalStatus: "Approved" },
    });
    await expect(approveInvoiceAdjustment("iadj_1")).resolves.toEqual({
      id: "iadj_1",
      status: "Executed",
      approvalStatus: "Approved",
    });
    expect(lastCall().document).toBe(ApproveInvoiceAdjustmentDocument);
    expect(lastCall().variables).toEqual({ adjustmentId: "iadj_1" });

    const rejected = {
      id: "iadj_2",
      status: "Rejected",
      approvalStatus: "Rejected",
      rejectionReason: "Duplicate",
    };
    requestGraphQLMock.mockResolvedValueOnce({ rejectInvoiceAdjustment: rejected });
    await expect(
      rejectInvoiceAdjustment({ adjustmentId: "iadj_2", reason: "Duplicate" }),
    ).resolves.toEqual(rejected);
    expect(lastCall().document).toBe(RejectInvoiceAdjustmentDocument);
    expect(lastCall().variables).toEqual({
      input: { adjustmentId: "iadj_2", reason: "Duplicate" },
    });
  });
});
