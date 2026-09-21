import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoiceApprovalPage } from "../page";

const mocks = vi.hoisted(() => ({
  listApprovals: vi.fn(),
  getSummary: vi.fn(),
  getDetail: vi.fn(),
  approve: vi.fn(),
  reject: vi.fn(),
  rest: vi.fn(),
}));

vi.mock("@/lib/graphql/invoice-adjustment", () => ({
  listInvoiceAdjustmentApprovals: mocks.listApprovals,
  fetchInvoiceAdjustmentOperationsSummary: mocks.getSummary,
  fetchInvoiceApprovalDetail: mocks.getDetail,
  approveInvoiceAdjustment: mocks.approve,
  rejectInvoiceAdjustment: mocks.reject,
}));
vi.mock("@/services/api", () => ({
  apiService: {
    invoiceAdjustmentService: {
      listApprovals: mocks.rest,
      getSummary: mocks.rest,
      getById: mocks.rest,
      approve: mocks.rest,
      reject: mocks.rest,
    },
  },
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));
vi.mock("@/components/billing/billing-workspace-layout", () => ({
  BillingWorkspaceLayout: ({
    toolbar,
    sidebar,
    detail,
  }: {
    toolbar?: ReactNode;
    sidebar: ReactNode;
    detail: ReactNode;
  }) => (
    <div>
      {toolbar}
      <aside>{sidebar}</aside>
      <main>{detail}</main>
    </div>
  ),
}));

let intersect: ((entries: Array<{ isIntersecting: boolean }>) => void) | null = null;

function queueRow(adjustmentId: string, invoiceNumber: string, overrides = {}) {
  return {
    adjustmentId,
    originalInvoiceId: `inv_${adjustmentId}`,
    originalInvoiceNumber: invoiceNumber,
    originalInvoiceStatus: "Posted",
    customerName: "Acme Freight",
    kind: "CreditOnly",
    reason: "Rate keyed wrong",
    policyReason: "",
    policySource: "Policy-controlled approval",
    creditTotalAmount: "125.5",
    rebillTotalAmount: "0",
    netDeltaAmount: "-125.5",
    wouldCreateUnappliedCredit: false,
    requiresReconciliationException: false,
    requiresReplacementInvoiceReview: false,
    submittedByName: "Dana Reviewer",
    submittedAt: 1_700_000_000,
    creditMemoInvoiceId: null,
    replacementInvoiceId: null,
    rebillQueueItemId: null,
    batchId: null,
    ...overrides,
  };
}

function page(items: unknown[], endCursor: string | null, hasNextPage = endCursor !== null) {
  return { items, pageInfo: { hasNextPage, endCursor } };
}

function renderPage(searchParams = "") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams}>
          <InvoiceApprovalPage />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  intersect = null;
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      constructor(callback: (entries: Array<{ isIntersecting: boolean }>) => void) {
        intersect = callback;
      }
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
  mocks.rest.mockRejectedValue(new Error("REST must not be called from the approval page"));
  mocks.getSummary.mockResolvedValue({
    approvalsPending: 4,
    reconciliationPending: 1,
    writeOffPending: 2,
    failedBatchItems: 3,
  });
  mocks.getDetail.mockResolvedValue({
    id: "iadj_1",
    status: "PendingApproval",
    lines: [
      { id: "l1", lineNumber: 1, description: "Linehaul", creditAmount: "100", rebillAmount: "0" },
      { id: "l2", lineNumber: 2, description: "Fuel", creditAmount: "25.5", rebillAmount: "0" },
    ],
  });
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("pending approvals over GraphQL", () => {
  it("reads the queue, the open adjustment, and the summary without any REST call", async () => {
    mocks.listApprovals.mockResolvedValue(page([queueRow("iadj_1", "INV-1001")], null));

    renderPage();

    expect(await screen.findByText("Linehaul")).toBeInTheDocument();
    expect(screen.getByText("Fuel")).toBeInTheDocument();
    expect(screen.getAllByText("INV-1001").length).toBeGreaterThan(0);
    expect(screen.getByText("4")).toBeInTheDocument();
    expect(mocks.listApprovals).toHaveBeenCalledWith(
      { first: 20, after: null, query: "", kind: null },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(mocks.getDetail).toHaveBeenCalledWith(
      "iadj_1",
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(mocks.rest).not.toHaveBeenCalled();
  });

  it("asks for the next page with the previous page's end cursor", async () => {
    mocks.listApprovals
      .mockResolvedValueOnce(page([queueRow("iadj_1", "INV-1001")], "cursor-1"))
      .mockResolvedValueOnce(page([queueRow("iadj_2", "INV-1002")], null));

    renderPage();
    await screen.findByText("Linehaul");

    act(() => intersect?.([{ isIntersecting: true }]));

    expect(await screen.findByText("INV-1002")).toBeInTheDocument();
    expect(mocks.listApprovals).toHaveBeenCalledTimes(2);
    expect(mocks.listApprovals.mock.calls[1][0]).toEqual({
      first: 20,
      after: "cursor-1",
      query: "",
      kind: null,
    });
    expect(screen.getAllByText("INV-1001").length).toBeGreaterThan(0);
  });

  it("filters by a known kind and drops one the schema does not have", async () => {
    mocks.listApprovals.mockResolvedValue(page([], null));

    renderPage("?kind=WriteOff&query=%20ACME%20");
    await waitFor(() => expect(mocks.listApprovals).toHaveBeenCalled());
    expect(mocks.listApprovals.mock.calls.at(-1)?.[0]).toEqual({
      first: 20,
      after: null,
      query: "ACME",
      kind: "WriteOff",
    });

    cleanup();
    mocks.listApprovals.mockClear();

    renderPage("?kind=Refund");
    await waitFor(() => expect(mocks.listApprovals).toHaveBeenCalled());
    expect(mocks.listApprovals.mock.calls.at(-1)?.[0]).toMatchObject({ kind: null });
  });

  it("approves the open adjustment by id", async () => {
    const user = userEvent.setup();
    mocks.listApprovals.mockResolvedValue(page([queueRow("iadj_1", "INV-1001")], null));
    mocks.approve.mockResolvedValue({
      id: "iadj_1",
      status: "Executed",
      approvalStatus: "Approved",
    });

    renderPage();
    await screen.findByText("Linehaul");
    await user.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() => expect(mocks.approve).toHaveBeenCalled());
    expect(mocks.approve.mock.calls[0][0]).toBe("iadj_1");
  });

  it("rejects with the trimmed reason", async () => {
    const user = userEvent.setup();
    mocks.listApprovals.mockResolvedValue(page([queueRow("iadj_1", "INV-1001")], null));
    mocks.reject.mockResolvedValue({
      id: "iadj_1",
      status: "Rejected",
      approvalStatus: "Rejected",
      rejectionReason: "Duplicate charge",
    });

    renderPage();
    await screen.findByText("Linehaul");
    await user.click(screen.getByRole("button", { name: "Reject" }));
    await user.type(
      screen.getByRole("textbox", { name: "Rejection reason" }),
      "  Duplicate charge  ",
    );
    await user.click(screen.getByRole("button", { name: "Confirm rejection" }));

    await waitFor(() => expect(mocks.reject).toHaveBeenCalled());
    expect(mocks.reject.mock.calls[0][0]).toEqual({
      adjustmentId: "iadj_1",
      reason: "Duplicate charge",
    });
  });

  it("says the queue did not load instead of claiming nothing is waiting", async () => {
    mocks.listApprovals.mockRejectedValue(new Error("network down"));

    renderPage();

    expect(await screen.findByText("Approvals did not load")).toBeInTheDocument();
    expect(screen.queryByText("Nothing waiting")).not.toBeInTheDocument();
  });

  it("says the open adjustment did not load instead of spinning forever", async () => {
    mocks.listApprovals.mockResolvedValue(page([queueRow("iadj_1", "INV-1001")], null));
    mocks.getDetail.mockResolvedValue(null);

    renderPage();

    expect(await screen.findByText("Adjustment did not load")).toBeInTheDocument();
  });
});
