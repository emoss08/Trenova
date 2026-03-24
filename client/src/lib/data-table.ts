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
import type { ColumnDef } from "@tanstack/react-table";

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
