import type {
  DataTableGroup,
  DataTableGroupKey,
  DataTableGroupScope,
  DataTableGrouping,
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

export type GroupedBodyItem<TRow> =
  | { kind: "group"; group: DataTableGroup; collapsed: boolean }
  | { kind: "row"; row: TRow; index: number };

type BuildGroupedBodyParams<TRow> = {
  rows: readonly TRow[];
  groups: readonly DataTableGroup[];
  getGroupKey: (row: TRow) => DataTableGroupKey;
  collapsedKeys: readonly DataTableGroupKey[];
  loadingKeys?: readonly DataTableGroupKey[];
  isFirstPage: boolean;
  isLastPage: boolean;
};

/**
 * Lays one page of grouped rows out as headers and rows. A header opens every
 * group the page touches, including one carried over from the previous page.
 * A collapsed group has no rows to anchor it, so it is placed by order: between
 * the visible groups it sorts between, ahead of the rows on the first page, and
 * after them on the last. While the next page loads, rows of a group that has
 * just collapsed are left out, and a group that has just opened keeps its
 * header in the same place until its rows arrive.
 */
export function buildGroupedBody<TRow>({
  rows,
  groups,
  getGroupKey,
  collapsedKeys,
  loadingKeys = [],
  isFirstPage,
  isLastPage,
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

  const emitCollapsedBefore = (limit: number) => {
    while (
      collapsedCursor < collapsedInOrder.length &&
      orderOf(collapsedInOrder[collapsedCursor].key) < limit
    ) {
      const group = collapsedInOrder[collapsedCursor];
      items.push({ kind: "group", group, collapsed: collapsed.has(group.key) });
      collapsedCursor += 1;
    }
  };

  let currentKey: DataTableGroupKey | undefined;
  let seenRow = false;

  rows.forEach((row, index) => {
    const key = getGroupKey(row);
    if (collapsed.has(key)) {
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
  });

  if (isLastPage || !seenRow) {
    if (seenRow || isFirstPage) {
      emitCollapsedBefore(Number.POSITIVE_INFINITY);
    }
  }

  return items;
}
