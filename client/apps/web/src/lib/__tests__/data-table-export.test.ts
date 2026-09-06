import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { buildCsv, fetchAllRows, type ExportColumn } from "../data-table-export";

const fetchGraphQLDataMock = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/data-table/use-data-table-query", () => ({
  fetchGraphQLData: fetchGraphQLDataMock,
}));

const exportGraphQLConfig = {
  document:
    "query ExportTable($input: DataTableConnectionInput!) { rows(input: $input) { totalCount } }",
  operationName: "ExportTable",
  connectionKey: "rows",
};

type TestRow = {
  id: string;
  name: string;
  amount: number | null;
  tags: string[];
  worker?: { firstName: string };
};

const columns: ExportColumn<TestRow>[] = [
  { id: "id", header: "ID", getValue: (row) => row.id },
  { id: "name", header: "Name", getValue: (row) => row.name },
  { id: "amount", header: "Amount, USD", getValue: (row) => row.amount },
  { id: "tags", header: "Tags", getValue: (row) => row.tags },
  { id: "worker", header: "Worker", getValue: (row) => row.worker?.firstName },
];

describe("buildCsv", () => {
  it("emits a header row followed by data rows", () => {
    const csv = buildCsv<TestRow>(
      [{ id: "1", name: "Alpha", amount: 10, tags: [], worker: { firstName: "Ann" } }],
      columns,
    );
    const lines = csv.split("\r\n");

    expect(lines).toHaveLength(2);
    expect(lines[0]).toBe('ID,Name,"Amount, USD",Tags,Worker');
    expect(lines[1]).toBe("1,Alpha,10,[],Ann");
  });

  it("escapes quotes, commas, and newlines", () => {
    const csv = buildCsv<TestRow>(
      [{ id: "1", name: 'He said "hi",\nthen left', amount: null, tags: [] }],
      columns,
    );
    const lines = csv.split("\r\n");

    expect(lines[1]).toBe('1,"He said ""hi"",\nthen left",,[],');
  });

  it("serializes arrays and objects as JSON", () => {
    const csv = buildCsv<TestRow>([{ id: "1", name: "A", amount: 0, tags: ["x", "y"] }], columns);

    expect(csv.split("\r\n")[1]).toBe('1,A,0,"[""x"",""y""]",');
  });

  it("handles empty row sets", () => {
    const csv = buildCsv<TestRow>([], columns);
    expect(csv).toBe('ID,Name,"Amount, USD",Tags,Worker');
  });
});

describe("fetchAllRows", () => {
  afterEach(() => {
    fetchGraphQLDataMock.mockReset();
  });

  it("reports the first page's total while later pages omit it", async () => {
    fetchGraphQLDataMock
      .mockResolvedValueOnce(
        cursorPage([{ id: "1" }, { id: "2" }], { totalCount: 3, endCursor: "c2" }),
      )
      .mockResolvedValueOnce(cursorPage([{ id: "3" }], { totalCount: null, endCursor: null }));
    const onProgress = vi.fn();

    const rows = await fetchAllRows<{ id: string }>({
      graphql: exportGraphQLConfig,
      options: { query: "", fieldFilters: [], filterGroups: [], sort: [] },
      onProgress,
    });

    expect(rows.map((row) => row.id)).toEqual(["1", "2", "3"]);
    expect(onProgress.mock.calls.map(([progress]) => progress)).toEqual([
      { fetched: 2, total: 3 },
      { fetched: 3, total: 3 },
    ]);
    expect(fetchGraphQLDataMock).toHaveBeenNthCalledWith(
      2,
      expect.any(Number),
      exportGraphQLConfig,
      expect.objectContaining({ cursor: "c2" }),
      { signal: undefined },
    );
  });

  it("forwards the abort signal to every page fetch", async () => {
    fetchGraphQLDataMock
      .mockResolvedValueOnce(cursorPage([{ id: "1" }], { totalCount: 2, endCursor: "c1" }))
      .mockResolvedValueOnce(cursorPage([{ id: "2" }], { totalCount: null, endCursor: null }));
    const controller = new AbortController();

    await fetchAllRows<{ id: string }>({
      graphql: exportGraphQLConfig,
      options: { query: "", fieldFilters: [], filterGroups: [], sort: [] },
      signal: controller.signal,
    });

    expect(fetchGraphQLDataMock).toHaveBeenCalledTimes(2);
    for (const call of fetchGraphQLDataMock.mock.calls) {
      expect(call[3]).toEqual({ signal: controller.signal });
    }
  });

  it("stops paging once the signal is aborted", async () => {
    const controller = new AbortController();
    fetchGraphQLDataMock.mockImplementationOnce(async () => {
      controller.abort();
      return cursorPage([{ id: "1" }], { totalCount: 5, endCursor: "c1" });
    });

    const rows = await fetchAllRows<{ id: string }>({
      graphql: exportGraphQLConfig,
      options: { query: "", fieldFilters: [], filterGroups: [], sort: [] },
      signal: controller.signal,
    });

    expect(rows.map((row) => row.id)).toEqual(["1"]);
    expect(fetchGraphQLDataMock).toHaveBeenCalledTimes(1);
  });

  it("reports no total when the server never returns one", async () => {
    fetchGraphQLDataMock.mockResolvedValueOnce(
      cursorPage([{ id: "1" }], { totalCount: null, endCursor: null }),
    );
    const onProgress = vi.fn();

    await fetchAllRows<{ id: string }>({
      graphql: exportGraphQLConfig,
      options: { query: "", fieldFilters: [], filterGroups: [], sort: [] },
      onProgress,
    });

    expect(onProgress).toHaveBeenCalledWith({ fetched: 1, total: null });
  });
});

function cursorPage<TData extends Record<string, unknown>>(
  results: TData[],
  page: { totalCount: number | null; endCursor: string | null },
): GenericLimitOffsetResponse<TData> {
  return {
    results,
    count: page.totalCount ?? results.length,
    next: null,
    prev: null,
    pageInfo: {
      mode: "cursor",
      hasNextPage: page.endCursor != null,
      endCursor: page.endCursor,
      totalCount: page.totalCount,
    },
  };
}
