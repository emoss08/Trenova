import { useReportDefinitionList } from "@/hooks/use-reports";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

/**
 * The library grid used to ask for a fixed first hundred reports with no way to
 * reach the hundred and first, and nothing on screen said the rest existed. It
 * pages now, so what the request carries is the contract worth pinning.
 */

function tablePage(names: string[], endCursor: string | null): Response {
  return new Response(
    JSON.stringify({
      data: {
        reportDefinitions: {
          edges: names.map((name, index) => ({
            cursor: `cursor-${name}`,
            node: {
              id: `rdef_${index}_${name}`,
              name,
              description: "",
              category: "Operations",
              tags: [],
              kind: "custom",
              cannedKey: null,
              cannedVersion: null,
              ownerId: "usr_1",
              visibility: "private",
              status: "active",
              diagnostics: [],
              catalogVersion: "1",
              definition: { entity: "shipment", columns: [] },
              defaultFormat: "csv",
              currentRevision: 1,
              lastRunAt: null,
              version: 1,
              createdAt: 1_700_000_000,
              updatedAt: 1_700_000_000,
            },
          })),
          pageInfo: { hasNextPage: endCursor != null, endCursor },
        },
      },
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );
}

let fetchMock: ReturnType<typeof vi.fn>;

function requestBody(call = 0): {
  variables: { input: Record<string, unknown>; includeTotalCount?: boolean };
} {
  return JSON.parse(String(fetchMock.mock.calls[call]?.[1]?.body));
}

let lastClient: QueryClient;

function wrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  lastClient = client;
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe("useReportDefinitionList", () => {
  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("report-list-token");
    fetchMock = vi.fn(async () => tablePage(["Aging"], null));
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("asks for a page rather than a fixed hundred rows", async () => {
    const { result } = renderHook(() => useReportDefinitionList(""), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const input = requestBody().variables.input;
    expect(input.first).not.toBe(100);
    expect(input).toHaveProperty("after", null);
  });

  it("reports that more of the library exists", async () => {
    fetchMock.mockImplementationOnce(async () => tablePage(["Aging"], "cursor-page-1"));

    const { result } = renderHook(() => useReportDefinitionList(""), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.hasNextPage).toBe(true);
  });

  it("carries the previous page's cursor into the next request", async () => {
    fetchMock
      .mockImplementationOnce(async () => tablePage(["Aging", "Backhaul"], "cursor-page-1"))
      .mockImplementationOnce(async () => tablePage(["Chargebacks"], null));

    const { result } = renderHook(() => useReportDefinitionList(""), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    await act(async () => {
      await result.current.fetchNextPage();
    });

    expect(requestBody(1).variables.input).toMatchObject({ after: "cursor-page-1" });
    await waitFor(() =>
      expect(lastClient.getQueryData(["reports", "definitionList", ""])).toMatchObject({
        pageParams: [null, "cursor-page-1"],
      }),
    );
  });

  it("hands the search term to the server", async () => {
    const { result } = renderHook(() => useReportDefinitionList("margin"), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(requestBody().variables.input).toMatchObject({ query: "margin" });
  });

  // The COUNT behind totalCount is gated on the field being selected, and the
  // grid shows per-category counts of what it holds, never a library total.
  it("does not make every page pay for a count nothing shows", async () => {
    const { result } = renderHook(() => useReportDefinitionList(""), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(requestBody().variables.includeTotalCount).toBe(false);
  });
});
