import { useReportDefinitionOptions } from "@/hooks/use-reports";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

/**
 * The report library is a cursor connection, and a picker over it has exactly
 * two obligations the previous one did not meet: it must hand the search term
 * to the server rather than filtering a fixed prefix of the library on the
 * client, and it must be able to reach past the first page. Both are asserted
 * against the request the transport actually sends.
 */

type Page = {
  names: string[];
  hasNextPage: boolean;
  endCursor: string | null;
};

function connectionResponse({ names, hasNextPage, endCursor }: Page): Response {
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
              kind: "custom",
              status: "published",
              visibility: "private",
              lastRunAt: null,
              updatedAt: 1_700_000_000,
            },
          })),
          pageInfo: { hasNextPage, endCursor },
        },
      },
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );
}

let fetchMock: ReturnType<typeof vi.fn>;

function requestBody(call = 0): {
  operationName: string;
  variables: { input: Record<string, unknown> };
} {
  return JSON.parse(String(fetchMock.mock.calls[call]?.[1]?.body));
}

function wrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe("useReportDefinitionOptions", () => {
  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("report-options-token");
    fetchMock = vi.fn(async () =>
      connectionResponse({ names: ["Aging"], hasNextPage: false, endCursor: null }),
    );
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("sends the search term to the server rather than filtering on the client", async () => {
    const { result } = renderHook(() => useReportDefinitionOptions("margin"), {
      wrapper: wrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const body = requestBody();
    expect(body.operationName).toBe("ReportDefinitionOptions");
    expect(body.variables.input).toMatchObject({ query: "margin" });
  });

  it("omits the query entirely when nothing is typed", async () => {
    const { result } = renderHook(() => useReportDefinitionOptions(""), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(requestBody().variables.input.query).toBeUndefined();
  });

  it("asks for the first page without a cursor", async () => {
    const { result } = renderHook(() => useReportDefinitionOptions(""), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(requestBody().variables.input.after).toBeNull();
  });

  // The COUNT behind totalCount is gated on the field being selected, and a
  // picker never shows a total — asking for one would make every keystroke pay
  // for a count nobody reads.
  it("never asks for a total count", async () => {
    const { result } = renderHook(() => useReportDefinitionOptions(""), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(requestBody().variables.input).not.toHaveProperty("includeTotalCount");
    expect(JSON.stringify(requestBody())).not.toContain("totalCount");
  });

  it("carries the previous page's cursor into the next request and appends the rows", async () => {
    fetchMock
      .mockImplementationOnce(async () =>
        connectionResponse({
          names: ["Aging", "Backhaul"],
          hasNextPage: true,
          endCursor: "cursor-page-1",
        }),
      )
      .mockImplementationOnce(async () =>
        connectionResponse({ names: ["Chargebacks"], hasNextPage: false, endCursor: null }),
      );

    const { result } = renderHook(() => useReportDefinitionOptions(""), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.hasNextPage).toBe(true);
    expect(result.current.data?.map((entry) => entry.name)).toEqual(["Aging", "Backhaul"]);

    await act(async () => {
      await result.current.fetchNextPage();
    });

    expect(requestBody(1).variables.input).toMatchObject({ after: "cursor-page-1" });
    await waitFor(() =>
      expect(result.current.data?.map((entry) => entry.name)).toEqual([
        "Aging",
        "Backhaul",
        "Chargebacks",
      ]),
    );
    expect(result.current.hasNextPage).toBe(false);
  });

  // A last page that reports hasNextPage with no cursor would otherwise loop
  // forever on the same request.
  it("stops paging when the server reports another page but no cursor", async () => {
    fetchMock.mockImplementationOnce(async () =>
      connectionResponse({ names: ["Aging"], hasNextPage: true, endCursor: null }),
    );

    const { result } = renderHook(() => useReportDefinitionOptions(""), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.hasNextPage).toBe(false);
  });

  it("stays idle until the picker asks for it", async () => {
    renderHook(() => useReportDefinitionOptions("", false), { wrapper: wrapper() });

    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
