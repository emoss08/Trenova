import type {
  DataTableFilterField,
  FieldFilter,
  FilterConnector,
  FilterGroup,
  FilterGroupItem,
  FilterItem,
  FilterOperator,
  FilterVariant,
  SingleFilterItem,
  SortDirection,
  SortField,
  Column,
  ColumnDef,
  Header,
  Row,
} from "@trenova/shared/types/data-table";
import type { EmptyTableColumn } from "@trenova/shared/components/ui/empty-table";
import { toTitleCase } from "@trenova/shared/lib/utils";
import type { SelectOption } from "@trenova/shared/types/fields";
import type {
  FormatRuleColor,
  TableColumnPinning,
  TableConfig,
  TableFormatRule,
} from "@/types/table-configuration";
import type { CellData, ColumnPinningState, RowData, SortingState } from "@tanstack/react-table";
import type { CSSProperties } from "react";
import { stableStringify } from "@/lib/stable-stringify";
import { defineLabels, translateLabel } from "@trenova/shared/i18n/labels";

export type UrlFilterState = {
  fieldFilters: FieldFilter[];
  filterGroups: FilterGroup[];
};

export function filterItemsToUrlFilterState(items: FilterItem[]): UrlFilterState {
  const fieldFilters: FieldFilter[] = [];
  const filterGroups: FilterGroup[] = [];
  for (const item of items) {
    if (item.type === "filter") {
      fieldFilters.push({ field: item.apiField, operator: item.operator, value: item.value });
    } else {
      filterGroups.push({
        filters: item.items.map((i) => ({
          field: i.apiField,
          operator: i.operator,
          value: i.value,
        })),
      });
    }
  }
  return { fieldFilters, filterGroups };
}

export function buildFilterItemsFromUrlState(
  state: UrlFilterState,
  fields: readonly FilterableField[],
): FilterItem[] {
  return [
    ...initializeFilterItemsFromFieldFilters(state.fieldFilters, fields),
    ...initializeFilterItemsFromFilterGroups(
      state.filterGroups.filter((g) => g.filters?.length > 0),
      fields,
    ),
  ];
}

export function serializeUrlFilterState(state: UrlFilterState): string {
  return stableStringify({
    fieldFilters: state.fieldFilters,
    filterGroups: state.filterGroups,
  });
}

export const FILTER_OPERATORS: Record<FilterVariant, FilterOperator[]> = {
  text: ["contains", "eq", "ne", "startswith", "endswith", "isnull", "isnotnull"],
  number: ["eq", "ne", "gt", "gte", "lt", "lte", "isnull", "isnotnull"],
  date: [
    "eq",
    "gt",
    "gte",
    "lt",
    "lte",
    "lastndays",
    "nextndays",
    "today",
    "yesterday",
    "tomorrow",
    "daterange",
  ],
  select: ["eq", "ne", "in", "notin"],
  boolean: ["eq"],
  record: ["in", "notin", "eq", "ne", "isnull", "isnotnull"],
};

export const CONNECTOR_LABELS: Record<FilterConnector, string> = defineLabels({
  and: "And",
  or: "Or",
});

export const OPERATOR_LABELS: Record<FilterOperator, string> = {
  eq: "equals",
  ne: "not equals",
  gt: "greater than",
  gte: "greater than or equal",
  lt: "less than",
  lte: "less than or equal",
  contains: "contains",
  startswith: "starts with",
  endswith: "ends with",
  ilike: "matches",
  in: "is any of",
  notin: "is none of",
  isnull: "is empty",
  isnotnull: "is not empty",
  daterange: "is between",
  lastndays: "in last N days",
  nextndays: "in next N days",
  today: "is today",
  yesterday: "is yesterday",
  tomorrow: "is tomorrow",
};

export const OPERATORS_WITHOUT_VALUE: FilterOperator[] = [
  "isnull",
  "isnotnull",
  "today",
  "yesterday",
  "tomorrow",
];

export function getOperatorsForVariant(variant: FilterVariant): FilterOperator[] {
  return FILTER_OPERATORS[variant] || FILTER_OPERATORS.text;
}

export function getOperatorLabel(operator: FilterOperator): string {
  return OPERATOR_LABELS[operator] || operator;
}

export function getConnectorLabel(connector: FilterConnector): string {
  return CONNECTOR_LABELS[connector] || connector;
}

export function getDefaultOperatorForVariant(variant: FilterVariant): FilterOperator {
  switch (variant) {
    case "text":
      return "contains";
    case "number":
      return "eq";
    case "date":
      return "eq";
    case "select":
      return "eq";
    case "boolean":
      return "eq";
    case "record":
      return "in";
    default:
      return "eq";
  }
}

export function operatorRequiresValue(operator: FilterOperator): boolean {
  return !OPERATORS_WITHOUT_VALUE.includes(operator);
}

export function generateFilterId(): string {
  return `filter-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}

export function sanitizeFilterValue(value: unknown): unknown {
  if (typeof value === "string") return value.trim();
  if (Array.isArray(value)) {
    return value.filter((v) => v !== null && v !== undefined);
  }
  return value;
}

export function isValidFilterValue(operator: FilterOperator, value: unknown): boolean {
  if (!operatorRequiresValue(operator)) {
    return true;
  }

  if (value === null || value === undefined || value === "") {
    return false;
  }

  if (Array.isArray(value) && value.length === 0) {
    return false;
  }

  return true;
}

export function updateSortField(
  currentSort: SortField[],
  field: string,
  direction: SortDirection | null,
): SortField[] {
  if (direction === null) {
    return currentSort.filter((s) => s.field !== field);
  }

  const existing = currentSort.find((s) => s.field === field);
  if (existing) {
    return currentSort.map((s) => (s.field === field ? { ...s, direction } : s));
  }

  return [...currentSort, { field, direction }];
}

export function generateGroupId(): string {
  return `group-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}

function singleFilterToFieldFilter(filter: SingleFilterItem): FieldFilter | null {
  if (!isValidFilterValue(filter.operator, filter.value)) return null;
  const value = sanitizeFilterValue(filter.value);
  if (Array.isArray(value) && value.length === 0) return null;
  return {
    field: filter.apiField,
    operator: filter.operator,
    value,
  };
}

export function convertFilterItemsToFilterGroups(items: FilterItem[]): FilterGroup[] {
  if (items.length === 0) return [];

  const groups: FilterGroup[] = [];
  let currentGroup: FieldFilter[] = [];

  for (let i = 0; i < items.length; i++) {
    const item = items[i];

    if (item.type === "filter") {
      const fieldFilter = singleFilterToFieldFilter(item);
      if (!fieldFilter) continue;

      if (i === 0 || item.connector === "or") {
        currentGroup.push(fieldFilter);
      } else {
        if (currentGroup.length > 0) {
          groups.push({ filters: currentGroup });
        }
        currentGroup = [fieldFilter];
      }
    } else if (item.type === "group") {
      if (currentGroup.length > 0) {
        groups.push({ filters: currentGroup });
        currentGroup = [];
      }

      const groupFilters: FieldFilter[] = [];
      for (const subFilter of item.items) {
        const fieldFilter = singleFilterToFieldFilter(subFilter);
        if (fieldFilter) {
          groupFilters.push(fieldFilter);
        }
      }
      if (groupFilters.length > 0) {
        groups.push({ filters: groupFilters });
      }
    }
  }

  if (currentGroup.length > 0) {
    groups.push({ filters: currentGroup });
  }

  return groups;
}

/**
 * One way a table can be filtered, resolved from a column or from the table's own
 * filter fields. `id` names it in the builder; `apiField` is what the server reads.
 */
export type FilterableField = {
  id: string;
  apiField: string;
  label: string;
  filterType: FilterVariant;
  filterOptions?: SelectOption[];
  filterRecord?: string;
  defaultOperator: FilterOperator;
};

function toFilterableField(id: string, field: DataTableFilterField): FilterableField {
  return {
    id,
    apiField: field.apiField,
    label: field.label,
    filterType: field.filterType,
    filterOptions: field.filterOptions,
    filterRecord: field.filterRecord,
    defaultOperator:
      field.defaultFilterOperator ?? getDefaultOperatorForVariant(field.filterType),
  };
}

/**
 * Every way a table can be filtered: each filterable column, the extra filters a
 * column offers, then the table's own filter fields. A field named twice is offered
 * once, by the first that names it.
 */
export function getFilterableFields<TData extends RowData>(
  columns: readonly ColumnDef<TData>[],
  tableFields: readonly DataTableFilterField[] = [],
): FilterableField[] {
  const fields: FilterableField[] = [];
  const seen = new Set<string>();
  const add = (id: string, field: DataTableFilterField) => {
    if (seen.has(field.apiField)) return;
    seen.add(field.apiField);
    fields.push(toFilterableField(id, field));
  };

  for (const column of columns) {
    const meta = column.meta;
    if (!meta) continue;
    const columnId = String("accessorKey" in column ? column.accessorKey : column.id);
    if (meta.filterable === true && meta.apiField) {
      add(columnId, {
        apiField: meta.apiField,
        label: fieldLabel({ id: columnId, columnDef: column }),
        filterType: meta.filterType || "text",
        filterOptions: meta.filterOptions,
        filterRecord: meta.filterRecord,
        defaultFilterOperator: meta.defaultFilterOperator,
      });
    }
    for (const extra of meta.extraFilters ?? []) {
      add(`${columnId}:${extra.apiField}`, extra);
    }
  }
  for (const field of tableFields) {
    add(`field:${field.apiField}`, field);
  }

  return fields;
}

function resolveFilterableField(
  apiField: string,
  fields: readonly FilterableField[],
): FilterableField {
  return (
    fields.find((field) => field.apiField === apiField) ?? {
      id: apiField,
      apiField,
      label: translateLabel(toTitleCase(apiField)),
      filterType: "text",
      defaultOperator: "contains",
    }
  );
}

export function filterItemFromField(
  field: FilterableField,
  filter: Pick<FieldFilter, "operator" | "value">,
  connector: FilterConnector,
): SingleFilterItem {
  return {
    type: "filter",
    id: generateFilterId(),
    connector,
    field: field.id,
    apiField: field.apiField,
    label: field.label,
    operator: filter.operator,
    value: filter.value,
    filterType: field.filterType,
    filterOptions: field.filterOptions,
    filterRecord: field.filterRecord,
  };
}

export function initializeFilterItemsFromFilterGroups(
  filterGroups: FilterGroup[],
  fields: readonly FilterableField[],
): FilterItem[] {
  const items: FilterItem[] = [];

  for (const group of filterGroups) {
    if (group.filters.length === 1) {
      const filter = group.filters[0];
      items.push(filterItemFromField(resolveFilterableField(filter.field, fields), filter, "and"));
      continue;
    }

    const groupItem: FilterGroupItem = {
      type: "group",
      id: generateGroupId(),
      connector: "and",
      items: group.filters.map((filter, filterIndex) =>
        filterItemFromField(
          resolveFilterableField(filter.field, fields),
          filter,
          filterIndex === 0 ? "and" : "or",
        ),
      ),
    };
    items.push(groupItem);
  }

  return items;
}

export function convertFilterItemsToFieldFilters(items: FilterItem[]): FieldFilter[] | null {
  if (items.some((item) => item.type === "group")) return null;
  return (items as SingleFilterItem[]).map((item) => ({
    field: item.apiField,
    operator: item.operator,
    value: item.value,
  }));
}

export function columnSizeVar(columnId: string): string {
  return `--col-${columnId.replaceAll(".", "-")}-size`;
}

export function columnPinOffsetVar(columnId: string, side: "start" | "end"): string {
  return `--col-${columnId.replaceAll(".", "-")}-${side}`;
}

export type ColumnLayout = {
  vars: Record<string, string>;
  totalSize: number;
};

export function columnLayout<TData extends RowData>(headers: Header<TData>[]): ColumnLayout {
  const vars: Record<string, string> = {};
  let totalSize = 0;
  for (const header of headers) {
    const { column } = header;
    const size = header.getSize();
    vars[columnSizeVar(column.id)] = `${size}px`;
    totalSize += size;

    const pinned = column.getIsPinned();
    if (pinned === "start") {
      vars[columnPinOffsetVar(column.id, "start")] = `${column.getStart("start")}px`;
    } else if (pinned === "end") {
      vars[columnPinOffsetVar(column.id, "end")] = `${column.getAfter("end")}px`;
    }
  }
  return { vars, totalSize };
}

export function toColumnPinningState(
  pinning: TableColumnPinning | undefined | null,
): ColumnPinningState {
  return { start: pinning?.left ?? [], end: pinning?.right ?? [] };
}

export function fromColumnPinningState(
  pinning: ColumnPinningState | undefined | null,
): TableColumnPinning {
  return { left: pinning?.start ?? [], right: pinning?.end ?? [] };
}

export function withRequiredPinning(
  pinning: TableColumnPinning,
  required: TableColumnPinning | undefined,
): TableColumnPinning {
  if (!required) return pinning;
  const requiredIds = new Set([...required.left, ...required.right]);
  const unrequired = (ids: string[]) => ids.filter((id) => !requiredIds.has(id));
  return {
    left: [...required.left, ...unrequired(pinning.left)],
    right: [...unrequired(pinning.right), ...required.right],
  };
}

export function pinnedCellStyle<TData extends RowData>(
  column: Column<TData>,
): CSSProperties | undefined {
  return pinnedSideStyle(column.id, column.getIsPinned());
}

/** The offset a cell pinned to `side` takes, for a caller that already knows the side. */
export function pinnedSideStyle(
  columnId: string,
  side: false | "start" | "end",
): CSSProperties | undefined {
  if (!side) return undefined;
  return side === "start"
    ? { insetInlineStart: `var(${columnPinOffsetVar(columnId, "start")})` }
    : { insetInlineEnd: `var(${columnPinOffsetVar(columnId, "end")})` };
}

/** The widest a column grows when fitted to its content; wider text is truncated. */
export const MAX_FITTED_COLUMN_WIDTH = 640;

/**
 * How wide each column needs to be to show its widest header or cell on the page
 * without truncating. Each column's width variable is let go to `max-content` for a
 * single layout pass, every head is read, and the variables are put back, so
 * fitting any number of columns costs one forced layout.
 */
export function measureColumnFits(
  tableElement: HTMLTableElement,
  columnIds: readonly string[],
): Record<string, number> {
  const { style } = tableElement;
  const saved = new Map<string, string>();
  for (const id of columnIds) {
    const name = columnSizeVar(id);
    saved.set(name, style.getPropertyValue(name));
    style.setProperty(name, "max-content");
  }

  const fits: Record<string, number> = {};
  try {
    for (const id of columnIds) {
      const head = tableElement.querySelector<HTMLElement>(
        `th[data-column-id="${CSS.escape(id)}"]`,
      );
      const width = head?.getBoundingClientRect().width ?? 0;
      if (width > 0) fits[id] = Math.ceil(width);
    }
  } finally {
    for (const [name, value] of saved) {
      if (value) style.setProperty(name, value);
      else style.removeProperty(name);
    }
  }
  return fits;
}

/** A fitted width kept inside what the column allows and what a table can show. */
export function clampFittedWidth<TData extends RowData>(
  column: Column<TData>,
  width: number,
): number {
  const { minSize, maxSize } = column.columnDef;
  const upper = Math.min(maxSize ?? MAX_FITTED_COLUMN_WIDTH, MAX_FITTED_COLUMN_WIDTH);
  return Math.max(minSize ?? 0, Math.min(upper, width));
}

/** Whether a person can change a column's width at all. */
export function isColumnResizable<TData extends RowData, TValue extends CellData = CellData>(
  column: Column<TData, TValue>,
): boolean {
  const { minSize, maxSize } = column.columnDef;
  return column.getCanResize() && !(minSize !== undefined && minSize === maxSize);
}

/** A body cell's width and, when its column is pinned, its offset: what lines a cell up with its header. */
export function columnCellStyle<TData extends RowData>(column: Column<TData>): CSSProperties {
  return {
    width: `var(${columnSizeVar(column.id)})`,
    maxWidth: `var(${columnSizeVar(column.id)})`,
    ...pinnedCellStyle(column),
  };
}

export function pinnedCellClass<TData extends RowData>(column: Column<TData>): string | undefined {
  const pinned = column.getIsPinned();
  if (!pinned) return undefined;
  const isBoundary =
    pinned === "start" ? column.getIsLastColumn("start") : column.getIsFirstColumn("end");
  if (!isBoundary) return "sticky z-10 bg-background";
  return pinned === "start"
    ? "sticky z-10 bg-background shadow-[inset_-1px_0_0_0_var(--border)]"
    : "sticky z-10 bg-background shadow-[inset_1px_0_0_0_var(--border)]";
}

function areVisibilityMapsEqual(
  a: Record<string, boolean> | undefined,
  b: Record<string, boolean> | undefined,
): boolean {
  const keys = new Set([...Object.keys(a ?? {}), ...Object.keys(b ?? {})]);
  for (const key of keys) {
    if ((a?.[key] ?? true) !== (b?.[key] ?? true)) return false;
  }
  return true;
}

function areSizingMapsEqual(
  a: Record<string, number> | undefined,
  b: Record<string, number> | undefined,
): boolean {
  const keys = new Set([...Object.keys(a ?? {}), ...Object.keys(b ?? {})]);
  for (const key of keys) {
    if (a?.[key] !== b?.[key]) return false;
  }
  return true;
}

const EMPTY_PINNING: TableColumnPinning = { left: [], right: [] };

export function isTableConfigEqual(a: TableConfig, b: TableConfig): boolean {
  return (
    a.pageSize === b.pageSize &&
    (a.density ?? "comfortable") === (b.density ?? "comfortable") &&
    JSON.stringify(a.fieldFilters ?? []) === JSON.stringify(b.fieldFilters ?? []) &&
    JSON.stringify(a.filterGroups ?? []) === JSON.stringify(b.filterGroups ?? []) &&
    JSON.stringify(a.sort ?? []) === JSON.stringify(b.sort ?? []) &&
    JSON.stringify(a.columnOrder ?? []) === JSON.stringify(b.columnOrder ?? []) &&
    JSON.stringify(a.columnPinning ?? EMPTY_PINNING) ===
      JSON.stringify(b.columnPinning ?? EMPTY_PINNING) &&
    JSON.stringify(a.formatRules ?? []) === JSON.stringify(b.formatRules ?? []) &&
    areVisibilityMapsEqual(a.columnVisibility, b.columnVisibility) &&
    areSizingMapsEqual(a.columnSizing, b.columnSizing)
  );
}

export const FORMAT_RULE_COLOR_CLASSES: Record<FormatRuleColor, string> = {
  red: "bg-danger-subtle hover:bg-danger-subtle dark:hover:bg-danger-subtle",
  amber: "bg-warning-subtle hover:bg-warning-subtle dark:hover:bg-warning-subtle",
  green: "bg-success-subtle hover:bg-success-subtle dark:hover:bg-success-subtle",
  blue: "bg-accent-sky/10 hover:bg-accent-sky/15 dark:hover:bg-accent-sky/20",
  purple: "bg-accent-violet/10 hover:bg-accent-violet/15 dark:hover:bg-accent-violet/20",
  gray: "bg-muted-foreground/10 hover:bg-muted-foreground/15",
};

export const FORMAT_RULE_COLOR_SWATCHES: Record<FormatRuleColor, string> = {
  red: "bg-danger",
  amber: "bg-warning",
  green: "bg-success",
  blue: "bg-accent-sky",
  purple: "bg-accent-violet",
  gray: "bg-muted-foreground",
};

function isEmptyRuleValue(value: unknown): boolean {
  return value === null || value === undefined || value === "";
}

export function stringifyUnknown(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "bigint" || typeof value === "boolean") {
    return value.toString();
  }
  return JSON.stringify(value) ?? "";
}

function formatRulePredicate(rule: TableFormatRule): (value: unknown) => boolean {
  const target = rule.value;

  switch (rule.operator) {
    case "isnull":
      return isEmptyRuleValue;
    case "isnotnull":
      return (value) => !isEmptyRuleValue(value);
    case "contains": {
      const needle = stringifyUnknown(target).toLowerCase();
      return (value) =>
        !isEmptyRuleValue(value) && stringifyUnknown(value).toLowerCase().includes(needle);
    }
    case "eq":
    case "ne": {
      const wantEqual = rule.operator === "eq";
      const targetString = stringifyUnknown(target).toLowerCase();
      return (value) => {
        if (isEmptyRuleValue(value)) return !wantEqual;
        const equal =
          typeof value === "number" && typeof target === "number"
            ? value === target
            : stringifyUnknown(value).toLowerCase() === targetString;
        return equal === wantEqual;
      };
    }
    case "gt":
    case "gte":
    case "lt":
    case "lte": {
      const targetNumber = Number(target);
      if (Number.isNaN(targetNumber)) return () => false;
      return (value) => {
        const valueNumber = Number(value);
        if (isEmptyRuleValue(value) || Number.isNaN(valueNumber)) return false;
        switch (rule.operator) {
          case "gt":
            return valueNumber > targetNumber;
          case "gte":
            return valueNumber >= targetNumber;
          case "lt":
            return valueNumber < targetNumber;
          default:
            return valueNumber <= targetNumber;
        }
      };
    }
    default:
      return () => false;
  }
}

export type CompiledFormatRules<TData extends RowData> = (row: Row<TData>) => string | undefined;

export function findColumnIdForField<TData extends RowData>(
  field: string,
  leafColumns: Column<TData, unknown>[],
): string | null {
  const column = leafColumns.find((col) => {
    const def = col.columnDef;
    return (
      def.meta?.apiField === field ||
      ("accessorKey" in def && def.accessorKey === field) ||
      col.id === field
    );
  });
  return column?.id ?? null;
}

export function compileFormatRules<TData extends RowData>(
  rules: TableFormatRule[],
  leafColumns: Column<TData, unknown>[],
): CompiledFormatRules<TData> | null {
  if (rules.length === 0) return null;

  const compiled: {
    columnId: string;
    predicate: (value: unknown) => boolean;
    className: string;
  }[] = [];

  for (const rule of rules) {
    const columnId = findColumnIdForField(rule.field, leafColumns);
    if (!columnId) continue;
    compiled.push({
      columnId,
      predicate: formatRulePredicate(rule),
      className: FORMAT_RULE_COLOR_CLASSES[rule.color],
    });
  }

  if (compiled.length === 0) return null;

  return (row) => {
    for (const rule of compiled) {
      if (rule.predicate(row.getValue(rule.columnId))) {
        return rule.className;
      }
    }
    return undefined;
  };
}

export function initializeFilterItemsFromFieldFilters(
  fieldFilters: FieldFilter[],
  fields: readonly FilterableField[],
): FilterItem[] {
  return fieldFilters.map((filter) =>
    filterItemFromField(resolveFilterableField(filter.field, fields), filter, "and"),
  );
}

/**
 * The name a column goes by when it is offered as a field to filter or format by.
 * Its own label comes first, as it says what is matched ("Customer name") where the
 * header only names the column ("Customer"); the header is the fallback.
 */
export function fieldLabel(column: LabelledColumn): string {
  const label = column.columnDef.meta?.label;
  return label ? translateLabel(label) : columnHeaderLabel(column);
}

type LabelledColumn = {
  id: string;
  columnDef: { header?: unknown; meta?: { label?: string; filterType?: string } };
};

/**
 * The name a column shows in its header cell: a string header as written,
 * otherwise the label its meta carries, otherwise its id read as words.
 */
export function columnHeaderLabel(column: LabelledColumn): string {
  const { header, meta } = column.columnDef;
  if (typeof header === "string") return translateLabel(header);
  return translateLabel(meta?.label || toTitleCase(column.id));
}

/**
 * The visible columns of a table as the headings of an empty sketch. The
 * selection column is chrome, not data, so it is left out; numeric columns
 * keep their right alignment.
 */
export function emptyTableColumns(columns: readonly LabelledColumn[]): EmptyTableColumn[] {
  return columns
    .filter((column) => column.id !== "select")
    .map((column) => ({
      label: columnHeaderLabel(column),
      numeric: column.columnDef.meta?.filterType === "number",
    }));
}

/**
 * Which way a column is sorted, read from the sorting state itself. A header drawn
 * from a header group the table caches across sorts reads it this way, so its
 * arrow follows the sort under the React Compiler.
 */
export function sortDirectionOf(sorting: SortingState, columnId: string): false | "asc" | "desc" {
  const entry = sorting.find((sort) => sort.id === columnId);
  if (!entry) return false;
  return entry.desc ? "desc" : "asc";
}

/**
 * Whether a row was changed after a moment, read from the `updatedAt` (Unix seconds)
 * its query carries. A row without one is never marked.
 */
export function isChangedSince(row: unknown, since: number): boolean {
  if (since <= 0 || row === null || typeof row !== "object") return false;
  const updatedAt = (row as { updatedAt?: unknown }).updatedAt;
  return typeof updatedAt === "number" && updatedAt > since;
}

/**
 * The field a column's header counts and filters values of, when the table can count
 * it: the column's `facetField`, else its `apiField` for a select or record filter.
 */
export function columnFacetField<TData extends RowData>(
  column: Column<TData>,
  fields: readonly FilterableField[],
  facetable: ReadonlySet<string>,
): FilterableField | null {
  const meta = column.columnDef.meta;
  if (!meta || meta.facetField === false) return null;
  const apiField =
    meta.facetField ??
    (meta.filterType === "select" || meta.filterType === "record" ? meta.apiField : undefined);
  if (!apiField || !facetable.has(apiField)) return null;

  return (
    fields.find((field) => field.apiField === apiField) ??
    toFilterableField(`${column.id}:${apiField}`, {
      apiField,
      label: fieldLabel(column),
      filterType: meta.filterType ?? "select",
      filterOptions: meta.filterOptions,
      filterRecord: meta.filterRecord,
    })
  );
}
