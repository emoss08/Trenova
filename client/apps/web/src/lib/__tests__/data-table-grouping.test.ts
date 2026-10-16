import { describe, expect, it } from "vitest";
import type {
  DataTableGroup,
  DataTableGroupScope,
  SortField,
} from "@trenova/shared/types/data-table";
import {
  buildGroupedBody,
  groupedScope,
  groupedSort,
  groupScopedQuery,
  toggleGroupKey,
} from "../data-table-grouping";

type TestRow = { id: string; stage: number };

const GROUPS: DataTableGroup[] = [
  { key: 1, label: "Late" },
  { key: 2, label: "Needs coverage" },
  { key: 3, label: "Moving" },
  { key: 4, label: "Scheduled" },
  { key: 5, label: "Delivered" },
];

const row = (id: string, stage: number): TestRow => ({ id, stage });
const getKey = (r: TestRow) => r.stage;

function shape(items: ReturnType<typeof buildGroupedBody<TestRow>>) {
  return items.map((item) => {
    switch (item.kind) {
      case "group":
        return `group:${item.group.key}${item.collapsed ? ":collapsed" : ""}`;
      case "row":
        return `row:${item.row.id}@${item.index}`;
      case "pending":
        return `pending:${item.key}x${item.count}`;
    }
  });
}

describe("groupedSort", () => {
  it("puts the group field ahead of the person's own sort", () => {
    const sort: SortField[] = [{ field: "createdAt", direction: "desc" }];
    expect(groupedSort({ field: "stageRank" }, sort)).toEqual([
      { field: "stageRank", direction: "asc" },
      { field: "createdAt", direction: "desc" },
    ]);
  });

  it("keeps the group field first when the person also sorted by it, honouring their direction", () => {
    const sort: SortField[] = [
      { field: "proNumber", direction: "asc" },
      { field: "stageRank", direction: "desc" },
    ];
    expect(groupedSort({ field: "stageRank" }, sort)).toEqual([
      { field: "stageRank", direction: "desc" },
      { field: "proNumber", direction: "asc" },
    ]);
  });

  it("follows the group field with its tie-breakers, ahead of the person's sort", () => {
    const sort: SortField[] = [
      { field: "customerId", direction: "desc" },
      { field: "createdAt", direction: "desc" },
    ];
    expect(
      groupedSort(
        { field: "customer.name", tieBreakers: [{ field: "customerId", direction: "asc" }] },
        sort,
      ),
    ).toEqual([
      { field: "customer.name", direction: "asc" },
      { field: "customerId", direction: "asc" },
      { field: "createdAt", direction: "desc" },
    ]);
  });

  it("leaves the sort alone when grouping is off", () => {
    const sort: SortField[] = [{ field: "proNumber", direction: "asc" }];
    expect(groupedSort(undefined, sort)).toBe(sort);
  });
});

describe("groupedScope", () => {
  it("excludes collapsed groups server-side", () => {
    expect(
      groupedScope({ field: "stageRank", collapsedKeys: [2, 5] }),
    ).toEqual<DataTableGroupScope>({
      fieldFilters: [{ field: "stageRank", operator: "notin", value: [2, 5] }],
      filterGroups: [],
    });
  });

  it("lets a grouping whose keys are not field values say how to drop them", () => {
    const scope: DataTableGroupScope = {
      fieldFilters: [],
      filterGroups: [
        {
          filters: [
            { field: "shipperStop.scheduledWindowStart", operator: "lt", value: 100 },
            { field: "shipperStop.scheduledWindowStart", operator: "gte", value: 200 },
          ],
        },
      ],
    };
    const seen: unknown[] = [];
    expect(
      groupedScope({
        field: "shipperStop.scheduledWindowStart",
        collapsedKeys: ["2026-10-07"],
        collapsedScope: (keys) => {
          seen.push(keys);
          return scope;
        },
      }),
    ).toBe(scope);
    expect(seen).toEqual([["2026-10-07"]]);
  });

  it("adds nothing when no group is collapsed or grouping is off", () => {
    const empty: DataTableGroupScope = { fieldFilters: [], filterGroups: [] };
    expect(groupedScope({ field: "stageRank", collapsedKeys: [] })).toEqual(empty);
    expect(
      groupedScope({
        field: "stageRank",
        collapsedKeys: [],
        collapsedScope: () => ({
          fieldFilters: [{ field: "x", operator: "eq", value: 1 }],
          filterGroups: [],
        }),
      }),
    ).toEqual(empty);
    expect(groupedScope(undefined)).toEqual(empty);
  });
});

describe("buildGroupedBody", () => {
  it("opens each group once with a header, keeping row indexes contiguous", () => {
    const items = buildGroupedBody({
      rows: [row("a", 1), row("b", 1), row("c", 3)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [],
      isFirstPage: true,
      isLastPage: true,
    });
    expect(shape(items)).toEqual(["group:1", "row:a@0", "row:b@1", "group:3", "row:c@2"]);
  });

  it("repeats the header of a group that continues from the previous page", () => {
    const items = buildGroupedBody({
      rows: [row("d", 3), row("e", 4)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [],
      isFirstPage: false,
      isLastPage: false,
    });
    expect(shape(items)).toEqual(["group:3", "row:d@0", "group:4", "row:e@1"]);
  });

  it("places a collapsed group between the visible groups it sorts between", () => {
    const items = buildGroupedBody({
      rows: [row("a", 1), row("c", 3)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [2],
      isFirstPage: false,
      isLastPage: false,
    });
    expect(shape(items)).toEqual(["group:1", "row:a@0", "group:2:collapsed", "group:3", "row:c@1"]);
  });

  it("shows collapsed groups that sort before every row only on the first page", () => {
    const base = {
      rows: [row("c", 3)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [1, 2],
      isLastPage: false,
    };
    expect(shape(buildGroupedBody({ ...base, isFirstPage: true }))).toEqual([
      "group:1:collapsed",
      "group:2:collapsed",
      "group:3",
      "row:c@0",
    ]);
    expect(shape(buildGroupedBody({ ...base, isFirstPage: false }))).toEqual([
      "group:3",
      "row:c@0",
    ]);
  });

  it("shows collapsed groups that sort after every row only on the last page", () => {
    const base = {
      rows: [row("a", 1)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [4, 5],
      isFirstPage: false,
    };
    expect(shape(buildGroupedBody({ ...base, isLastPage: true }))).toEqual([
      "group:1",
      "row:a@0",
      "group:4:collapsed",
      "group:5:collapsed",
    ]);
    expect(shape(buildGroupedBody({ ...base, isLastPage: false }))).toEqual(["group:1", "row:a@0"]);
  });

  it("lists every collapsed group when they hide all the rows", () => {
    const items = buildGroupedBody({
      rows: [],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [5, 1],
      isFirstPage: true,
      isLastPage: true,
    });
    expect(shape(items)).toEqual(["group:1:collapsed", "group:5:collapsed"]);
  });

  it("puts a row whose key names no group under a header of its own, after the known groups", () => {
    const items = buildGroupedBody({
      rows: [row("a", 1), row("z", 9)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [],
      isFirstPage: true,
      isLastPage: true,
    });
    expect(shape(items)).toEqual(["group:1", "row:a@0", "group:9", "row:z@1"]);
  });
});

describe("buildGroupedBody while the next page loads", () => {
  it("folds a group the moment it collapses, even with its rows still on screen", () => {
    const items = buildGroupedBody({
      rows: [row("a", 1), row("b", 1), row("c", 3)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [1],
      isFirstPage: true,
      isLastPage: true,
    });
    expect(shape(items)).toEqual(["group:1:collapsed", "group:3", "row:c@2"]);
  });

  it("keeps the header of a group that just opened while its rows load", () => {
    const items = buildGroupedBody({
      rows: [row("c", 3)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [],
      loadingKeys: [2],
      isFirstPage: true,
      isLastPage: true,
    });
    expect(shape(items)).toEqual(["group:2", "pending:group:2x25", "group:3", "row:c@0"]);
  });

  it("holds open as many rows as a just-opened group has under its header", () => {
    const items = buildGroupedBody({
      rows: [row("a", 1), row("c", 3)],
      groups: GROUPS.map((group) => (group.key === 2 ? { ...group, count: 3 } : group)),
      getGroupKey: getKey,
      pageSize: 25,
      collapsedKeys: [],
      loadingKeys: [2],
      isFirstPage: true,
      isLastPage: false,
    });
    expect(shape(items)).toEqual([
      "group:1",
      "row:a@0",
      "group:2",
      "pending:group:2x3",
      "group:3",
      "row:c@1",
    ]);
  });

  it("holds open no more rows for a just-opened group than the page has room for", () => {
    const items = buildGroupedBody({
      rows: [row("a", 1), row("b", 1), row("c", 3)],
      groups: GROUPS.map((group) => (group.key === 2 ? { ...group, count: 40 } : group)),
      getGroupKey: getKey,
      pageSize: 3,
      collapsedKeys: [],
      loadingKeys: [2],
      isFirstPage: true,
      isLastPage: false,
    });
    expect(shape(items)).toEqual([
      "group:1",
      "row:a@0",
      "row:b@1",
      "group:2",
      "pending:group:2x1",
      "group:3",
      "row:c@2",
    ]);
  });

  it("holds the foot of the page open for the rows that move up after a group collapses", () => {
    const items = buildGroupedBody({
      rows: [row("a", 1), row("b", 1), row("c", 3)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 3,
      collapsedKeys: [1],
      isFirstPage: true,
      isLastPage: false,
    });
    expect(shape(items)).toEqual(["group:1:collapsed", "group:3", "row:c@2", "pending:backfillx2"]);
  });

  it("holds nothing open after a collapse on the last page, where no rows are left to move up", () => {
    const items = buildGroupedBody({
      rows: [row("a", 1), row("b", 1), row("c", 3)],
      groups: GROUPS,
      getGroupKey: getKey,
      pageSize: 3,
      collapsedKeys: [1],
      isFirstPage: true,
      isLastPage: true,
    });
    expect(shape(items)).toEqual(["group:1:collapsed", "group:3", "row:c@2"]);
  });
});

describe("toggleGroupKey", () => {
  it("adds a group that was open to the end of the collapsed keys", () => {
    expect(toggleGroupKey([3, 1], 2)).toEqual([3, 1, 2]);
  });

  it("removes a group that was collapsed, keeping the others in order", () => {
    expect(toggleGroupKey([3, 2, 1], 2)).toEqual([3, 1]);
  });
});

describe("groupScopedQuery", () => {
  const ownScopeFilters = [{ field: "status", operator: "eq" as const, value: "Open" }];
  const fieldFilters = [{ field: "customerId", operator: "eq" as const, value: "cus_1" }];
  const filterGroups = [
    { filters: [{ field: "proNumber", operator: "contains" as const, value: "S" }] },
  ];
  const sort: SortField[] = [{ field: "stage", direction: "asc" }];

  it("puts the host's scope, then the collapsed groups' filters, ahead of the person's filters", () => {
    const groupScope: DataTableGroupScope = {
      fieldFilters: [{ field: "stage", operator: "notin", value: [1] }],
      filterGroups: [{ filters: [{ field: "shipDate", operator: "isnull", value: null }] }],
    };
    expect(
      groupScopedQuery({
        query: "acme",
        ownScopeFilters,
        fieldFilters,
        filterGroups,
        sort,
        groupScope,
      }),
    ).toEqual({
      scopeFilters: [...ownScopeFilters, ...groupScope.fieldFilters],
      filterGroups: [...groupScope.filterGroups, ...filterGroups],
      options: {
        query: "acme",
        fieldFilters: [...ownScopeFilters, ...groupScope.fieldFilters, ...fieldFilters],
        filterGroups: [...groupScope.filterGroups, ...filterGroups],
        sort,
      },
    });
  });

  it("passes the person's filters through untouched when nothing is collapsed", () => {
    const result = groupScopedQuery({
      ownScopeFilters: [],
      fieldFilters,
      filterGroups,
      sort,
      groupScope: { fieldFilters: [], filterGroups: [] },
    });
    expect(result.scopeFilters).toEqual([]);
    expect(result.options.fieldFilters).toBe(fieldFilters);
    expect(result.options.filterGroups).toBe(filterGroups);
  });
});
