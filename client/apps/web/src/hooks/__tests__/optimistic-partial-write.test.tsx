import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";

const PROBLEM_BASE = "https://api.trenova.app/problems/";

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

const JOURNAL_KEY = ["manualJournal", "mj_1"] as const;
const COMMITTED = { id: "mj_1", reason: "Original reason" };
const OPTIMISTIC = { id: "mj_1", reason: "Edited reason" };

// The response shape characterised on the real manual-journal path: two serially-executed
// mutations, the first succeeding into `data`, the second rejecting into `errors`.
function partiallyFailedWrite(): Response {
  return new Response(
    JSON.stringify({
      data: { updateManualJournalDraft: { id: "mj_1", reason: "Edited reason" } },
      errors: [
        {
          message: "Only draft journals can be submitted",
          path: ["submitManualJournal"],
          extensions: { code: "INVALID", type: `${PROBLEM_BASE}validation-error` },
        },
      ],
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );
}

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe("a partially-failed write must not commit its optimistic update", () => {
  let queryClient: QueryClient;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("graphql-token");
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(JOURNAL_KEY, COMMITTED);
  });

  afterEach(() => {
    clearCsrfToken();
    vi.unstubAllGlobals();
    queryClient.clear();
  });

  function renderJournalMutation() {
    const onSuccess = vi.fn();
    const onError = vi.fn();

    const { result } = renderHook(
      () =>
        useOptimisticMutation<{ updateManualJournalDraft: { id: string } }, { reason: string }, typeof COMMITTED>(
          {
            queryKey: [...JOURNAL_KEY],
            resourceName: "manual journal",
            mutationFn: (data) =>
              requestGraphQL({
                document:
                  "mutation UpdateManualJournalDraft($reason: String!) { updateManualJournalDraft(reason: $reason) { id } submitManualJournal { id } }",
                operationName: "UpdateManualJournalDraft",
                variables: data,
              }),
            optimisticUpdate: () => OPTIMISTIC,
            onSuccess,
            onError,
          },
        ),
      { wrapper: wrapper(queryClient) },
    );

    return { result, onSuccess, onError };
  }

  it("rolls the cache back and reports failure", async () => {
    fetchMock.mockResolvedValue(partiallyFailedWrite());
    const { result, onSuccess, onError } = renderJournalMutation();

    await act(async () => {
      result.current.mutate({ reason: "Edited reason" });
    });
    await waitFor(() => expect(result.current.isPending).toBe(false));

    expect(result.current.isError).toBe(true);
    expect(onSuccess).not.toHaveBeenCalled();
    expect(onError).toHaveBeenCalledTimes(1);
    expect(queryClient.getQueryData(JOURNAL_KEY)).toEqual(COMMITTED);
  });

  it("commits normally when the write fully succeeds", async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({ data: { updateManualJournalDraft: { id: "mj_1" } } }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    const { result, onSuccess, onError } = renderJournalMutation();

    await act(async () => {
      result.current.mutate({ reason: "Edited reason" });
    });
    await waitFor(() => expect(result.current.isPending).toBe(false));

    expect(result.current.isError).toBe(false);
    expect(onSuccess).toHaveBeenCalledTimes(1);
    expect(onError).not.toHaveBeenCalled();
    // onSettled invalidates, so the optimistic value is what remains in cache here.
    expect(queryClient.getQueryData(JOURNAL_KEY)).toEqual(OPTIMISTIC);
  });
});
