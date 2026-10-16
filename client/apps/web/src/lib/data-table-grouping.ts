import type {
  DataTableGroup,
  DataTableGroupKey,
  DataTableGroupScope,
  DataTableGrouping,
  DataTableQueryOptions,
  FieldFilter,
  FilterGroup,
  SortField,
} from "@trenova/shared/types/data-table";

type GroupSortSource = Pick<DataTableGrouping<unknown>, "field" | "direction" | "tieBreakers">;
type GroupScopeSource = Pick<
  DataTableGrouping<unknown>,
  "field" | "collapsedKeys" | "collapsedScope"
>;

const NO_GROUP_SCOPE: DataTableGroupScope = { fieldFilters: [], filterGroups: [] };

/**
 * The sort a grouped table sends: the group field first, so the server pages
 * through the rows in group order, then the person's own sort within a group.
 * A sort the person put on the group field itself sets its direction.
 */
export function groupedSort(grouping: GroupSortSource | undefined, sort: SortField[]): SortField[] {
  if (!grouping) {
    return sort;
  }
  const own = sort.find((entry) => entry.field === grouping.field);
  const tieBreakers = grouping.tieBreakers ?? [];
  const leading = new Set([grouping.field, ...tieBreakers.map((entry) => entry.field)]);
  return [
    { field: grouping.field, direction: own?.direction ?? grouping.direction ?? "asc" },
    ...tieBreakers,
    ...sort.filter((entry) => !leading.has(entry.field)),
  ];
}

/** Collapsed groups are dropped by the server, so they never take a page's rows. */
export function groupedScope(grouping: GroupScopeSource | undefined): DataTableGroupScope {
  if (!grouping || grouping.collapsedKeys.length === 0) {
    return NO_GROUP_SCOPE;
  }
  if (grouping.collapsedScope) {
    return grouping.collapsedScope(grouping.collapsedKeys);
  }
  return {
    fieldFilters: [
      { field: grouping.field, operator: "notin", value: [...grouping.collapsedKeys] },
    ],
    filterGroups: [],
  };
}

/** The collapsed keys after a group's header is toggled, in the order the host keeps them. */
export function toggleGroupKey(
  keys: readonly DataTableGroupKey[],
  key: DataTableGroupKey,
): DataTableGroupKey[] {
  return keys.includes(key) ? keys.filter((entry) => entry !== key) : [...keys, key];
}

type GroupScopedQuerySource = {
  query?: string;
  ownScopeFilters: FieldFilter[];
  fieldFilters: FieldFilter[];
  filterGroups: FilterGroup[];
  sort: SortField[];
  groupScope: DataTableGroupScope;
};

export type GroupScopedQuery = {
  /** The host's own scope plus the filters that drop collapsed groups. */
  scopeFilters: FieldFilter[];
  filterGroups: FilterGroup[];
  options: Pick<DataTableQueryOptions, "query"> &
    Required<Pick<DataTableQueryOptions, "fieldFilters" | "filterGroups" | "sort">>;
};

/**
 * The page request with collapsed groups dropped. The table's own page and a
 * page warmed for a toggle that has not happened yet are built here, so the
 * two land on the same cache entry.
 */
export function groupScopedQuery({
  query,
  ownScopeFilters,
  fieldFilters,
  filterGroups,
  sort,
  groupScope,
}: GroupScopedQuerySource): GroupScopedQuery {
  const scopeFilters =
    groupScope.fieldFilters.length > 0
      ? [...ownScopeFilters, ...groupScope.fieldFilters]
      : ownScopeFilters;
  const scopedFilterGroups =
    groupScope.filterGroups.length > 0
      ? [...groupScope.filterGroups, ...filterGroups]
      : filterGroups;
  return {
    scopeFilters,
    filterGroups: scopedFilterGroups,
    options: {
      query,
      fieldFilters: scopeFilters.length > 0 ? [...scopeFilters, ...fieldFilters] : fieldFilters,
      filterGroups: scopedFilterGroups,
      sort,
    },
  };
}

export type GroupedBodyItem<TRow> =
  | { kind: "group"; group: DataTableGroup; collapsed: boolean }
  | { kind: "row"; row: TRow; index: number }
  | { kind: "pending"; key: string; count: number };

type BuildGroupedBodyParams<TRow> = {
  rows: readonly TRow[];
  groups: readonly DataTableGroup[];
  getGroupKey: (row: TRow) => DataTableGroupKey;
  collapsedKeys: readonly DataTableGroupKey[];
  loadingKeys?: readonly DataTableGroupKey[];
  isFirstPage: boolean;
  isLastPage: boolean;
  /** Rows a page holds, so a pending group never promises more than fit on it. */
  pageSize: number;
};

/**
 * Lays one page of grouped rows out as headers and rows. A header opens every
 * group the page touches, including one carried over from the previous page.
 * A collapsed group has no rows to anchor it, so it is placed by order: between
 * the visible groups it sorts between, ahead of the rows on the first page, and
 * after them on the last. While the next page loads, rows of a group that has
 * just collapsed are left out, and a group that has just opened keeps its
 * header in the same place until its rows arrive. Rows that are on their way
 * are held open as `pending` items: under a group that has just opened, as
 * many as the group has and the page has room for, and at the foot of a page
 * that is not the last, as many as a just-collapsed group gave up, since the
 * next groups' rows move up to fill them.
 */
export function buildGroupedBody<TRow>({
  rows,
  groups,
  getGroupKey,
  collapsedKeys,
  loadingKeys = [],
  isFirstPage,
  isLastPage,
  pageSize,
}: BuildGroupedBodyParams<TRow>): GroupedBodyItem<TRow>[] {
  const order = new Map<DataTableGroupKey, number>();
  groups.forEach((group, index) => order.set(group.key, index));
  const groupByKey = new Map(groups.map((group) => [group.key, group]));
  const collapsed = new Set(collapsedKeys);
  const keysWithRows = new Set(rows.map(getGroupKey));
  const loading = new Set(
    loadingKeys.filter((key) => !collapsed.has(key) && !keysWithRows.has(key)),
  );
  const collapsedInOrder = groups.filter(
    (group) => collapsed.has(group.key) || loading.has(group.key),
  );
  const orderOf = (key: DataTableGroupKey) => order.get(key) ?? groups.length;

  const items: GroupedBodyItem<TRow>[] = [];
  let collapsedCursor = 0;
  let shownRows = 0;
  let foldedRows = 0;

  const emitCollapsedBefore = (limit: number) => {
    while (
      collapsedCursor < collapsedInOrder.length &&
      orderOf(collapsedInOrder[collapsedCursor].key) < limit
    ) {
      const group = collapsedInOrder[collapsedCursor];
      items.push({ kind: "group", group, collapsed: collapsed.has(group.key) });
      if (loading.has(group.key)) {
        const count = Math.min(group.count ?? pageSize, pageSize - shownRows);
        if (count > 0) items.push({ kind: "pending", key: `group:${group.key}`, count });
      }
      collapsedCursor += 1;
    }
  };

  let currentKey: DataTableGroupKey | undefined;
  let seenRow = false;

  rows.forEach((row, index) => {
    const key = getGroupKey(row);
    if (collapsed.has(key)) {
      foldedRows += 1;
      return;
    }
    if (!seenRow || key !== currentKey) {
      const position = orderOf(key);
      if (seenRow || isFirstPage) {
        emitCollapsedBefore(position);
      } else {
        while (
          collapsedCursor < collapsedInOrder.length &&
          orderOf(collapsedInOrder[collapsedCursor].key) < position
        ) {
          collapsedCursor += 1;
        }
      }
      items.push({
        kind: "group",
        group: groupByKey.get(key) ?? { key, label: String(key) },
        collapsed: false,
      });
      currentKey = key;
      seenRow = true;
    }
    items.push({ kind: "row", row, index });
    shownRows += 1;
  });

  if (isLastPage || !seenRow) {
    if (seenRow || isFirstPage) {
      emitCollapsedBefore(Number.POSITIVE_INFINITY);
    }
  }
  if (!isLastPage && foldedRows > 0) {
    items.push({ kind: "pending", key: "backfill", count: foldedRows });
  }

  return items;
}
