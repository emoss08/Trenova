import { billingQueue } from "@/lib/queries/billing-queue";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useInvoiceDetails } from "../use-invoice-details";

const mocks = vi.hoisted(() => ({ updateInvoiceDetails: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { updateInvoiceDetails: mocks.updateInvoiceDetails } },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: Object.assign(vi.fn(), { error: vi.fn(), success: vi.fn() }),
}));

afterEach(() => vi.clearAllMocks());

function queueItem(overrides: Partial<BillingQueueItem> = {}): BillingQueueItem {
  return {
    id: "bqi_1",
    version: 4,
    status: "InReview",
    billType: "Invoice",
    poNumber: "PO-1",
    invoiceMemo: "",
    paymentTerm: "Net30",
    invoiceDate: null,
    packetDocumentIds: [],
    ...overrides,
  } as BillingQueueItem;
}

function setup(item: BillingQueueItem) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(billingQueue.get(item.id).queryKey, item);
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(() => useInvoiceDetails(item), { wrapper });
  const cached = () => client.getQueryData<BillingQueueItem>(billingQueue.get(item.id).queryKey);
  return { result, cached };
}

function deferred<T>() {
  let resolve: (value: T) => void = () => {};
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

describe("useInvoiceDetails", () => {
  it("shows a change on the item before the save comes back", () => {
    mocks.updateInvoiceDetails.mockReturnValue(new Promise(() => undefined));
    const { result, cached } = setup(queueItem());

    act(() => {
      void result.current.save({ packetDocumentIds: ["doc_pod"] });
    });

    expect(cached()?.packetDocumentIds).toEqual(["doc_pod"]);
  });

  // The second click used to go out with the version the first was about to bump,
  // and the server refused it as a conflict.
  it("saves one change at a time, each with the version the last save returned", async () => {
    const first = deferred<BillingQueueItem>();
    mocks.updateInvoiceDetails
      .mockReturnValueOnce(first.promise)
      .mockResolvedValueOnce(queueItem({ version: 6, packetDocumentIds: ["doc_pod", "doc_bol"] }));
    const { result, cached } = setup(queueItem());

    let firstSave: Promise<unknown> = Promise.resolve();
    let secondSave: Promise<unknown> = Promise.resolve();
    act(() => {
      firstSave = result.current.save({ packetDocumentIds: ["doc_pod"] });
      secondSave = result.current.save({ packetDocumentIds: ["doc_pod", "doc_bol"] });
    });

    expect(cached()?.packetDocumentIds).toEqual(["doc_pod", "doc_bol"]);
    await waitFor(() => expect(mocks.updateInvoiceDetails).toHaveBeenCalledTimes(1));
    expect(mocks.updateInvoiceDetails).toHaveBeenLastCalledWith(
      "bqi_1",
      expect.objectContaining({ version: 4, packetDocumentIds: ["doc_pod", "doc_bol"] }),
    );

    await act(async () => {
      first.resolve(queueItem({ version: 5, packetDocumentIds: ["doc_pod", "doc_bol"] }));
      await firstSave;
    });
    await act(async () => {
      await secondSave;
    });

    expect(mocks.updateInvoiceDetails).toHaveBeenCalledTimes(2);
    expect(mocks.updateInvoiceDetails).toHaveBeenLastCalledWith(
      "bqi_1",
      expect.objectContaining({ version: 5 }),
    );
    expect(cached()).toMatchObject({ version: 6, packetDocumentIds: ["doc_pod", "doc_bol"] });
  });

  it("keeps a later change on screen while an earlier save comes back", async () => {
    const first = deferred<BillingQueueItem>();
    const second = deferred<BillingQueueItem>();
    mocks.updateInvoiceDetails
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    const { result, cached } = setup(queueItem());

    let firstSave: Promise<unknown> = Promise.resolve();
    act(() => {
      firstSave = result.current.save({ packetDocumentIds: ["doc_pod"] });
    });
    await waitFor(() => expect(mocks.updateInvoiceDetails).toHaveBeenCalledTimes(1));
    act(() => {
      void result.current.save({ packetDocumentIds: ["doc_pod", "doc_bol"] });
    });

    await act(async () => {
      first.resolve(queueItem({ version: 5, packetDocumentIds: ["doc_pod"] }));
      await firstSave;
    });

    expect(cached()).toMatchObject({ version: 5, packetDocumentIds: ["doc_pod", "doc_bol"] });
  });
});
