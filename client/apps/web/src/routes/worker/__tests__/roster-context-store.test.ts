import { workerTableGraphQLConfig, type WorkerRow } from "@/lib/graphql/worker-table";
import { buildDataTableQueryKey } from "@/hooks/data-table/use-data-table-query";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { DataTableQueryOptions } from "@trenova/shared/types/data-table";
import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  captureRosterContext,
  rosterLabel,
  rosterNeighbors,
  rosterPageKey,
  stepRoster,
  useRosterContextStore,
  type RosterContext,
  type RosterPage,
} from "../roster-context-store";

const { requestGraphQL } = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL,
}));

const graphql = workerTableGraphQLConfig(null);
const PAGE_SIZE = 2;
const FILTERED: Omit<DataTableQueryOptions, "cursor"> = {
  query: undefined,
  fieldFilters: [{ field: "profile.complianceStatus", operator: "eq", value: "NonCompliant" }],
  filterGroups: [],
  sort: [{ field: "lastName", direction: "asc" }],
};

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function row(id: string): WorkerRow {
  return { id } as WorkerRow;
}

function page(
  ids: string[],
  { hasNextPage, endCursor, totalCount }: Partial<NonNullable<RosterPage["pageInfo"]>> = {},
): RosterPage {
  return {
    results: ids.map(row),
    count: totalCount ?? ids.length,
    next: null,
    prev: null,
    pageInfo: {
      mode: "cursor",
      hasNextPage: hasNextPage ?? false,
      endCursor: endCursor ?? null,
      totalCount: totalCount ?? null,
      totalCountExact: true,
    },
  };
}

function tableKey(
  pageIndex: number,
  cursor: string | null,
  options: Omit<DataTableQueryOptions, "cursor"> = FILTERED,
  config = graphql,
) {
  return buildDataTableQueryKey(
    "worker-list",
    config,
    { pageIndex, pageSize: PAGE_SIZE },
    { ...options, cursor },
  );
}

let client: QueryClient;
const unsubscribers: Array<() => void> = [];

function seed(key: ReturnType<typeof tableKey>, data: RosterPage, { observed = false } = {}) {
  client.setQueryData(key, data);
  if (observed) {
    const observer = new QueryObserver(client, { queryKey: key, enabled: false });
    unsubscribers.push(observer.subscribe(() => undefined));
  }
}

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  requestGraphQL.mockReset();
});

afterEach(() => {
  for (const unsubscribe of unsubscribers.splice(0)) unsubscribe();
  client.clear();
  useRosterContextStore.getState().clear();
});

function capture(workerId: string) {
  return captureRosterContext(client, {
    workerId,
    graphql,
    search: "?fieldFilters=x",
    label: () => "Non-compliant",
  });
}

describe("captureRosterContext", () => {
  it("takes the page the person clicked on, with every cursor of its scope", () => {
    seed(
      tableKey(0, null),
      page(["wrk_a", "wrk_b"], { hasNextPage: true, endCursor: "c1", totalCount: 5 }),
    );
    seed(tableKey(1, "c1"), page(["wrk_c", "wrk_d"], { hasNextPage: true, endCursor: "c2" }), {
      observed: true,
    });

    expect(capture("wrk_d")).toEqual({
      scope: { graphql, pageSize: PAGE_SIZE, options: FILTERED },
      cursors: { 0: null, 1: "c1" },
      pageIndex: 1,
      label: "Non-compliant",
      search: "?fieldFilters=x",
    });
  });

  it("prefers the page on screen when a row sits in an older page of another filter too", () => {
    const unfiltered = { ...FILTERED, fieldFilters: [] };
    seed(tableKey(0, null, unfiltered), page(["wrk_a", "wrk_d"]));
    seed(tableKey(1, "c1"), page(["wrk_c", "wrk_d"]), { observed: true });

    expect(capture("wrk_d")?.scope.options).toEqual(FILTERED);
    expect(capture("wrk_d")?.pageIndex).toBe(1);
  });

  it("leaves cursors of another scope out", () => {
    seed(tableKey(1, "other", { ...FILTERED, query: "ann" }), page(["wrk_x"]));
    seed(tableKey(0, null), page(["wrk_a"]), { observed: true });

    expect(capture("wrk_a")?.cursors).toEqual({ 0: null });
  });

  it("skips the table's pinned and linked reads and finds nothing for a row it never listed", () => {
    client.setQueryData(["worker-list", "linked", "wrk_z"], page(["wrk_z"]));

    expect(capture("wrk_z")).toBeNull();
  });
});

describe("rosterNeighbors", () => {
  function context(pageIndex: number, cursors: RosterContext["cursors"]): RosterContext {
    return {
      scope: { graphql, pageSize: PAGE_SIZE, options: FILTERED },
      cursors,
      pageIndex,
      label: "Non-compliant",
      search: "",
    };
  }

  it("counts the position across pages and names the neighbours on the page", () => {
    const neighbors = rosterNeighbors(
      context(1, { 0: null, 1: "c1" }),
      page(["wrk_c", "wrk_d"], { hasNextPage: true, endCursor: "c2" }),
      "wrk_c",
      5,
    );

    expect(neighbors).toEqual({
      index: 0,
      position: 3,
      total: 5,
      previous: null,
      next: "wrk_d",
      canPrevious: true,
      canNext: true,
    });
  });

  it("stops at the first and last worker of the list", () => {
    expect(
      rosterNeighbors(context(0, { 0: null }), page(["wrk_a", "wrk_b"]), "wrk_a", 2),
    ).toMatchObject({ canPrevious: false, canNext: true });
    expect(
      rosterNeighbors(context(0, { 0: null }), page(["wrk_a", "wrk_b"]), "wrk_b", 2),
    ).toMatchObject({ canPrevious: true, canNext: false });
  });

  it("has nothing to step through when the worker has left the page", () => {
    expect(rosterNeighbors(context(0, { 0: null }), page(["wrk_a"]), "wrk_q", null)).toBeNull();
    expect(rosterNeighbors(context(0, { 0: null }), undefined, "wrk_a", null)).toBeNull();
  });
});

describe("stepRoster", () => {
  it("steps inside the page without a read", async () => {
    seed(tableKey(0, null), page(["wrk_a", "wrk_b"]), { observed: true });
    const context = capture("wrk_a");
    if (!context) throw new Error("no context");

    await expect(stepRoster(client, context, "wrk_a", "next")).resolves.toEqual({
      workerId: "wrk_b",
      context,
    });
    expect(requestGraphQL).not.toHaveBeenCalled();
  });

  it("reads the next page with the table's own key and cursor across the page edge", async () => {
    seed(tableKey(0, null), page(["wrk_a", "wrk_b"], { hasNextPage: true, endCursor: "c1" }), {
      observed: true,
    });
    requestGraphQL.mockResolvedValue({
      workers: {
        edges: [{ node: row("wrk_c") }, { node: row("wrk_d") }],
        pageInfo: { hasNextPage: false, endCursor: null },
        totalCount: null,
      },
    });
    const context = capture("wrk_b");
    if (!context) throw new Error("no context");

    const stepped = await stepRoster(client, context, "wrk_b", "next");

    expect(stepped?.workerId).toBe("wrk_c");
    expect(stepped?.context.pageIndex).toBe(1);
    expect(stepped?.context.cursors).toEqual({ 0: null, 1: "c1" });
    expect(requestGraphQL).toHaveBeenCalledTimes(1);
    expect(requestGraphQL.mock.calls[0]?.[0]).toMatchObject({
      operationName: "WorkerTable",
      variables: { includeTotalCount: false, input: { first: PAGE_SIZE, after: "c1" } },
    });
    expect(client.getQueryData(tableKey(1, "c1"))).toMatchObject({
      results: [{ id: "wrk_c" }, { id: "wrk_d" }],
    });
    expect(rosterPageKey(stepped!.context, 1)).toEqual(tableKey(1, "c1"));
  });

  it("steps back onto the last worker of the page before, read from the cache", async () => {
    seed(tableKey(0, null), page(["wrk_a", "wrk_b"], { hasNextPage: true, endCursor: "c1" }));
    seed(tableKey(1, "c1"), page(["wrk_c", "wrk_d"]), { observed: true });
    const context = capture("wrk_c");
    if (!context) throw new Error("no context");

    const stepped = await stepRoster(client, context, "wrk_c", "previous");

    expect(stepped).toEqual({ workerId: "wrk_b", context: { ...context, pageIndex: 0 } });
    expect(requestGraphQL).not.toHaveBeenCalled();
  });

  it("goes nowhere past the end of the list", async () => {
    seed(tableKey(0, null), page(["wrk_a", "wrk_b"]), { observed: true });
    const context = capture("wrk_b");
    if (!context) throw new Error("no context");

    await expect(stepRoster(client, context, "wrk_b", "next")).resolves.toBeNull();
  });
});

describe("rosterLabel", () => {
  const now = 1_800_000_000;

  it("names the missing credential first", () => {
    expect(
      rosterLabel(t, { options: FILTERED, missingCredential: { name: "Medical card" }, now }),
    ).toBe("Workers missing Medical card");
    expect(rosterLabel(t, { options: {}, missingCredential: { name: null }, now })).toBe(
      "Workers missing a credential",
    );
  });

  it("names a search", () => {
    expect(rosterLabel(t, { options: { query: " ann " }, missingCredential: null, now })).toBe(
      "Workers matching “ann”",
    );
  });

  it("names a roster view by its own label", () => {
    expect(rosterLabel(t, { options: FILTERED, missingCredential: null, now })).toBe(
      "Non-compliant",
    );
  });

  it("calls any other filter filtered, and no filter all workers", () => {
    expect(
      rosterLabel(t, {
        options: { fieldFilters: [{ field: "city", operator: "eq", value: "Joliet" }] },
        missingCredential: null,
        now,
      }),
    ).toBe("Filtered workers");
    expect(rosterLabel(t, { options: {}, missingCredential: null, now })).toBe("All workers");
  });
});
