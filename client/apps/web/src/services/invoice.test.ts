import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceService } from "./invoice";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe("InvoiceService.listEmailAttempts", () => {
  it("reads a page of the invoice's emails and says whether more remain", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      invoiceEmailAttempts: { items: [], total: 40, totalExact: true, hasMore: true },
    });

    const page = await new InvoiceService().listEmailAttempts("inv_1", { limit: 20, offset: 20 });

    expect(vi.mocked(requestGraphQL).mock.calls[0][0]).toMatchObject({
      operationName: "InvoiceEmailAttempts",
      variables: { invoiceId: "inv_1", limit: 20, offset: 20 },
    });
    expect(page.hasMore).toBe(true);
    expect(page.total).toBe(40);
  });
});

describe("InvoiceService.generatePdf", () => {
  it("queues the PDF and parses the workflow it started", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      generateInvoicePdf: {
        invoiceId: "inv_1",
        workflowId: "wf_1",
        workflowRunId: "run_1",
        status: "Queued",
      },
    });

    const result = await new InvoiceService().generatePdf("inv_1");

    expect(vi.mocked(requestGraphQL).mock.calls[0][0]).toMatchObject({
      operationName: "GenerateInvoicePdf",
      variables: { invoiceId: "inv_1" },
    });
    expect(result.workflowId).toBe("wf_1");
  });
});
