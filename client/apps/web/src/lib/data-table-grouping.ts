import type {
  DataTableGroup,
  DataTableGroupKey,
  DataTableGrouping,
  FieldFilter,
  SortField,
} from "@trenova/shared/types/data-table";

type GroupSortSource = Pick<DataTableGrouping<unknown>, "field" | "direction">;
type GroupScopeSource = Pick<DataTableGrouping<unknown>, "field" | "collapsedKeys">;

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
  return [
    { field: grouping.field, direction: own?.direction ?? grouping.direction ?? "asc" },
    ...sort.filter((entry) => entry.field !== grouping.field),
  ];
}

/** Collapsed groups are dropped by the server, so they never take a page's rows. */
export function groupedScopeFilters(grouping: GroupScopeSource | undefined): FieldFilter[] {
  if (!grouping || grouping.collapsedKeys.length === 0) {
    return [];
  }
  return [{ field: grouping.field, operator: "notin", value: [...grouping.collapsedKeys] }];
}

export type GroupedBodyItem<TRow> =
  | { kind: "group"; group: DataTableGroup; collapsed: boolean }
  | { kind: "row"; row: TRow; index: number };

type BuildGroupedBodyParams<TRow> = {
  rows: readonly TRow[];
  groups: readonly DataTableGroup[];
  getGroupKey: (row: TRow) => DataTableGroupKey;
  collapsedKeys: readonly DataTableGroupKey[];
  isFirstPage: boolean;
  isLastPage: boolean;
};

/**
 * Lays one page of grouped rows out as headers and rows. A header opens every
 * group the page touches, including one carried over from the previous page.
 * A collapsed group has no rows to anchor it, so it is placed by order: between
 * the visible groups it sorts between, ahead of the rows on the first page, and
 * after them on the last.
 */
export function buildGroupedBody<TRow>({
  rows,
  groups,
  getGroupKey,
  collapsedKeys,
  isFirstPage,
  isLastPage,
}: BuildGroupedBodyParams<TRow>): GroupedBodyItem<TRow>[] {
  const order = new Map<DataTableGroupKey, number>();
  groups.forEach((group, index) => order.set(group.key, index));
  const groupByKey = new Map(groups.map((group) => [group.key, group]));
  const collapsed = new Set(collapsedKeys);
  const collapsedInOrder = groups.filter((group) => collapsed.has(group.key));
  const orderOf = (key: DataTableGroupKey) => order.get(key) ?? groups.length;

  const items: GroupedBodyItem<TRow>[] = [];
  let collapsedCursor = 0;

  const emitCollapsedBefore = (limit: number) => {
    while (
      collapsedCursor < collapsedInOrder.length &&
      orderOf(collapsedInOrder[collapsedCursor].key) < limit
    ) {
      items.push({ kind: "group", group: collapsedInOrder[collapsedCursor], collapsed: true });
      collapsedCursor += 1;
    }
  };

  let currentKey: DataTableGroupKey | undefined;
  let seenRow = false;

  rows.forEach((row, index) => {
    const key = getGroupKey(row);
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
