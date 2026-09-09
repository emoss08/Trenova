import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { queries } from "@/lib/queries";
import { useEDITransactionSetOptions } from "@/hooks/use-edi-transaction-set-options";

vi.mock("@/lib/graphql/select-options", () => ({
  fetchGraphQLSelectOptions: vi.fn(),
}));

const fetchMock = vi.mocked(fetchGraphQLSelectOptions);

const fallback = [
  { value: "204", label: "204" },
  { value: "990", label: "990" },
];

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

function catalogResponse() {
  return {
    results: [
      {
        id: "edits_x12_204",
        label: "204 - Motor Carrier Load Tender",
        description: null,
        meta: { code: "204" },
      },
    ],
    count: 1,
    next: null,
    prev: null,
    pageInfo: { mode: "offset" as const, hasNextPage: false, endCursor: null, totalCount: 1 },
  };
}

describe("useEDITransactionSetOptions", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  });

  afterEach(() => {
    queryClient.clear();
    fetchMock.mockReset();
  });

  it("serves the static fallback until the catalog arrives, then the catalog labels", async () => {
    let resolveFetch: (value: ReturnType<typeof catalogResponse>) => void = () => {};
    fetchMock.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveFetch = resolve;
      }),
    );

    const { result } = renderHook(() => useEDITransactionSetOptions(fallback), {
      wrapper: wrapper(queryClient),
    });

    expect(result.current).toEqual(fallback);

    resolveFetch(catalogResponse());

    await waitFor(() =>
      expect(result.current).toEqual([{ value: "204", label: "204 - Motor Carrier Load Tender" }]),
    );
  });

  it("asks the EDI_TRANSACTION_SET resource for active sets only and forwards the abort signal", async () => {
    fetchMock.mockResolvedValueOnce(catalogResponse());

    renderHook(() => useEDITransactionSetOptions(fallback), { wrapper: wrapper(queryClient) });

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    const [params, options] = fetchMock.mock.calls[0];
    expect(params).toMatchObject({
      resource: "EDI_TRANSACTION_SET",
      filters: { status: "Active" },
    });
    expect(params.initialLimit).toBeGreaterThan(6);
    expect(options?.signal).toBeInstanceOf(AbortSignal);
  });

  it("keeps the fallback when the catalog request fails", async () => {
    fetchMock.mockRejectedValueOnce(new Error("offline"));

    const { result } = renderHook(() => useEDITransactionSetOptions(fallback), {
      wrapper: wrapper(queryClient),
    });

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(queryClient.getQueryState(queries.edi.transactionSetOptions().queryKey)?.status).toBe(
        "error",
      ),
    );
    expect(result.current).toEqual(fallback);
  });
});
