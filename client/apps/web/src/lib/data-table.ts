import type {
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
} from "@/types/data-table";
import type { SelectOption } from "@/types/fields";
import type {
  FormatRuleColor,
  TableColumnPinning,
  TableConfig,
  TableFormatRule,
} from "@/types/table-configuration";
import type { Column, ColumnDef, Row } from "@tanstack/react-table";
import type { CSSProperties } from "react";

export const FILTER_OPERATORS: Record<FilterVariant, FilterOperator[]> = {
  text: [
    "contains",
    "eq",
    "ne",
    "startswith",
    "endswith",
    "isnull",
    "isnotnull",
  ],
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
};

export const CONNECTOR_LABELS: Record<FilterConnector, string> = {
  and: "And",
  or: "Or",
};

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

export function getOperatorsForVariant(
  variant: FilterVariant,
): FilterOperator[] {
  return FILTER_OPERATORS[variant] || FILTER_OPERATORS.text;
}

export function getOperatorLabel(operator: FilterOperator): string {
  return OPERATOR_LABELS[operator] || operator;
}

export function getConnectorLabel(connector: FilterConnector): string {
  return CONNECTOR_LABELS[connector] || connector;
}

export function getDefaultOperatorForVariant(
  variant: FilterVariant,
): FilterOperator {
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

export function isValidFilterValue(
  operator: FilterOperator,
  value: unknown,
): boolean {
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
    return currentSort.map((s) =>
      s.field === field ? { ...s, direction } : s,
    );
  }

  return [...currentSort, { field, direction }];
}

export function generateGroupId(): string {
  return `group-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}

function singleFilterToFieldFilter(
  filter: SingleFilterItem,
): FieldFilter | null {
  if (!isValidFilterValue(filter.operator, filter.value)) return null;
  const value = sanitizeFilterValue(filter.value);
  if (Array.isArray(value) && value.length === 0) return null;
  return {
    field: filter.apiField,
    operator: filter.operator,
    value,
  };
}

export function convertFilterItemsToFilterGroups(
  items: FilterItem[],
): FilterGroup[] {
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

function resolveColumnFields<TData>(
  field: string,
  columns: ColumnDef<TData>[],
): {
  columnField: string;
  apiField: string;
  label: string;
  filterType: FilterVariant;
  filterOptions?: SelectOption[];
} {
  const column = columns.find(
    (c) =>
      c.meta?.apiField === field ||
      ("accessorKey" in c && c.accessorKey === field),
  );
  const columnField =
    column && "accessorKey" in column
      ? String(column.accessorKey)
      : String(column?.id ?? field);
  const apiField = column?.meta?.apiField || field;
  return {
    columnField,
    apiField,
    label: column?.meta?.label || apiField,
    filterType: (column?.meta?.filterType || "text") as FilterVariant,
    filterOptions: column?.meta?.filterOptions as SelectOption[] | undefined,
  };
}

export function initializeFilterItemsFromFilterGroups<TData>(
  filterGroups: FilterGroup[],
  columns: ColumnDef<TData>[],
): FilterItem[] {
  if (filterGroups.length === 0) return [];

  const items: FilterItem[] = [];

  for (let groupIndex = 0; groupIndex < filterGroups.length; groupIndex++) {
    const group = filterGroups[groupIndex];

    if (group.filters.length === 1) {
      const f = group.filters[0];
      const resolved = resolveColumnFields(f.field, columns);

      items.push({
        type: "filter",
        id: generateFilterId(),
        connector: "and",
        field: resolved.columnField,
        apiField: resolved.apiField,
        label: resolved.label,
        operator: f.operator,
        value: f.value,
        filterType: resolved.filterType,
        filterOptions: resolved.filterOptions,
      });
    } else {
      const groupItem: FilterGroupItem = {
        type: "group",
        id: generateGroupId(),
        connector: "and",
        items: group.filters.map((f, filterIndex) => {
          const resolved = resolveColumnFields(f.field, columns);
          return {
            type: "filter" as const,
            id: generateFilterId(),
            connector: filterIndex === 0 ? ("and" as const) : ("or" as const),
            field: resolved.columnField,
            apiField: resolved.apiField,
            label: resolved.label,
            operator: f.operator,
            value: f.value,
            filterType: resolved.filterType,
            filterOptions: resolved.filterOptions,
          };
        }),
      };
      items.push(groupItem);
    }
  }

  return items;
}

export function convertFilterItemsToFieldFilters(
  items: FilterItem[],
): FieldFilter[] | null {
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

export function columnPinOffsetVar(columnId: string, side: "left" | "right"): string {
  return `--col-${columnId.replaceAll(".", "-")}-${side}`;
}

export function pinnedCellStyle(column: Column<any, unknown>): CSSProperties | undefined {
  const pinned = column.getIsPinned();
  if (!pinned) return undefined;
  return pinned === "left"
    ? { left: `var(${columnPinOffsetVar(column.id, "left")})` }
    : { right: `var(${columnPinOffsetVar(column.id, "right")})` };
}

export function pinnedCellClass(column: Column<any, unknown>): string | undefined {
  const pinned = column.getIsPinned();
  if (!pinned) return undefined;
  const isBoundary =
    pinned === "left" ? column.getIsLastColumn("left") : column.getIsFirstColumn("right");
  if (pinned === "left") {
    return isBoundary
      ? "sticky z-10 bg-background shadow-[inset_-1px_0_0_0_var(--border)]"
      : "sticky z-10 bg-background";
  }
  return isBoundary
    ? "sticky z-10 bg-background shadow-[inset_1px_0_0_0_var(--border)]"
    : "sticky z-10 bg-background";
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
  red: "bg-red-500/10 hover:bg-red-500/15 dark:bg-red-500/15 dark:hover:bg-red-500/20",
  amber: "bg-amber-500/10 hover:bg-amber-500/15 dark:bg-amber-500/15 dark:hover:bg-amber-500/20",
  green:
    "bg-emerald-500/10 hover:bg-emerald-500/15 dark:bg-emerald-500/15 dark:hover:bg-emerald-500/20",
  blue: "bg-sky-500/10 hover:bg-sky-500/15 dark:bg-sky-500/15 dark:hover:bg-sky-500/20",
  purple:
    "bg-violet-500/10 hover:bg-violet-500/15 dark:bg-violet-500/15 dark:hover:bg-violet-500/20",
  gray: "bg-muted-foreground/10 hover:bg-muted-foreground/15",
};

export const FORMAT_RULE_COLOR_SWATCHES: Record<FormatRuleColor, string> = {
  red: "bg-red-500",
  amber: "bg-amber-500",
  green: "bg-emerald-500",
  blue: "bg-sky-500",
  purple: "bg-violet-500",
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

export type CompiledFormatRules<TData> = (row: Row<TData>) => string | undefined;

export function findColumnIdForField<TData>(
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

export function compileFormatRules<TData>(
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

export function initializeFilterItemsFromFieldFilters<TData>(
  fieldFilters: FieldFilter[],
  columns: ColumnDef<TData>[],
): FilterItem[] {
  return fieldFilters.map((f) => {
    const resolved = resolveColumnFields(f.field, columns);
    return {
      type: "filter",
      id: generateFilterId(),
      connector: "and",
      field: resolved.columnField,
      apiField: resolved.apiField,
      label: resolved.label,
      operator: f.operator,
      value: f.value,
      filterType: resolved.filterType,
      filterOptions: resolved.filterOptions,
    };
  });
}
